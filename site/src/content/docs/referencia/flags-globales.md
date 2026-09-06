---
title: Flags globales
description: Flags heredados por todos los subcomandos de husk.
---

Todos los subcomandos de `husk` heredan estos flags desde el comando raíz:

| Flag | Descripción |
|---|---|
| `--kubeconfig <path>` | Ruta al kubeconfig. Por defecto, `$KUBECONFIG` o `~/.kube/config`. |
| `--context <name>` | Contexto del kubeconfig a usar. |
| `-n, --namespace <ns>` | Namespace por defecto para los comandos que lo aceptan (`inventory`, `sizing report`). Los comandos cluster-wide (`capacity nodes`, `dr assess`, `score`, `report generate`) lo ignoran a propósito. |
| `-o, --output <fmt>` | `table` (por defecto) \| `markdown` \| `json` \| `excel`. No todos los comandos soportan `excel` — ver cada página de referencia. |
| `--output-file <path>` | Archivo de salida. Requerido para `excel`; opcional para el resto (por defecto, stdout). |
| `-v, --verbose` | Habilita logging de nivel debug, incluidas las queries PromQL ejecutadas y sus tiempos de respuesta. |
| `--config <path>` | Ruta a un archivo de configuración YAML. Por defecto, `~/.husk/config.yaml` si existe — ver [Configuración](/husk/guias/configuracion/). |

## Variables de entorno

| Variable | Efecto |
|---|---|
| `KUBECONFIG` | Ruta al kubeconfig, si `--kubeconfig` no se especifica. |
| `HUSK_DEBUG=1` | Igual que `--verbose`, para causa técnica y stack trace en los mensajes de error, sin tener que repetir el flag. |

## Manejo de errores

Los errores de husk siempre tienen la forma:

```
Error: <qué falló>
Sugerencia: <qué hacer al respecto>
```

La causa técnica completa y el stack trace solo se muestran con
`--verbose`/`HUSK_DEBUG=1` — el resto del tiempo, el mensaje de error es
intencionalmente corto y accionable.
