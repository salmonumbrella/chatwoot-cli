package resolve_test

import (
	"errors"
	"testing"

	"github.com/chatwoot/chatwoot-cli/internal/resolve"
)

func TestFuzzyMatch_ExactHit(t *testing.T) {
	items := []resolve.Named{
		{ID: 1, Name: "Support Inbox"},
		{ID: 2, Name: "Sales Inbox"},
	}
	id, err := resolve.FuzzyMatch("Support Inbox", items)
	if err != nil {
		t.Fatal(err)
	}
	if id != 1 {
		t.Fatalf("expected ID 1, got %d", id)
	}
}

func TestFuzzyMatch_PartialHit(t *testing.T) {
	items := []resolve.Named{
		{ID: 1, Name: "Support Inbox"},
		{ID: 2, Name: "Sales Inbox"},
	}
	id, err := resolve.FuzzyMatch("supp", items)
	if err != nil {
		t.Fatal(err)
	}
	if id != 1 {
		t.Fatalf("expected ID 1, got %d", id)
	}
}

func TestFuzzyMatch_CaseInsensitive(t *testing.T) {
	items := []resolve.Named{
		{ID: 1, Name: "Support Inbox"},
	}
	id, err := resolve.FuzzyMatch("SUPPORT", items)
	if err != nil {
		t.Fatal(err)
	}
	if id != 1 {
		t.Fatalf("expected ID 1, got %d", id)
	}
}

func TestFuzzyMatch_NoMatch(t *testing.T) {
	items := []resolve.Named{
		{ID: 1, Name: "Support Inbox"},
	}
	_, err := resolve.FuzzyMatch("billing", items)
	if err == nil {
		t.Fatal("expected error for no match")
	}
}

func TestFuzzyMatch_Ambiguous(t *testing.T) {
	items := []resolve.Named{
		{ID: 1, Name: "Support US"},
		{ID: 2, Name: "Support EU"},
	}
	_, err := resolve.FuzzyMatch("support", items)
	if err == nil {
		t.Fatal("expected ambiguity error")
	}
	var ae *resolve.AmbiguousError
	if !errors.As(err, &ae) {
		t.Fatalf("expected AmbiguousError, got %T: %v", err, err)
	}
	if len(ae.Matches) == 0 {
		t.Fatalf("expected candidates in ambiguity error: %+v", ae)
	}
}

func TestFuzzyMatch_PrefersExactOverFuzzy(t *testing.T) {
	items := []resolve.Named{
		{ID: 1, Name: "Sales"},
		{ID: 2, Name: "Sales Inbox"},
	}
	id, err := resolve.FuzzyMatch("Sales", items)
	if err != nil {
		t.Fatal(err)
	}
	if id != 1 {
		t.Fatalf("expected exact match ID 1, got %d", id)
	}
}

func TestFuzzyMatch_EmptyQuery(t *testing.T) {
	items := []resolve.Named{{ID: 1, Name: "Support"}}
	_, err := resolve.FuzzyMatch("", items)
	if err == nil {
		t.Fatal("expected error for empty query")
	}
}

func TestFuzzyMatch_EmptyItems(t *testing.T) {
	_, err := resolve.FuzzyMatch("support", nil)
	if err == nil {
		t.Fatal("expected error for empty items")
	}
}

func TestFuzzyMatchAll_ReturnsRanked(t *testing.T) {
	items := []resolve.Named{
		{ID: 1, Name: "Support Inbox"},
		{ID: 2, Name: "Sales Inbox"},
		{ID: 3, Name: "Shipping"},
	}
	matches := resolve.FuzzyMatchAll("s", items, 10)
	if len(matches) == 0 {
		t.Fatal("expected at least one match")
	}
	for _, m := range matches {
		if m.ID == 0 {
			t.Fatal("match should have non-zero ID")
		}
	}
}
