package main

import (
	"bytes"
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
	c := NewClient(Config{APIKey: "test-key", BaseURL: srv.URL + "/api", Model: "default-model"})
	c.httpClient = srv.Client()
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

func TestRun(t *testing.T) {
	var got map[string]any
	c, done := newTestServer(t, http.StatusOK, testResponse, &got)
	defer done()

	var out bytes.Buffer
	err := run(context.Background(), c, "examples/support_questions.json",
		[]string{"examples/support_billing.json"}, nil, &out)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got["questions"].(map[string]any)["urgency"]; !ok {
		t.Errorf("got questions %v", got["questions"])
	}
	if _, ok := got["state"].(map[string]any)["ticket"]; !ok {
		t.Errorf("got state %v", got["state"])
	}

	var res result
	if err := json.Unmarshal(out.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	if res.State != "examples/support_billing.json" || res.ID != "gen-dec-1" {
		t.Errorf("got %+v", res)
	}
}

func TestRunMultipleStates(t *testing.T) {
	c, done := newTestServer(t, http.StatusOK, testResponse, nil)
	defer done()

	var out bytes.Buffer
	err := run(context.Background(), c, "examples/support_questions.json",
		[]string{"examples/support_billing.json", "examples/support_login.json"}, nil, &out)
	if err != nil {
		t.Fatal(err)
	}

	dec := json.NewDecoder(&out)
	var states []string
	for dec.More() {
		var res result
		if err := dec.Decode(&res); err != nil {
			t.Fatal(err)
		}
		states = append(states, res.State)
	}
	if len(states) != 2 || states[0] != "examples/support_billing.json" ||
		states[1] != "examples/support_login.json" {
		t.Errorf("got states %v", states)
	}
}

func TestRunStdin(t *testing.T) {
	var got map[string]any
	c, done := newTestServer(t, http.StatusOK, testResponse, &got)
	defer done()

	var out bytes.Buffer
	err := run(context.Background(), c, "examples/support_questions.json", nil,
		strings.NewReader(`{"ticket": "help"}`), &out)
	if err != nil {
		t.Fatal(err)
	}
	if got["state"].(map[string]any)["ticket"] != "help" {
		t.Errorf("got state %v", got["state"])
	}
	if !strings.Contains(out.String(), `"state": "-"`) {
		t.Errorf("got %s", out.String())
	}
}

func TestRunErrors(t *testing.T) {
	c, done := newTestServer(t, http.StatusOK, testResponse, nil)
	defer done()

	cases := []struct {
		name      string
		questions string
		states    []string
		stdin     string
		err       string
	}{
		{name: "bad stdin", questions: "examples/support_questions.json", stdin: "{not json",
			err: "-:"},
		{name: "both stdin", questions: "-", stdin: "{}", err: "both"},
		{name: "missing state", questions: "examples/support_questions.json",
			states: []string{"missing.json"}, err: "missing.json"},
		{name: "missing questions", questions: "missing.json", err: "missing.json"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := run(context.Background(), c, tc.questions, tc.states,
				strings.NewReader(tc.stdin), io.Discard)
			if err == nil || !strings.Contains(err.Error(), tc.err) {
				t.Fatalf("got error %v, want %q", err, tc.err)
			}
		})
	}
}
