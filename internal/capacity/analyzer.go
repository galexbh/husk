package capacity

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/galexbh/husk/internal/config"
	"github.com/galexbh/husk/internal/huskerr"
	"github.com/galexbh/husk/internal/k8sclient"
	"github.com/galexbh/husk/internal/model"
	"github.com/galexbh/husk/internal/promclient"
)

// Las etiquetas estándar (y su equivalente legado) que declaran la zona de
// disponibilidad de un nodo.
const (
	zoneLabel              = "topology.kubernetes.io/zone"
	failureDomainZoneLabel = "failure-domain.beta.kubernetes.io/zone"
)

// Analyzer produce un model.CapacityReport combinando Nodes y Pods con el
// consumo histórico observado en Prometheus (opcional: prom puede ser nil
// si Thanos Querier no está disponible, degradando con gracia).
type Analyzer struct {
	client                   *k8sclient.Client
	prom                     *promclient.Client
	lookback                 string
	headroomThresholdPercent float64
}

// New construye un Analyzer a partir de la configuración de
// capacity.{headroom_threshold_percent,lookback}. prom puede ser nil.
func New(client *k8sclient.Client, prom *promclient.Client, cfg config.CapacityConfig) *Analyzer {
	return &Analyzer{
		client:                   client,
		prom:                     prom,
		lookback:                 cfg.Lookback,
		headroomThresholdPercent: cfg.HeadroomThresholdPercent,
	}
}

// Analyze recolecta Nodes y Pods de todo el cluster (sin filtro de
// namespace: el headroom debe reflejar toda la carga real, incluidos los
// namespaces de sistema) y produce el reporte de capacity.
func (a *Analyzer) Analyze(ctx context.Context) (*model.CapacityReport, error) {
	nodes, err := a.client.Kubernetes.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, huskerr.New("no se pudo listar Nodes", "verifica el permiso de lectura sobre nodes", err)
	}

	pods, err := a.client.Kubernetes.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, huskerr.New("no se pudo listar Pods", "verifica el permiso de lectura sobre pods", err)
	}

	reqCPU, reqMem, podCount := sumRequestsByNode(pods.Items)

	report := &model.CapacityReport{
		HeadroomThresholdPercent: a.headroomThresholdPercent,
		HasMetrics:               a.prom != nil,
		GeneratedAt:              time.Now(),
	}

	for _, n := range nodes.Items {
		nc := a.buildNodeCapacity(ctx, n, reqCPU[n.Name], reqMem[n.Name], podCount[n.Name])
		if nc.Tainted {
			report.ExcludedNodes = append(report.ExcludedNodes, nc)
		} else {
			report.Nodes = append(report.Nodes, nc)
		}
	}

	risks, err := a.detectConcentration(ctx, pods.Items, nodes.Items)
	if err != nil {
		return nil, err
	}
	report.ConcentrationRisks = risks

	return report, nil
}

// sumRequestsByNode suma los requests de CPU/memoria de los pods activos
// (no Succeeded/Failed) agrupados por el nodo donde están programados,
// igual que la tabla "Allocated resources" de `kubectl describe node`
// (suma los contenedores de spec.Containers; no considera initContainers).
func sumRequestsByNode(pods []corev1.Pod) (map[string]resource.Quantity, map[string]resource.Quantity, map[string]int) {
	cpu := make(map[string]resource.Quantity)
	mem := make(map[string]resource.Quantity)
	count := make(map[string]int)

	for _, p := range pods {
		if p.Spec.NodeName == "" {
			continue
		}
		if p.Status.Phase == corev1.PodSucceeded || p.Status.Phase == corev1.PodFailed {
			continue
		}

		node := p.Spec.NodeName
		count[node]++

		podCPU := cpu[node]
		podMem := mem[node]
		for _, c := range p.Spec.Containers {
			if q, ok := c.Resources.Requests[corev1.ResourceCPU]; ok {
				podCPU.Add(q)
			}
			if q, ok := c.Resources.Requests[corev1.ResourceMemory]; ok {
				podMem.Add(q)
			}
		}
		cpu[node] = podCPU
		mem[node] = podMem
	}

	return cpu, mem, count
}

func (a *Analyzer) buildNodeCapacity(ctx context.Context, n corev1.Node, reqCPU, reqMem resource.Quantity, podCount int) model.NodeCapacity {
	ready := false
	for _, cond := range n.Status.Conditions {
		if cond.Type == corev1.NodeReady {
			ready = cond.Status == corev1.ConditionTrue
			break
		}
	}

	tainted := false
	for _, t := range n.Spec.Taints {
		if t.Effect == corev1.TaintEffectNoSchedule || t.Effect == corev1.TaintEffectNoExecute {
			tainted = true
			break
		}
	}

	zone := n.Labels[zoneLabel]
	if zone == "" {
		zone = n.Labels[failureDomainZoneLabel]
	}

	allocCPU := n.Status.Allocatable[corev1.ResourceCPU]
	allocMem := n.Status.Allocatable[corev1.ResourceMemory]
	allocPods := n.Status.Allocatable[corev1.ResourcePods]

	cpuHeadroom := round2(headroomPercent(allocCPU, reqCPU))
	memHeadroom := round2(headroomPercent(allocMem, reqMem))

	nc := model.NodeCapacity{
		Name:                  n.Name,
		Roles:                 rolesFromLabels(n.Labels),
		Ready:                 ready,
		Unschedulable:         n.Spec.Unschedulable,
		Tainted:               tainted,
		Zone:                  zone,
		AllocatableCPU:        allocCPU.String(),
		AllocatableMemory:     allocMem.String(),
		AllocatablePods:       allocPods.String(),
		RequestedCPU:          reqCPU.String(),
		RequestedMemory:       reqMem.String(),
		PodCount:              podCount,
		CPUHeadroomPercent:    cpuHeadroom,
		MemoryHeadroomPercent: memHeadroom,
	}

	if a.prom != nil {
		a.enrichWithMetrics(ctx, &nc)
	}

	nc.Risk = nodeRisk(ready, n.Spec.Unschedulable, cpuHeadroom, memHeadroom, a.headroomThresholdPercent)
	return nc
}

// enrichWithMetrics añade el consumo histórico observado. Los errores de
// consulta se ignoran en silencio (ya quedan registrados en debug por
// promclient.Client.Query): un problema puntual con una query no debe
// tumbar todo el reporte de capacity.
func (a *Analyzer) enrichWithMetrics(ctx context.Context, nc *model.NodeCapacity) {
	if v, ok := a.queryScalar(ctx, promclient.NodeCPUAvgQuery(nc.Name, a.lookback)); ok {
		nc.ObservedCPUAvg = formatCPU(v)
	}
	if v, ok := a.queryScalar(ctx, promclient.NodeCPUPeakQuery(nc.Name, a.lookback)); ok {
		nc.ObservedCPUPeak = formatCPU(v)
	}
	if v, ok := a.queryScalar(ctx, promclient.NodeMemoryAvgQuery(nc.Name, a.lookback)); ok {
		nc.ObservedMemoryAvg = formatMemory(v)
	}
	if v, ok := a.queryScalar(ctx, promclient.NodeMemoryPeakQuery(nc.Name, a.lookback)); ok {
		nc.ObservedMemoryPeak = formatMemory(v)
	}
}

func (a *Analyzer) queryScalar(ctx context.Context, query string) (float64, bool) {
	val, err := a.prom.Query(ctx, query)
	if err != nil {
		return 0, false
	}
	return promclient.ScalarOrNaN(val)
}
