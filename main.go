package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

func main() {
	cfg, err := loadConfig("gjevt.hcl")
	if err == nil {
		err = run(context.Background(), NewClient(cfg), os.Args[1:], os.Stdin, os.Stdout)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "gjevt:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, c *Client, args []string, stdin io.Reader, stdout io.Writer) error {
	in := stdin
	if len(args) > 0 {
		f, err := os.Open(args[0])
		if err != nil {
			return err
		}
		defer f.Close()
		in = f
	}

	var req DecisionRequest
	if err := json.NewDecoder(in).Decode(&req); err != nil {
		return err
	}

	resp, err := c.Decide(ctx, req)
	if err != nil {
		return err
	}

	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(resp)
}
