package main

import (
	"encoding/json"
	"unicode/utf8"
)

const maxStateSize = 96 * 1024

func stateSize(state any) int {
	b, _ := json.Marshal(state)
	return len(b)
}

type leaf struct {
	s   string
	set func(string)
}

func collectLeaves(v any, set func(any), leaves []leaf) []leaf {
	switch v := v.(type) {
	case string:
		return append(leaves, leaf{v, func(s string) { set(s) }})
	case map[string]any:
		for k, x := range v {
			leaves = collectLeaves(x, func(n any) { v[k] = n }, leaves)
		}
	case []any:
		for i, x := range v {
			leaves = collectLeaves(x, func(n any) { v[i] = n }, leaves)
		}
	}
	return leaves
}

func cutString(s string, n int) string {
	for n > 0 && !utf8.RuneStart(s[n]) {
		n -= 1
	}
	return s[:n]
}

func truncateState(state any, target int) (any, bool) {
	shrunk := false
	for {
		size := stateSize(state)
		if size <= target {
			return state, shrunk
		}

		leaves := collectLeaves(state, func(n any) { state = n }, nil)
		var longest *leaf
		for i := range leaves {
			if longest == nil || len(leaves[i].s) > len(longest.s) {
				longest = &leaves[i]
			}
		}
		if longest == nil || longest.s == "" {
			return state, shrunk
		}
		longest.set(cutString(longest.s, max(len(longest.s)-(size-target), 0)))
		shrunk = true
	}
}
