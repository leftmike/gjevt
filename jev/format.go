package jev

import (
	"fmt"
	"math"
	"strconv"
)

func (a Answer) String() string {
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
