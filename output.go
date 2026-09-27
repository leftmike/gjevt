package main

import (
	"fmt"
	"io"
	"math"
	"slices"
	"strconv"
	"text/tabwriter"
)

func formatAnswer(a Answer) string {
	switch a.Type {
	case "noul":
		if a.Noul == nil {
			break
		}
		if *a.Noul >= 0.5 {
			return fmt.Sprintf("yes (%.2f)", *a.Noul)
		}
		return fmt.Sprintf("no (%.2f)", 1-*a.Noul)
	case "choice":
		return a.Choice + formatConfidence(a.Confidence)
	case "score":
		if a.Score == nil {
			break
		}
		s := strconv.FormatFloat(*a.Score, 'f', -1, 64)
		if label, ok := a.Legend[strconv.Itoa(int(math.Round(*a.Score)))]; ok {
			s += ": " + label
		}
		return s + formatConfidence(a.Confidence)
	}
	return "?"
}

func formatConfidence(c *float64) string {
	if c == nil {
		return ""
	}
	return fmt.Sprintf(" (confidence %.2f)", *c)
}

func printResult(w io.Writer, res result) error {
	fmt.Fprintln(w, res.State)

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	names := make([]string, 0, len(res.Answers))
	for name := range res.Answers {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		fmt.Fprintf(tw, "  %s\t%s\n", name, formatAnswer(res.Answers[name]))
	}
	if err := tw.Flush(); err != nil {
		return err
	}

	_, err := fmt.Fprintf(w, "  [%s: %d in, %d out, $%.7f]\n", res.Model, res.Usage.InputTokens,
		res.Usage.OutputTokens, res.Usage.Cost)
	return err
}
