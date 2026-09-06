// @ts-check
import { defineConfig } from "astro/config";
import starlight from "@astrojs/starlight";

export default defineConfig({
  site: "https://galexbh.github.io",
  base: "/husk",
  integrations: [
    starlight({
      title: "husk",
      description:
        "CLI de sizing, DR readiness e inventario para Kubernetes/OpenShift.",
      social: [
        {
          icon: "github",
          label: "GitHub",
          href: "https://github.com/galexbh/husk",
        },
      ],
      editLink: {
        baseUrl: "https://github.com/galexbh/husk/edit/main/site/",
      },
      locales: {
        root: { label: "Español", lang: "es" },
      },
      sidebar: [
        {
          label: "Empezando",
          items: [
            { label: "Introducción", slug: "" },
            { label: "Instalación", slug: "empezando/instalacion" },
            { label: "Guía rápida", slug: "empezando/guia-rapida" },
          ],
        },
        {
          label: "Guías",
          items: [
            { label: "Configuración (config.yaml)", slug: "guias/configuracion" },
            { label: "RBAC y permisos", slug: "guias/rbac" },
            { label: "Docker y OCI", slug: "guias/docker" },
            { label: "Historial y report diff", slug: "guias/historial-y-diff" },
            { label: "Métricas PromQL", slug: "guias/metricas" },
          ],
        },
        {
          label: "Referencia de comandos",
          items: [
            { label: "Buscador de comandos", slug: "referencia/buscador" },
            { label: "Flags globales", slug: "referencia/flags-globales" },
            { label: "husk connect health", slug: "referencia/connect" },
            { label: "husk init rbac", slug: "referencia/init" },
            { label: "husk inventory", slug: "referencia/inventory" },
            { label: "husk sizing report", slug: "referencia/sizing" },
            { label: "husk capacity nodes", slug: "referencia/capacity" },
            { label: "husk dr assess", slug: "referencia/dr" },
            { label: "husk score", slug: "referencia/score" },
            { label: "husk report", slug: "referencia/report" },
            { label: "husk export grafana-dashboard", slug: "referencia/export" },
          ],
        },
        {
          label: "Arquitectura",
          items: [
            { label: "Estructura del proyecto", slug: "arquitectura/estructura" },
            { label: "Modelo de datos y seguridad", slug: "arquitectura/modelo-y-seguridad" },
          ],
        },
      ],
    }),
  ],
});
