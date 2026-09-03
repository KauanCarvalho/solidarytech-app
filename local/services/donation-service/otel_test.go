package main

import (
	"os"
	"testing"
)

func TestOtlpEndpoint_DefaultFallback(t *testing.T) {
	os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")

	got := otlpEndpoint()
	want := "otel-collector.monitoring.svc.cluster.local:4317"

	if got != want {
		t.Fatalf("esperado endpoint padrão %q, obtido %q", want, got)
	}
}

func TestOtlpEndpoint_RespectsEnvVar(t *testing.T) {
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "custom-collector:4317")

	got := otlpEndpoint()
	want := "custom-collector:4317"

	if got != want {
		t.Fatalf("esperado endpoint customizado %q, obtido %q", want, got)
	}
}
