// Package promclient recolecta datos de Prometheus/Thanos Querier: resuelve
// el transporte HTTP (bearer token + TLS), ejecuta queries PromQL y expone
// helpers tipados. Como internal/k8sclient, solo recolecta — no genera
// salida visual ni contiene lógica de negocio de sizing/capacity.
package promclient

import (
	"net/http"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"

	"github.com/galexbh/husk/internal/huskerr"
)

// Logger es el subconjunto de *slog.Logger que este paquete necesita, para
// no acoplar su firma pública a slog.
type Logger interface {
	Debug(msg string, args ...any)
	Warn(msg string, args ...any)
}

// Client envuelve la API HTTP de Prometheus/Thanos Querier.
type Client struct {
	api    v1.API
	logger Logger
}

// New construye un Client contra baseURL (la route de Thanos Querier),
// autenticando con bearerToken sobre transport (ver DiscoverTransport para
// la resolución de TLS).
func New(baseURL, bearerToken string, transport http.RoundTripper, logger Logger) (*Client, error) {
	cli, err := api.NewClient(api.Config{
		Address:      baseURL,
		RoundTripper: &bearerRoundTripper{token: bearerToken, next: transport},
	})
	if err != nil {
		return nil, huskerr.New(
			"no se pudo crear el cliente de Prometheus/Thanos",
			"verifica la URL de Thanos Querier y el bearer token",
			err,
		)
	}
	return &Client{api: v1.NewAPI(cli), logger: logger}, nil
}
