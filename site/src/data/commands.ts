// Fuente única de datos para el buscador de comandos interactivo
// (src/components/CommandFinder.astro). Mantenla en sincronía con las
// páginas de referencia bajo src/content/docs/referencia/.
export interface CommandFlag {
  flag: string;
  description: string;
}

export interface CommandEntry {
  /** Nombre completo tal como se escribe en la terminal. */
  name: string;
  /** Categoría para agrupar/filtrar. */
  category: string;
  /** Descripción de una línea. */
  summary: string;
  /** Flags propios del comando (no los globales). */
  flags: CommandFlag[];
  /** Insignias cortas: requisitos o capacidades notables. */
  badges?: string[];
  /** Ruta a la página de referencia completa. */
  href: string;
}

export const commands: CommandEntry[] = [
  {
    name: "husk connect health",
    category: "Conectividad",
    summary:
      "Valida API server, tipo de cluster, Thanos Querier y permisos mínimos de lectura.",
    flags: [],
    badges: ["diagnóstico"],
    href: "/husk/referencia/connect/",
  },
  {
    name: "husk init rbac",
    category: "Configuración inicial",
    summary:
      "Genera el ClusterRole/ClusterRoleBinding de solo lectura para un ServiceAccount.",
    flags: [
      { flag: "--name", description: "Nombre del ClusterRole/ClusterRoleBinding (default husk-reader)." },
      { flag: "--service-account", description: "ServiceAccount que usará el rol (default husk)." },
      { flag: "--service-account-namespace", description: "Namespace del ServiceAccount (default husk)." },
    ],
    badges: ["solo lectura"],
    href: "/husk/referencia/init/",
  },
  {
    name: "husk inventory",
    category: "Inventario",
    summary:
      "Inventario completo: Deployments, StatefulSets, DaemonSets, Services, PVCs, ConfigMaps, Secrets (solo nombres), Nodes, StorageClasses, CRDs.",
    flags: [
      { flag: "--extended", description: "Incluye RBAC, NetworkPolicies, PDBs, ResourceQuotas, LimitRanges, HPAs, Ingresses/Routes." },
    ],
    badges: ["Excel"],
    href: "/husk/referencia/inventory/",
  },
  {
    name: "husk inventory summary",
    category: "Inventario",
    summary:
      "Versión ejecutiva del inventario: conteos agregados y hallazgos de riesgo.",
    flags: [
      { flag: "--extended", description: "Necesario para el hallazgo de namespaces sin ResourceQuota." },
    ],
    href: "/husk/referencia/inventory/",
  },
  {
    name: "husk sizing report",
    category: "Sizing",
    summary:
      "Compara requests/limits contra el consumo histórico real (CPU P95, memoria P99 y pico).",
    flags: [
      { flag: "--dry-run", description: "Imprime el patch YAML sugerido por contenedor; nunca aplica cambios." },
    ],
    badges: ["requiere OpenShift", "--dry-run"],
    href: "/husk/referencia/sizing/",
  },
  {
    name: "husk capacity nodes",
    category: "Capacity",
    summary:
      "Allocatable vs requests por nodo, headroom y riesgos de concentración de carga.",
    flags: [],
    badges: ["funciona sin Prometheus"],
    href: "/husk/referencia/capacity/",
  },
  {
    name: "husk dr assess",
    category: "DR readiness",
    summary:
      "OADP/Velero, cobertura de backups, snapshot de etcd, soporte CSI snapshot, PDBs y topology spread.",
    flags: [],
    href: "/husk/referencia/dr/",
  },
  {
    name: "husk score",
    category: "Score",
    summary:
      "Score de resiliencia (0-100): sizing, DR, capacity, PDB y topology spread ponderados.",
    flags: [],
    href: "/husk/referencia/score/",
  },
  {
    name: "husk report generate",
    category: "Reportes",
    summary:
      "Reporte consolidado: resumen ejecutivo, hallazgos priorizados, detalle técnico y recomendaciones. Guarda un snapshot en el historial.",
    flags: [
      { flag: "--appendix", description: "Incluye el snapshot completo como apéndice JSON." },
    ],
    badges: ["Excel", "historial"],
    href: "/husk/referencia/report/",
  },
  {
    name: "husk report diff",
    category: "Reportes",
    summary: "Compara dos snapshots del historial: regresiones y mejoras.",
    flags: [
      { flag: "--from", description: "Snapshot inicial (JSON). Requerido." },
      { flag: "--to", description: "Snapshot final (JSON). Requerido." },
    ],
    href: "/husk/referencia/report/",
  },
  {
    name: "husk export grafana-dashboard",
    category: "Exportación",
    summary:
      "Dashboard de Grafana (JSON exportable) con paneles de CPU/memoria por namespace y nodo, más alertas activas.",
    flags: [],
    badges: ["no requiere cluster"],
    href: "/husk/referencia/export/",
  },
  {
    name: "husk version",
    category: "Utilidades",
    summary: "Versión, commit, fecha de build y versión de Go.",
    flags: [],
    href: "/husk/empezando/instalacion/#verificar-la-instalación",
  },
  {
    name: "husk completion",
    category: "Utilidades",
    summary: "Autocompletado de shell (bash, zsh, fish).",
    flags: [],
    href: "/husk/empezando/instalacion/#autocompletado-de-shell",
  },
];

export const globalFlags: CommandFlag[] = [
  { flag: "--kubeconfig", description: "Ruta al kubeconfig ($KUBECONFIG o ~/.kube/config por defecto)." },
  { flag: "--context", description: "Contexto del kubeconfig a usar." },
  { flag: "-n, --namespace", description: "Namespace por defecto (inventory, sizing report)." },
  { flag: "-o, --output", description: "table (default) | markdown | json | excel." },
  { flag: "--output-file", description: "Archivo de salida; requerido para excel." },
  { flag: "-v, --verbose", description: "Logging de nivel debug, incluidas las queries PromQL." },
  { flag: "--config", description: "Ruta a un config.yaml (~/.husk/config.yaml por defecto)." },
];
