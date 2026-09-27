package main

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"
)

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
		if got := formatAnswer(c.a); got != c.want {
			t.Errorf("formatAnswer(%+v): got %q, want %q", c.a, got, c.want)
		}
	}
}

func TestRunPretty(t *testing.T) {
	c, done := newTestServer(t, http.StatusOK, testResponse, nil)
	defer done()

	var out bytes.Buffer
	err := run(context.Background(), c, options{}, "examples/support_questions.json",
		[]string{"examples/support_billing.json", "examples/support_refund.txt"}, nil, &out,
		io.Discard)
	if err != nil {
		t.Fatal(err)
	}

	want := `examples/support_billing.json
  angry     no (0.82)
  category  billing (confidence 1.00)
  [typesafe/jev-1.13-20260917: 441 in, 70 out, $0.0000185]

examples/support_refund.txt
  angry     no (0.82)
  category  billing (confidence 1.00)
  [typesafe/jev-1.13-20260917: 441 in, 70 out, $0.0000185]
`
	if got := out.String(); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}
