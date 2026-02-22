package cmd

import "strings"

// levenshtein computes the Levenshtein edit distance between two strings.
func levenshtein(a, b string) int {
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	// Use a single row plus a prev value to reduce allocation.
	row := make([]int, lb+1)
	for j := range row {
		row[j] = j
	}

	for i := 1; i <= la; i++ {
		prev := i - 1
		row[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			val := min3(row[j]+1, row[j-1]+1, prev+cost)
			prev = row[j]
			row[j] = val
		}
	}
	return row[lb]
}

func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// suggestCommand finds the closest command name to the unknown input.
// Returns empty string if no close match (distance > 3).
func suggestCommand(unknown string, commands []string) string {
	unknown = strings.ToLower(unknown)
	bestDist := 4 // threshold: only suggest if distance <= 3
	bestMatch := ""
	for _, cmd := range commands {
		d := levenshtein(unknown, strings.ToLower(cmd))
		if d < bestDist {
			bestDist = d
			bestMatch = cmd
		}
	}
	return bestMatch
}

// suggestFlag finds the closest flag name to the unknown input.
// Strips leading dashes from both the input and the known flags for comparison,
// but returns the match with its original prefix.
func suggestFlag(unknown string, flags []string) string {
	stripped := strings.TrimLeft(unknown, "-")
	if stripped == "" {
		return ""
	}
	stripped = strings.ToLower(stripped)
	bestDist := 4
	bestMatch := ""
	for _, f := range flags {
		fStripped := strings.TrimLeft(f, "-")
		d := levenshtein(stripped, strings.ToLower(fStripped))
		if d < bestDist {
			bestDist = d
			bestMatch = f
		}
	}
	return bestMatch
}
