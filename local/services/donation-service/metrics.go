package main

import (
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// httpMetrics agrupa os instrumentos de métricas HTTP, roteados pelo OTel Collector
// para o Prometheus via prometheusremotewrite.
type httpMetrics struct {
	requestsTotal   metric.Int64Counter
	requestDuration metric.Float64Histogram
}

// newHTTPMetrics cria os instrumentos http_requests_total (counter) e
// http_request_duration_seconds (histogram) — nomes usados diretamente pelas
// PrometheusRules de SLO e pelo dashboard SRE do Grafana.
func newHTTPMetrics(serviceName string) (*httpMetrics, error) {
	meter := otel.Meter(serviceName)

	requestsTotal, err := meter.Int64Counter(
		"http_requests_total",
		metric.WithDescription("Total de requisições HTTP recebidas"),
	)
	if err != nil {
		return nil, err
	}

	requestDuration, err := meter.Float64Histogram(
		"http_request_duration_seconds",
		metric.WithDescription("Duração das requisições HTTP em segundos"),
	)
	if err != nil {
		return nil, err
	}

	return &httpMetrics{requestsTotal: requestsTotal, requestDuration: requestDuration}, nil
}

// statusRecorder captura o status code da resposta para o middleware de métricas.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

// middleware envolve um handler HTTP, registrando contagem e duração de cada requisição —
// os dois SLIs do donation-service (disponibilidade e latência) são calculados a partir daqui.
func (m *httpMetrics) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rec, r)

		attrs := metric.WithAttributes(
			attribute.String("method", r.Method),
			attribute.String("path", r.URL.Path),
			attribute.Int("status", rec.status),
		)

		m.requestsTotal.Add(r.Context(), 1, attrs)
		m.requestDuration.Record(r.Context(), time.Since(start).Seconds(), attrs)
	})
}
