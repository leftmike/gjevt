package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

type result struct {
	State string `json:"state"`
	*DecisionResponse
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: gjevt questions [state...]")
		os.Exit(2)
	}

	cfg, err := loadConfig("gjevt.hcl")
	if err == nil {
		err = run(context.Background(), NewClient(cfg), os.Args[1], os.Args[2:], os.Stdin,
			os.Stdout)
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

func run(ctx context.Context, c *Client, questionsPath string, statePaths []string,
	stdin io.Reader, stdout io.Writer) error {

	var questions map[string]Question
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
	for _, path := range statePaths {
		state, err := readState(path, stdin)
		if err != nil {
			return err
		}

		resp, err := c.Decide(ctx, DecisionRequest{State: state, Questions: questions})
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		if err := enc.Encode(result{State: path, DecisionResponse: resp}); err != nil {
			return err
		}
	}
	return nil
}
