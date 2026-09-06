package promclient

import "net/http"

// bearerRoundTripper adjunta el bearer token del usuario a cada petición,
// delegando el resto (incluida la validación TLS) al transporte siguiente.
type bearerRoundTripper struct {
	token string
	next  http.RoundTripper
}

func (t *bearerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	cloned := req.Clone(req.Context())
	cloned.Header.Set("Authorization", "Bearer "+t.token)

	next := t.next
	if next == nil {
		next = http.DefaultTransport
	}
	return next.RoundTrip(cloned)
}
