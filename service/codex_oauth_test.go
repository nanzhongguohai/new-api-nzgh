package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestExchangeCodexAuthorizationCodeIncludesUpstreamErrorDetails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm failed: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"message":"Access denied by upstream","type":"invalid_request_error","code":"access_denied"}}`))
	}))
	defer server.Close()

	_, err := exchangeCodexAuthorizationCode(
		context.Background(),
		server.Client(),
		server.URL,
		codexOAuthClientID,
		"dummy-code",
		"dummy-verifier",
		codexOAuthRedirectURI,
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	got := err.Error()
	if !strings.Contains(got, "status=403") {
		t.Fatalf("expected status detail, got %q", got)
	}
	if !strings.Contains(got, "Access denied by upstream") {
		t.Fatalf("expected upstream message, got %q", got)
	}
	if !strings.Contains(got, "code=access_denied") {
		t.Fatalf("expected upstream code, got %q", got)
	}
}

func TestRefreshCodexOAuthTokenParsesSuccessPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"access","refresh_token":"refresh","expires_in":3600}`))
	}))
	defer server.Close()

	res, err := refreshCodexOAuthToken(
		context.Background(),
		server.Client(),
		server.URL,
		codexOAuthClientID,
		"refresh-token",
	)
	if err != nil {
		t.Fatalf("refreshCodexOAuthToken returned error: %v", err)
	}
	if res.AccessToken != "access" {
		t.Fatalf("unexpected access token: %q", res.AccessToken)
	}
	if res.RefreshToken != "refresh" {
		t.Fatalf("unexpected refresh token: %q", res.RefreshToken)
	}
	if res.ExpiresAt.IsZero() {
		t.Fatal("expected expires_at to be set")
	}
}
