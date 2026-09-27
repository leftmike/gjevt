package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/leftmike/gjevt/jev"
)

type result struct {
	State     string `json:"state"`
	Truncated bool   `json:"truncated,omitempty"`
	*jev.DecisionResponse
}

type options struct {
	json bool
	skip bool
}

func main() {
	var opts options
	flag.BoolVar(&opts.json, "json", false, "print results as JSON")
	flag.BoolVar(&opts.skip, "skip", false, "skip state which is too long instead of truncating it")
	flag.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), "usage: gjevt [-json] [-skip] questions [state...]")
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(2)
	}

	cfg, err := loadConfig("gjevt.hcl")
	if err == nil {
		err = run(context.Background(), jev.NewClient(cfg), opts, flag.Arg(0), flag.Args()[1:],
			os.Stdin, os.Stdout, os.Stderr)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "gjevt:", err)
		os.Exit(1)
	}
}

func readFile(path string, stdin io.Reader) ([]byte, error) {
	if path == "-" {
		return io.ReadAll(stdin)
	}
	return os.ReadFile(path)
}

func readJSON(path string, stdin io.Reader, v any) error {
	b, err := readFile(path, stdin)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, v); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

func readState(path string, stdin io.Reader) (any, error) {
	b, err := readFile(path, stdin)
	if err != nil {
		return nil, err
	}
	b = bytes.TrimSpace(b)
	if !bytes.HasPrefix(b, []byte("{")) {
		return string(b), nil
	}
	var state any
	if err := json.Unmarshal(b, &state); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return state, nil
}

func run(ctx context.Context, c *jev.Client, opts options, questionsPath string,
	statePaths []string, stdin io.Reader, stdout, stderr io.Writer) error {

	var questions map[string]jev.Question
	if err := readJSON(questionsPath, stdin, &questions); err != nil {
		return err
	}

	if len(statePaths) == 0 {
		if questionsPath == "-" {
			return errors.New("questions and state cannot both be read from stdin")
		}
		statePaths = []string{"-"}
	}

	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	printed := false
	for _, path := range statePaths {
		state, err := readState(path, stdin)
		if err != nil {
			return err
		}

		req := jev.DecisionRequest{State: state, Questions: questions}
		var resp *jev.DecisionResponse
		var truncated bool
		if opts.skip {
			resp, err = c.Decide(ctx, req)
		} else {
			resp, truncated, err = c.DecideTruncated(ctx, req)
		}
		if errors.Is(err, jev.ErrTooLong) && opts.skip {
			fmt.Fprintf(stderr, "gjevt: %s: skipped: %s\n", path, err)
			continue
		} else if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}

		res := result{State: path, Truncated: truncated, DecisionResponse: resp}
		if opts.json {
			err = enc.Encode(res)
		} else {
			if printed {
				fmt.Fprintln(stdout)
			}
			err = printResult(stdout, res)
		}
		printed = true
		if err != nil {
			return err
		}
	}
	return nil
}
