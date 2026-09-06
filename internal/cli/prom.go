package cli

import (
	"context"
	"log/slog"

	"github.com/galexbh/husk/internal/k8sclient"
	"github.com/galexbh/husk/internal/promclient"
)

// newOptionalPromClient intenta construir un cliente de Prometheus/Thanos
// Querier. Varios comandos (capacity nodes, score) tratan el consumo
// histórico como un enriquecimiento opcional: si no está disponible
// (Kubernetes vanilla, Thanos inaccesible, sin bearer token), degradan con
// gracia en vez de fallar. reason describe por qué, para mostrarlo al
// usuario cuando corresponda.
func newOptionalPromClient(ctx context.Context, client *k8sclient.Client, logger *slog.Logger) (prom *promclient.Client, reason string) {
	if !client.IsOpenShift {
		return nil, "el cluster no es OpenShift"
	}

	thanosURL, err := client.ThanosQuerierURL(ctx)
	if err != nil {
		logger.Debug("Thanos Querier no accesible", "error", err)
		return nil, "Thanos Querier no accesible"
	}

	token, err := client.BearerToken()
	if err != nil {
		logger.Debug("no se pudo obtener el bearer token para Prometheus", "error", err)
		return nil, "no se pudo obtener el bearer token para Prometheus"
	}

	transport := promclient.DiscoverTransport(ctx, client.Kubernetes, logger)
	promCli, err := promclient.New(thanosURL, token, transport, logger)
	if err != nil {
		logger.Debug("no se pudo crear el cliente de Prometheus", "error", err)
		return nil, "no se pudo crear el cliente de Prometheus"
	}

	return promCli, ""
}
