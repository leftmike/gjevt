package jev

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const testResponse = `{
  "id": "gen-dec-1",
  "model": "typesafe/jev-1.13-20260917",
  "provider": "TypeSafe",
  "answers": {
    "angry": {"type": "noul", "noul": 0.18},
    "category": {"type": "choice", "choice": "billing", "probabilities": {"billing": 1}, "confidence": 1}
  },
  "usage": {"input_tokens": 441, "output_tokens": 70, "cost": 0.000018522}
}`

func newTestServer(t *testing.T, status int, body string, got *map[string]any) (*Client, func()) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/alpha/decisions" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-key" {
			t.Errorf("got Authorization %q", auth)
		}
		if got != nil {
			if err := json.NewDecoder(r.Body).Decode(got); err != nil {
				t.Error(err)
			}
		}
		w.WriteHeader(status)
		io.WriteString(w, body)
	}))
	c := NewClient(Config{APIKey: "test-key", BaseURL: srv.URL + "/api", Model: "default-model"},
		WithHTTPClient(srv.Client()))
	return c, srv.Close
}

func TestDecide(t *testing.T) {
	var got map[string]any
	c, done := newTestServer(t, http.StatusOK, testResponse, &got)
	defer done()

	resp, err := c.Decide(context.Background(), DecisionRequest{
		State: map[string]any{"ticket": "refund please"},
		Questions: map[string]Question{
			"angry": {Type: "noul", Instructions: "Is the customer angry?"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if got["model"] != "default-model" {
		t.Errorf("got model %v, want default-model", got["model"])
	}
	q := got["questions"].(map[string]any)["angry"].(map[string]any)
	if _, ok := q["criteria"]; ok {
		t.Errorf("criteria should be omitted: %v", q)
	}

	if a := resp.Answers["angry"]; a.Noul == nil || *a.Noul != 0.18 || a.Confidence != nil {
		t.Errorf("got angry %+v", a)
	}
	if a := resp.Answers["category"]; a.Choice != "billing" || a.Probabilities["billing"] != 1 {
		t.Errorf("got category %+v", a)
	}
	if resp.Usage.InputTokens != 441 {
		t.Errorf("got usage %+v", resp.Usage)
	}
}

func TestDecideModelOverride(t *testing.T) {
	var got map[string]any
	c, done := newTestServer(t, http.StatusOK, testResponse, &got)
	defer done()

	if _, err := c.Decide(context.Background(), DecisionRequest{Model: "typesafe/jev-1.13"}); err != nil {
		t.Fatal(err)
	}
	if got["model"] != "typesafe/jev-1.13" {
		t.Errorf("got model %v, want typesafe/jev-1.13", got["model"])
	}
}

func TestDecideHTTPError(t *testing.T) {
	c, done := newTestServer(t, http.StatusUnauthorized, `{"error":"bad key"}`, nil)
	defer done()

	_, err := c.Decide(context.Background(), DecisionRequest{})
	if err == nil || !strings.Contains(err.Error(), "401") || !strings.Contains(err.Error(), "bad key") {
		t.Fatalf("got error %v", err)
	}
}

func TestNewClientDefaults(t *testing.T) {
	c := NewClient(Config{APIKey: "k"})
	if c.cfg.BaseURL != DefaultBaseURL || c.cfg.Model != DefaultModel {
		t.Errorf("got %+v", c.cfg)
	}
	if c.httpClient != http.DefaultClient {
		t.Errorf("got http client %v", c.httpClient)
	}

	hc := &http.Client{}
	if c := NewClient(Config{}, WithHTTPClient(hc)); c.httpClient != hc {
		t.Errorf("got http client %v", c.httpClient)
	}
}

func ptr(f float64) *float64 { return &f }

func TestFormatAnswer(t *testing.T) {
	legend := map[string]string{"0": "low", "1": "medium", "2": "high"}
	cases := []struct {
		a    Answer
		want string
	}{
		{Answer{Type: "noul", Noul: ptr(0.99)}, "yes (0.99)"},
		{Answer{Type: "noul", Noul: ptr(0.5)}, "yes (0.50)"},
		{Answer{Type: "noul", Noul: ptr(0.07)}, "no (0.93)"},
		{Answer{Type: "noul", Noul: ptr(0)}, "no (1.00)"},
		{Answer{Type: "choice", Choice: "billing", Confidence: ptr(1)},
			"billing (confidence 1.00)"},
		{Answer{Type: "choice", Choice: "billing"}, "billing"},
		{Answer{Type: "score", Score: ptr(1), Legend: legend, Confidence: ptr(0.92)},
			"1: medium (confidence 0.92)"},
		{Answer{Type: "score", Score: ptr(1.6), Legend: legend}, "1.6: high"},
		{Answer{Type: "score", Score: ptr(0.09), Legend: legend}, "0.09: low"},
		{Answer{Type: "score", Score: ptr(7)}, "7"},
		{Answer{Type: "noul"}, "?"},
		{Answer{Type: "other"}, "?"},
	}
	for _, c := range cases {
		if got := c.a.String(); got != c.want {
			t.Errorf("String(%+v): got %q, want %q", c.a, got, c.want)
		}
	}
}
