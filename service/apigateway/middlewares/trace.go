package middlewares

import (
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// TraceContextMiddleware extracts the incoming W3C trace context
// (traceparent/tracestate) from the HTTP request headers and attaches it to
// the request context, so spans created by the gateway become children of the
// trace started upstream (e.g. NGINX or the client). When no trace context is
// present a new trace is started by the span created in the route handler.
func TraceContextMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := otel.GetTextMapPropagator().Extract(
				r.Context(),
				propagation.HeaderCarrier(r.Header),
			)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
