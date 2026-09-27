package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/leftmike/gjevt/jev"
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

func newTestServer(t *testing.T, status int, body string, got *map[string]any) (*jev.Client, func()) {
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
	c := jev.NewClient(jev.Config{APIKey: "test-key", BaseURL: srv.URL + "/api",
		Model: "default-model"}, jev.WithHTTPClient(srv.Client()))
	return c, srv.Close
}

func TestRun(t *testing.T) {
	var got map[string]any
	c, done := newTestServer(t, http.StatusOK, testResponse, &got)
	defer done()

	var out bytes.Buffer
	err := run(context.Background(), c, options{json: true}, "examples/support_questions.json",
		[]string{"examples/support_billing.json"}, nil, &out, io.Discard)
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
	err := run(context.Background(), c, options{json: true}, "examples/support_questions.json",
		[]string{"examples/support_billing.json", "examples/support_login.json"}, nil, &out,
		io.Discard)
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
	err := run(context.Background(), c, options{json: true}, "examples/support_questions.json", nil,
		strings.NewReader(`{"ticket": "help"}`), &out, io.Discard)
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
			err := run(context.Background(), c, options{}, tc.questions, tc.states,
				strings.NewReader(tc.stdin), io.Discard, io.Discard)
			if err == nil || !strings.Contains(err.Error(), tc.err) {
				t.Fatalf("got error %v, want %q", err, tc.err)
			}
		})
	}
}

func TestReadState(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want any
		err  bool
	}{
		{name: "text", in: "  hello world\n\n", want: "hello world"},
		{name: "multiline text", in: "\nline one\nline two\n", want: "line one\nline two"},
		{name: "json", in: "\n\t{\"a\": 1}\n", want: map[string]any{"a": 1.0}},
		{name: "array is text", in: "[1, 2]", want: "[1, 2]"},
		{name: "bad json", in: "{not json", err: true},
		{name: "empty", in: " \n ", want: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := readState("-", strings.NewReader(tc.in))
			if tc.err {
				if err == nil {
					t.Fatalf("got %v, want error", got)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestRunTextState(t *testing.T) {
	var got map[string]any
	c, done := newTestServer(t, http.StatusOK, testResponse, &got)
	defer done()

	err := run(context.Background(), c, options{}, "examples/support_questions.json",
		[]string{"examples/support_refund.txt"}, nil, io.Discard, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	s, ok := got["state"].(string)
	if !ok || !strings.HasPrefix(s, "Subject:") || strings.HasSuffix(s, "\n") {
		t.Errorf("got state %#v", got["state"])
	}
}
