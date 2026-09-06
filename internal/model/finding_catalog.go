package model

// FindingGuidance es la explicación y la recomendación asociadas a una
// categoría de hallazgo.
type FindingGuidance struct {
	// Explanation dice qué significa el hallazgo y por qué importa.
	Explanation string
	// Recommendation dice qué acción concreta tomar.
	Recommendation string
}

// findingCatalog es la fuente única de verdad categoría -> guía, reutilizada
// por cualquier Finding sin importar qué analizador lo produjo ni qué
// comando lo muestre (incluido cualquier `--output json` individual). Si
// agregas una categoría de hallazgo nueva (un model.NewFinding con un
// category distinto), agrega su entrada aquí en el mismo cambio — ver
// CLAUDE.md.
var findingCatalog = map[string]FindingGuidance{
	"single-replica": {
		Explanation:    "El workload corre con una sola réplica: la pérdida de un único pod (nodo drenado, reinicio, crash) causa una interrupción completa del servicio.",
		Recommendation: "Aumenta replicas a 2 o más para tolerar la pérdida de un pod, o documenta por qué el workload es de instancia única.",
	},
	"missing-limits": {
		Explanation:    "Uno o más contenedores del workload no declaran resources.limits: pueden consumir CPU/memoria sin tope y afectar a otros workloads del mismo nodo.",
		Recommendation: "Define resources.limits.cpu y resources.limits.memory; usa `husk sizing report --dry-run` para una recomendación basada en consumo real.",
	},
	"missing-resourcequota": {
		Explanation:    "El namespace no tiene ningún ResourceQuota: no hay un tope agregado a lo que sus workloads pueden consumir en conjunto.",
		Recommendation: "Crea un ResourceQuota para el namespace, para evitar que un workload sin límites agote la capacidad del cluster.",
	},
	"sizing-no-limits": {
		Explanation:    "El contenedor tiene consumo histórico observable pero no declara resources.limits: no hay tope a su consumo real.",
		Recommendation: "Define resources.limits; usa `husk sizing report --dry-run` para el patch sugerido.",
	},
	"sizing-under-provisioned": {
		Explanation:    "El consumo observado (CPU P95 o memoria P99) supera el request declarado: el contenedor corre en riesgo de throttling de CPU u OOM kill.",
		Recommendation: "Aumenta requests/limits: el consumo observado supera lo declarado (riesgo de throttling/OOM). Ver `husk sizing report --dry-run`.",
	},
	"sizing-over-provisioned": {
		Explanation:    "El request declarado supera holgadamente (por el factor sizing.over_provision_factor) al consumo observado: el contenedor reserva más capacidad del cluster de la que usa.",
		Recommendation: "Reduce requests/limits al consumo real observado para liberar headroom del cluster. Ver `husk sizing report --dry-run`.",
	},
	"oadp-not-installed": {
		Explanation:    "El operador OADP (backups gestionados con Velero) no está instalado en el cluster: no hay mecanismo de backup a nivel de namespace/aplicación.",
		Recommendation: "Instala y configura el operador OADP en openshift-adp para habilitar backups gestionados con Velero.",
	},
	"oadp-unhealthy": {
		Explanation:    "OADP está instalado pero no reconcilia correctamente: los backups pueden no estar ejecutándose aunque parezca configurado.",
		Recommendation: "Revisa los logs del operador OADP y la DataProtectionApplication: la reconciliación está fallando.",
	},
	"missing-backup": {
		Explanation:    "El namespace no tiene ningún Backup de Velero completado que lo incluya: en un desastre, sus datos no son recuperables desde backup.",
		Recommendation: "Crea un Schedule/Backup de Velero que incluya este namespace.",
	},
	"stale-backup": {
		Explanation:    "El backup completado más reciente que incluye este namespace supera la antigüedad máxima configurada (dr.backup_max_age): un desastre hoy perdería más datos de los aceptables.",
		Recommendation: "Verifica que el Schedule de backup esté corriendo; el último backup completado supera la antigüedad máxima configurada.",
	},
	"etcd-snapshot-unverifiable": {
		Explanation:    "No se encontró evidencia (Job/CronJob) de un backup de etcd: la antigüedad real del último snapshot no se puede verificar automáticamente.",
		Recommendation: "Configura y documenta un CronJob de backup de etcd, o confirma manualmente la antigüedad del último snapshot.",
	},
	"etcd-snapshot-stale": {
		Explanation:    "El snapshot de etcd más reciente detectado supera la antigüedad máxima configurada (dr.etcd_snapshot_max_age): un desastre hoy perdería más estado del cluster del aceptable.",
		Recommendation: "Ejecuta un nuevo snapshot de etcd; el más reciente supera la antigüedad máxima configurada.",
	},
	"csi-snapshot-unsupported": {
		Explanation:    "La StorageClass no tiene un VolumeSnapshotClass asociado: los PVCs que la usan no se pueden respaldar vía snapshot CSI.",
		Recommendation: "Usa una StorageClass cuyo provisioner tenga un VolumeSnapshotClass asociado, o crea uno para el provisioner actual.",
	},
	"missing-pdb": {
		Explanation:    "El workload crítico no tiene un PodDisruptionBudget: un drenado de nodo (mantenimiento, autoescalado, upgrade) puede tumbar todas sus réplicas a la vez.",
		Recommendation: "Define un PodDisruptionBudget que cubra este workload, para protegerlo durante drenados de nodos.",
	},
	"missing-topology-spread": {
		Explanation:    "El workload crítico no declara topologySpreadConstraints: sus réplicas pueden terminar concentradas en el mismo nodo/zona, perdiendo la tolerancia a fallos que da tener varias réplicas.",
		Recommendation: "Agrega topologySpreadConstraints al pod template para distribuir las réplicas entre nodos/zonas.",
	},
	"node-saturated": {
		Explanation:    "El nodo tiene headroom crítico en CPU y memoria simultáneamente (o no está Ready/está unschedulable): un pico de carga adicional puede no poder programarse o causar eviction por presión de recursos.",
		Recommendation: "Añade capacidad al cluster o redistribuye carga: el nodo tiene poco headroom libre.",
	},
	"node-headroom-warning": {
		Explanation:    "El nodo tiene headroom ajustado en un solo eje (CPU o memoria, no ambos): todavía no es una saturación crítica, pero merece vigilancia antes de que escale.",
		Recommendation: "Vigila el eje con headroom ajustado; si se acerca al umbral crítico, añade capacidad o redistribuye carga antes de que escale a ALTO.",
	},
	"concentration-risk": {
		Explanation:    "Las réplicas de este workload están concentradas en muy pocos nodos o zonas: la pérdida de ese nodo/zona afecta a una fracción desproporcionada de sus réplicas.",
		Recommendation: "Agrega topologySpreadConstraints o antiafinidad para distribuir las réplicas en más nodos/zonas.",
	},
}

// unmappedCategoryFallback se usa cuando una categoría de Finding no tiene
// entrada en findingCatalog. Es un texto visible a propósito (no un campo
// vacío) para que un olvido se note en el JSON/Excel de salida, en vez de
// fallar en silencio.
const unmappedCategoryFallback = "sin explicación documentada para esta categoría de hallazgo"

// Guidance devuelve la guía para una categoría de hallazgo. Si la categoría
// no está catalogada, devuelve un texto que nombra explícitamente la
// categoría faltante y el archivo a editar.
func Guidance(category string) FindingGuidance {
	if g, ok := findingCatalog[category]; ok {
		return g
	}
	return FindingGuidance{
		Explanation:    unmappedCategoryFallback,
		Recommendation: `falta agregar la categoría "` + category + `" a internal/model/finding_catalog.go`,
	}
}
