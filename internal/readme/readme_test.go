package readme

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestValidateKey(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantErr string
	}{
		{"has credit", 200, `{"data":{"limit":50,"limit_remaining":47.6}}`, ""},
		{"no limit", 200, `{"data":{"limit":null,"limit_remaining":null}}`, ""},
		{"unreadable body", 200, `not json`, ""},
		{"exhausted", 200, `{"data":{"limit":50,"limit_remaining":0}}`, "key has no remaining credit"},
		{"overdrawn", 200, `{"data":{"limit":50,"limit_remaining":-1.5}}`, "key has no remaining credit"},
		{"unauthorized", 401, `{}`, "key rejected by openrouter"},
		{"forbidden", 403, `{}`, "key rejected by openrouter"},
		{"server error", 500, `{}`, "openrouter returned 500 Internal Server Error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if got := r.Header.Get("Authorization"); got != "Bearer k" {
					t.Errorf("Authorization = %q, want %q", got, "Bearer k")
				}
				w.WriteHeader(tt.status)
				w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			old := baseURL
			baseURL = srv.URL
			defer func() { baseURL = old }()

			err := ValidateKey(context.Background(), "k")
			switch {
			case tt.wantErr == "" && err != nil:
				t.Errorf("ValidateKey() = %v, want nil", err)
			case tt.wantErr != "" && err == nil:
				t.Errorf("ValidateKey() = nil, want %q", tt.wantErr)
			case tt.wantErr != "" && err.Error() != tt.wantErr:
				t.Errorf("ValidateKey() = %q, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestValidateKeyEmpty(t *testing.T) {
	if err := ValidateKey(context.Background(), ""); err == nil {
		t.Error("ValidateKey(\"\") = nil, want error")
	}
}
