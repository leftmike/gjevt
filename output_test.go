package main

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"
)

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
