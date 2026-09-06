// Package alertmanager recolecta alertas activas de Alertmanager (el
// stack de monitoreo de OpenShift) para correlacionarlas con los
// hallazgos de sizing y capacity: distingue un incidente actual (hallazgo
// con una alerta activa relacionada) de un riesgo preventivo. Solo
// recolecta y correlaciona — no genera salida visual.
package alertmanager

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/galexbh/husk/internal/huskerr"
	"github.com/galexbh/husk/internal/model"
)

// Logger es el subconjunto de *slog.Logger que este paquete necesita.
type Logger interface {
	Debug(msg string, args ...any)
}

// Client habla con la API v2 de Alertmanager.
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
	logger     Logger
}

// New construye un Client contra baseURL (la route de Alertmanager),
// autenticando con bearerToken sobre transport (ver
// promclient.DiscoverTransport para la resolución de TLS: Alertmanager
// pasa por la misma route por defecto de OpenShift que Thanos Querier).
func New(baseURL, bearerToken string, transport http.RoundTripper, logger Logger) *Client {
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		token:      bearerToken,
		httpClient: &http.Client{Transport: transport, Timeout: 15 * time.Second},
		logger:     logger,
	}
}

// apiAlert es la forma cruda de una alerta en la API v2 de Alertmanager
// (https://github.com/prometheus/alertmanager/blob/main/api/v2/openapi.yaml).
type apiAlert struct {
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	StartsAt    time.Time         `json:"startsAt"`
	Status      struct {
		State string `json:"state"`
	} `json:"status"`
}

// ListActiveAlerts devuelve las alertas activas (no silenciadas, no
// inhibidas) del cluster.
func (c *Client) ListActiveAlerts(ctx context.Context) ([]model.Alert, error) {
	url := c.baseURL + "/api/v2/alerts?active=true&silenced=false&inhibited=false"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, huskerr.New("no se pudo construir la petición a Alertmanager", "esto es un bug de husk; por favor reporta el issue", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)

	start := time.Now()
	resp, err := c.httpClient.Do(req)
	if c.logger != nil {
		c.logger.Debug("alertmanager query", "url", url, "elapsed", time.Since(start).String())
	}
	if err != nil {
		return nil, huskerr.New("no se pudo conectar a Alertmanager", "verifica que la route alertmanager-main esté accesible", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, huskerr.New(
			fmt.Sprintf("Alertmanager respondió %d", resp.StatusCode),
			"verifica que tengas el rol cluster-monitoring-view",
			nil,
		)
	}
	if resp.StatusCode >= 300 {
		return nil, huskerr.New(fmt.Sprintf("Alertmanager respondió %d", resp.StatusCode), "revisa el estado del stack de monitoreo", nil)
	}

	var raw []apiAlert
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, huskerr.New("no se pudo interpretar la respuesta de Alertmanager", "la versión de la API puede haber cambiado; reporta el issue", err)
	}

	alerts := make([]model.Alert, 0, len(raw))
	for _, a := range raw {
		alerts = append(alerts, model.Alert{
			Name:      a.Labels["alertname"],
			Namespace: a.Labels["namespace"],
			Pod:       a.Labels["pod"],
			Container: a.Labels["container"],
			Severity:  a.Labels["severity"],
			State:     a.Status.State,
			StartsAt:  a.StartsAt,
			Summary:   a.Annotations["summary"],
		})
	}
	return alerts, nil
}
