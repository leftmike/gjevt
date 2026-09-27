package main

import (
	"fmt"
	"io"
	"slices"
	"text/tabwriter"
)

func printResult(w io.Writer, res result) error {
	if res.Truncated {
		fmt.Fprintln(w, res.State, "(truncated)")
	} else {
		fmt.Fprintln(w, res.State)
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	names := make([]string, 0, len(res.Answers))
	for name := range res.Answers {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		fmt.Fprintf(tw, "  %s\t%s\n", name, res.Answers[name])
	}
	if err := tw.Flush(); err != nil {
		return err
	}

	_, err := fmt.Fprintf(w, "  [%s: %d in, %d out, $%.7f]\n", res.Model, res.Usage.InputTokens,
		res.Usage.OutputTokens, res.Usage.Cost)
	return err
}
