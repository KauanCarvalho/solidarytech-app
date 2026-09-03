package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// otel.Meter(...) sem um MeterProvider configurado retorna a implementação
// no-op da API — os testes abaixo não precisam de Collector/rede.

func TestNewHTTPMetrics(t *testing.T) {
	m, err := newHTTPMetrics("donation-service-test")
	if err != nil {
		t.Fatalf("erro inesperado ao criar métricas: %v", err)
	}

	if m.requestsTotal == nil {
		t.Fatal("requestsTotal não deveria ser nil")
	}

	if m.requestDuration == nil {
		t.Fatal("requestDuration não deveria ser nil")
	}
}

func TestMiddleware_ChainsNextHandler(t *testing.T) {
	m, err := newHTTPMetrics("donation-service-test-middleware")
	if err != nil {
		t.Fatalf("erro ao criar métricas: %v", err)
	}

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusCreated)
	})

	handler := m.middleware(next)

	req := httptest.NewRequest(http.MethodPost, "/donations", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if !called {
		t.Fatal("o handler encadeado não foi chamado pelo middleware")
	}

	if rec.Code != http.StatusCreated {
		t.Fatalf("esperado 201, obtido %d", rec.Code)
	}
}

func TestStatusRecorder_TracksWriteHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	sr := &statusRecorder{ResponseWriter: rec, status: http.StatusOK}

	if sr.status != http.StatusOK {
		t.Fatalf("status inicial esperado 200, obtido %d", sr.status)
	}

	sr.WriteHeader(http.StatusNotFound)

	if sr.status != http.StatusNotFound {
		t.Fatalf("status esperado 404 após WriteHeader, obtido %d", sr.status)
	}

	if rec.Code != http.StatusNotFound {
		t.Fatalf("statusRecorder deveria propagar WriteHeader ao ResponseWriter real, obtido %d", rec.Code)
	}
}
