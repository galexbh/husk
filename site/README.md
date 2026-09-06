# Documentación de husk (Starlight)

Sitio de documentación de `husk`, construido con
[Astro](https://astro.build) + [Starlight](https://starlight.astro.build/es/).
Es un proyecto Node.js independiente del módulo Go del CLI — vive en este
subdirectorio a propósito, para no mezclar sus dependencias con `go.mod`.

## Desarrollo local

```sh
npm install
npm run dev       # http://localhost:4321/husk/
```

## Build de producción

```sh
npm run build     # genera ./dist/
npm run preview   # sirve ./dist/ localmente para revisarlo
```

## Estructura

```
site/
├── astro.config.mjs        # integración de Starlight, título, sidebar
├── src/
│   ├── content.config.ts    # esquema de la colección "docs"
│   ├── data/commands.ts       # fuente única del buscador de comandos
│   ├── components/
│   │   └── CommandFinder.astro # buscador interactivo (filtrado en el cliente)
│   └── content/docs/          # todas las páginas, en Markdown/MDX
│       ├── index.mdx           # portada
│       ├── empezando/
│       ├── guias/
│       ├── referencia/
│       │   └── buscador.mdx      # usa <CommandFinder />
│       └── arquitectura/
```

Al agregar o cambiar un comando/flag, actualiza `src/data/commands.ts` — el
buscador (`/referencia/buscador/`) y su filtrado en vivo leen de ahí, sin
necesitar backend ni índice externo.

Cada página nueva bajo `src/content/docs/` necesita agregarse también al
`sidebar` de `astro.config.mjs` para aparecer en la navegación.

## Despliegue

Se publica automáticamente en GitHub Pages en cada push a `main` que toque
`site/**`, vía
[`.github/workflows/docs.yml`](../.github/workflows/docs.yml).
