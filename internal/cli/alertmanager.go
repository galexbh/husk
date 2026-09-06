package cli

import (
	"context"
	"log/slog"

	"github.com/galexbh/husk/internal/alertmanager"
	"github.com/galexbh/husk/internal/k8sclient"
	"github.com/galexbh/husk/internal/model"
	"github.com/galexbh/husk/internal/promclient"
)

// buildAlertCorrelation intenta correlacionar los hallazgos de sizing/
// capacity con las alertas activas de Alertmanager. Si Alertmanager no
// está disponible (Kubernetes vanilla, ruta inaccesible, sin permisos),
// devuelve una AlertCorrelation con Available=false en vez de un error:
// `report generate` se genera igual, sin esta sección.
func buildAlertCorrelation(ctx context.Context, client *k8sclient.Client, logger *slog.Logger, findings []model.Finding) *model.AlertCorrelation {
	if !client.IsOpenShift {
		return &model.AlertCorrelation{Available: false, Reason: "el cluster no es OpenShift"}
	}

	url, err := client.AlertmanagerURL(ctx)
	if err != nil {
		logger.Debug("Alertmanager no accesible", "error", err)
		return &model.AlertCorrelation{Available: false, Reason: "Alertmanager no accesible"}
	}
	token, err := client.BearerToken()
	if err != nil {
		logger.Debug("no se pudo obtener el bearer token para Alertmanager", "error", err)
		return &model.AlertCorrelation{Available: false, Reason: "no se pudo obtener el bearer token"}
	}

	transport := promclient.DiscoverTransport(ctx, client.Kubernetes, logger)
	amClient := alertmanager.New(url, token, transport, logger)

	alerts, err := amClient.ListActiveAlerts(ctx)
	if err != nil {
		logger.Debug("no se pudieron listar las alertas activas", "error", err)
		return &model.AlertCorrelation{Available: false, Reason: "no se pudieron listar las alertas activas"}
	}

	return &model.AlertCorrelation{
		Available:            true,
		ActiveAlerts:         alerts,
		CorrelatedFindingIDs: alertmanager.Correlate(findings, alerts),
	}
}
