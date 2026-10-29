package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPageHandlersRenderTemplates(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
	}{
		{name: "index", handler: index},
		{name: "create", handler: create},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/", nil)

			tt.handler(response, request)

			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d; body: %s", response.Code, http.StatusOK, response.Body.String())
			}
			if !strings.Contains(response.Body.String(), "Cover your page.") {
				t.Fatalf("response does not contain expected page content: %s", response.Body.String())
			}
		})
	}
}
