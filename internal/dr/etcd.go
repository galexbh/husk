package dr

import (
	"context"
	"fmt"
	"regexp"
	"time"

	configv1 "github.com/openshift/api/config/v1"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/galexbh/husk/internal/huskerr"
	"github.com/galexbh/husk/internal/model"
)

// etcdBackupNamePattern es la heurística usada para encontrar el
// CronJob/Job de backup de etcd: OpenShift no expone una API de solo
// lectura que indique directamente la antigüedad del snapshot, así que se
// infiere del último Job exitoso cuyo nombre sugiera que es un backup de
// etcd (por ejemplo, "etcd-backup", "backup-etcd", "etcd-snapshot").
var etcdBackupNamePattern = regexp.MustCompile(`(?i)(etcd.*(backup|snapshot))|((backup|snapshot).*etcd)`)

// assessEtcdSnapshot solo se evalúa en OpenShift. Si no encuentra
// evidencia de un backup de etcd, lo reporta como "no verificable" en vez
// de fallar — el usuario deberá confirmarlo manualmente.
func (a *Assessor) assessEtcdSnapshot(ctx context.Context, dr *model.DRReadiness) error {
	if !a.client.IsOpenShift {
		dr.EtcdCheckApplicable = false
		return nil
	}
	dr.EtcdCheckApplicable = true

	status := &model.EtcdSnapshotStatus{}
	dr.EtcdSnapshot = status

	if a.client.Config != nil {
		co, err := a.client.Config.ConfigV1().ClusterOperators().Get(ctx, "etcd", metav1.GetOptions{})
		if err == nil {
			status.ClusterOperatorHealthy = clusterOperatorHealthy(co)
		}
		// Un error aquí (por ejemplo, sin permiso de lectura sobre
		// clusteroperators) no debe tumbar todo el assess: se deja
		// ClusterOperatorHealthy en false, informativo.
	}

	maxAge, err := time.ParseDuration(a.cfg.EtcdSnapshotMaxAge)
	if err != nil {
		maxAge = 7 * 24 * time.Hour
	}

	lastSuccess, source, found, err := a.findEtcdBackupEvidence(ctx)
	if err != nil {
		return err
	}
	if !found {
		status.Verifiable = false
		return nil
	}

	status.Verifiable = true
	status.LastSuccessTime = lastSuccess
	status.Source = source
	status.AgeOK = time.Since(lastSuccess) <= maxAge
	return nil
}

// clusterOperatorHealthy interpreta status.conditions de un
// ClusterOperator (config.openshift.io/v1), buscando Available=True y
// Degraded=False.
func clusterOperatorHealthy(co *configv1.ClusterOperator) bool {
	available, degraded := false, false
	for _, c := range co.Status.Conditions {
		switch c.Type {
		case configv1.OperatorAvailable:
			available = c.Status == configv1.ConditionTrue
		case configv1.OperatorDegraded:
			degraded = c.Status == configv1.ConditionTrue
		}
	}
	return available && !degraded
}

func (a *Assessor) findEtcdBackupEvidence(ctx context.Context) (time.Time, string, bool, error) {
	cronJobs, err := a.client.Kubernetes.BatchV1().CronJobs("").List(ctx, listOpts)
	if err != nil {
		return time.Time{}, "", false, huskerr.New("no se pudo listar CronJobs", "verifica el permiso de lectura sobre cronjobs", err)
	}

	var best time.Time
	var bestSource string
	found := false

	for _, cj := range cronJobs.Items {
		if !etcdBackupNamePattern.MatchString(cj.Name) {
			continue
		}
		if cj.Status.LastSuccessfulTime == nil {
			continue
		}
		t := cj.Status.LastSuccessfulTime.Time
		if !found || t.After(best) {
			best = t
			bestSource = fmt.Sprintf("CronJob/%s/%s", cj.Namespace, cj.Name)
			found = true
		}
	}

	jobs, err := a.client.Kubernetes.BatchV1().Jobs("").List(ctx, listOpts)
	if err != nil {
		return time.Time{}, "", false, huskerr.New("no se pudo listar Jobs", "verifica el permiso de lectura sobre jobs", err)
	}
	for _, j := range jobs.Items {
		if !etcdBackupNamePattern.MatchString(j.Name) {
			continue
		}
		if !jobSucceeded(j) || j.Status.CompletionTime == nil {
			continue
		}
		t := j.Status.CompletionTime.Time
		if !found || t.After(best) {
			best = t
			bestSource = fmt.Sprintf("Job/%s/%s", j.Namespace, j.Name)
			found = true
		}
	}

	return best, bestSource, found, nil
}

func jobSucceeded(j batchv1.Job) bool {
	for _, c := range j.Status.Conditions {
		if c.Type == batchv1.JobComplete && c.Status == "True" {
			return true
		}
	}
	return false
}
