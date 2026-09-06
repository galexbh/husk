package promclient

import (
	"context"
	"math"
	"time"

	"github.com/prometheus/common/model"

	"github.com/galexbh/husk/internal/huskerr"
)

// Query ejecuta una consulta instantánea de PromQL contra Thanos Querier,
// registrando la query y su latencia en el logger de debug (--verbose).
func (c *Client) Query(ctx context.Context, query string) (model.Value, error) {
	start := time.Now()
	result, warnings, err := c.api.Query(ctx, query, time.Now())
	elapsed := time.Since(start)

	c.logger.Debug("promql query", "query", query, "elapsed", elapsed.String(), "warnings", len(warnings))

	if err != nil {
		return nil, huskerr.New(
			"la consulta a Prometheus/Thanos Querier falló",
			"corre con --verbose para ver la query ejecutada; verifica que tengas el rol cluster-monitoring-view",
			err,
		)
	}
	return result, nil
}

// ScalarOrNaN extrae el primer valor de un model.Vector, o reporta que no
// hay datos (vector vacío o valor NaN, que Prometheus devuelve cuando la
// serie no existe en la ventana consultada).
func ScalarOrNaN(v model.Value) (float64, bool) {
	vec, ok := v.(model.Vector)
	if !ok || len(vec) == 0 {
		return 0, false
	}
	val := float64(vec[0].Value)
	if math.IsNaN(val) {
		return 0, false
	}
	return val, true
}
