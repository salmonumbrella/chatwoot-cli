// Package resolve provides fuzzy name-to-ID matching for Chatwoot resources.
package resolve

import (
	"errors"
	"fmt"
	"strings"

	"github.com/sahilm/fuzzy"
)

// Named represents any resource with an ID and display name.
type Named struct {
	ID   int
	Name string
}

// Match is a fuzzy match result with score.
type Match struct {
	ID    int
	Name  string
	Score int
}

var (
	ErrEmptyQuery = errors.New("empty search query")
	ErrEmptyItems = errors.New("no items to match against")
)

// AmbiguousError indicates multiple candidates matched equally well.
// Matches are sorted best-first and capped (see FuzzyMatch / FuzzyMatchAll).
type AmbiguousError struct {
	Query   string
	Matches []Match
}

func (e *AmbiguousError) Error() string {
	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "ambiguous match for %q", e.Query)
	if len(e.Matches) > 0 {
		b.WriteString(", candidates:")
		for _, m := range e.Matches {
			_, _ = fmt.Fprintf(&b, "\n  %d: %s", m.ID, m.Name)
		}
	}
	return b.String()
}

type namedSourceLower []Named

func (s namedSourceLower) String(i int) string { return strings.ToLower(s[i].Name) }
func (s namedSourceLower) Len() int            { return len(s) }

// FuzzyMatch finds the best matching item by name and returns its ID.
//
// Behavior:
// - Empty query or empty items are errors.
// - Prefers exact case-insensitive matches over fuzzy matches.
// - Case-insensitive fuzzy matching.
// - If the top two fuzzy results tie on score, returns *AmbiguousError.
func FuzzyMatch(query string, items []Named) (int, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return 0, ErrEmptyQuery
	}
	if len(items) == 0 {
		return 0, ErrEmptyItems
	}

	// Exact case-insensitive match first (kubectl-style: exact wins).
	for _, item := range items {
		if strings.EqualFold(item.Name, query) {
			return item.ID, nil
		}
	}

	results := fuzzy.FindFrom(strings.ToLower(query), namedSourceLower(items))
	if len(results) == 0 {
		return 0, fmt.Errorf("no match found for %q", query)
	}
	if len(results) > 1 && results[0].Score == results[1].Score {
		return 0, &AmbiguousError{
			Query:   query,
			Matches: buildMatches(items, results, 5),
		}
	}
	return items[results[0].Index].ID, nil
}

// FuzzyMatchAll returns up to limit matches ranked by score (best first).
func FuzzyMatchAll(query string, items []Named, limit int) []Match {
	query = strings.TrimSpace(query)
	if query == "" || len(items) == 0 || limit <= 0 {
		return nil
	}

	results := fuzzy.FindFrom(strings.ToLower(query), namedSourceLower(items))
	return buildMatches(items, results, limit)
}

func buildMatches(items []Named, results fuzzy.Matches, limit int) []Match {
	if len(results) == 0 || limit <= 0 {
		return nil
	}
	if len(results) > limit {
		results = results[:limit]
	}
	matches := make([]Match, len(results))
	for i, r := range results {
		matches[i] = Match{
			ID:    items[r.Index].ID,
			Name:  items[r.Index].Name,
			Score: r.Score,
		}
	}
	return matches
}
