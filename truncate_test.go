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

	"github.com/leftmike/gjevt/jev"
)

func newLimitServer(t *testing.T, limit int) *jev.Client {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			State json.RawMessage `json:"state"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
		}
		if len(req.State) > limit {
			w.WriteHeader(http.StatusBadRequest)
			io.WriteString(w,
				`{"error":{"message":"HTTP 400: {\"detail\":{\"error_type\":\"max_tokens_exceeded\"}}","code":400}}`)
			return
		}
		io.WriteString(w, testResponse)
	}))
	t.Cleanup(srv.Close)
	return jev.NewClient(jev.Config{APIKey: "test-key", BaseURL: srv.URL + "/api"},
		jev.WithHTTPClient(srv.Client()))
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
