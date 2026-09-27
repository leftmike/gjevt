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
	"unicode/utf8"
)

func TestCutString(t *testing.T) {
	cases := []struct {
		s    string
		n    int
		want string
	}{
		{"hello", 3, "hel"},
		{"hello", 0, ""},
		{"héllo", 2, "h"},
		{"héllo", 3, "hé"},
		{"日本語", 4, "日"},
	}
	for _, c := range cases {
		if got := cutString(c.s, c.n); got != c.want {
			t.Errorf("cutString(%q, %d): got %q, want %q", c.s, c.n, got, c.want)
		}
	}
}

func TestTruncateState(t *testing.T) {
	got, shrunk := truncateState("short", 100)
	if got != "short" || shrunk {
		t.Errorf("got %q, %v", got, shrunk)
	}

	got, shrunk = truncateState(strings.Repeat("é", 100), 50)
	if s := got.(string); !shrunk || stateSize(s) > 50 || !utf8.ValidString(s) {
		t.Errorf("got %q, %v", s, shrunk)
	}

	state := map[string]any{
		"subject": "hello",
		"body":    strings.Repeat("x", 1000),
		"parts":   []any{strings.Repeat("y", 500), 42.0},
	}
	got, shrunk = truncateState(state, 400)
	m := got.(map[string]any)
	if !shrunk || stateSize(m) > 400 {
		t.Fatalf("got size %d, %v", stateSize(m), shrunk)
	}
	if m["subject"] != "hello" || m["parts"].([]any)[1] != 42.0 {
		t.Errorf("got %v", m)
	}

	got, shrunk = truncateState(map[string]any{"a": 1.0, "b": ""}, 1)
	if shrunk {
		t.Errorf("got %v, %v", got, shrunk)
	}
}

func newLimitServer(t *testing.T, limit int) *Client {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req DecisionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
		}
		if stateSize(req.State) > limit {
			w.WriteHeader(http.StatusBadRequest)
			io.WriteString(w,
				`{"error":{"message":"HTTP 400: {\"detail\":{\"error_type\":\"max_tokens_exceeded\"}}","code":400}}`)
			return
		}
		io.WriteString(w, testResponse)
	}))
	t.Cleanup(srv.Close)
	c := NewClient(Config{APIKey: "test-key", BaseURL: srv.URL + "/api"})
	c.httpClient = srv.Client()
	return c
}

func TestDecideTooLong(t *testing.T) {
	c := newLimitServer(t, 1000)

	_, err := c.Decide(context.Background(), DecisionRequest{State: strings.Repeat("x", 2000)})
	if err != ErrTooLong {
		t.Fatalf("got error %v, want ErrTooLong", err)
	}

	_, truncated, err := decide(context.Background(), c, nil, strings.Repeat("x", 2000), true)
	if err != ErrTooLong || truncated {
		t.Fatalf("got %v, %v; want ErrTooLong", truncated, err)
	}

	_, truncated, err = decide(context.Background(), c, nil, strings.Repeat("x", 200*1024), false)
	if err != nil || !truncated {
		t.Fatalf("got %v, %v", truncated, err)
	}

	_, truncated, err = decide(context.Background(), c, nil, "short", false)
	if err != nil || truncated {
		t.Fatalf("got %v, %v", truncated, err)
	}

	c = newLimitServer(t, 0)
	if _, _, err = decide(context.Background(), c, nil, "abc", false); err != ErrTooLong {
		t.Fatalf("got error %v, want ErrTooLong", err)
	}
}

func TestRunTruncateAndSkip(t *testing.T) {
	c := newLimitServer(t, 100)

	var out bytes.Buffer
	err := run(context.Background(), c, options{}, "examples/support_questions.json",
		[]string{"examples/support_refund.txt", "examples/support_login.json"}, nil, &out,
		io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out.String(), "examples/support_refund.txt (truncated)\n") ||
		!strings.Contains(out.String(), "\nexamples/support_login.json (truncated)\n") {
		t.Errorf("got:\n%s", out.String())
	}

	out.Reset()
	var errOut bytes.Buffer
	c = newLimitServer(t, 184)
	err = run(context.Background(), c, options{json: true, skip: true},
		"examples/support_questions.json",
		[]string{"examples/support_refund.txt", "examples/support_login.json"}, nil, &out,
		&errOut)
	if err != nil {
		t.Fatal(err)
	}
	if errOut.String() != "gjevt: examples/support_login.json: skipped: state is too long\n" {
		t.Errorf("got stderr %q", errOut.String())
	}
	var res result
	if err := json.Unmarshal(out.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	if res.State != "examples/support_refund.txt" || res.Truncated {
		t.Errorf("got %+v", res)
	}
}
