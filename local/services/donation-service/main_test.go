package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Os testes abaixo cobrem apenas os caminhos do DonationHandler que não tocam
// o banco de dados (payload inválido / método não permitido) e o HealthHandler.
// O caminho de escrita/leitura real no Postgres é coberto pelo smoke test de
// integração (scripts/check/donation-service.sh), não por teste unitário.

func TestHealthHandler(t *testing.T) {
	app := &App{}
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	app.HealthHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("esperado status 200, obtido %d", rec.Code)
	}

	if body := rec.Body.String(); !strings.Contains(body, `"status":"ok"`) {
		t.Fatalf("corpo inesperado: %s", body)
	}
}

func TestDonationHandler_MethodNotAllowed(t *testing.T) {
	app := &App{}
	req := httptest.NewRequest(http.MethodDelete, "/donations", nil)
	rec := httptest.NewRecorder()

	app.DonationHandler(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("esperado 405, obtido %d", rec.Code)
	}
}

func TestDonationHandler_InvalidPayload(t *testing.T) {
	app := &App{}
	body := strings.NewReader(`{"amount": "não é número"`) // JSON malformado de propósito
	req := httptest.NewRequest(http.MethodPost, "/donations", body)
	rec := httptest.NewRecorder()

	app.DonationHandler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("esperado 400 para payload inválido, obtido %d", rec.Code)
	}
}
