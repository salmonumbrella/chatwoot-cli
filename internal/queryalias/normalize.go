package queryalias

import (
	"fmt"
	"sort"
	"strings"
)

// Context identifies which parser rules to apply for alias normalization.
type Context int

const (
	// ContextQuery normalizes jq-style path segments in expressions.
	ContextQuery Context = iota + 1
	// ContextPath normalizes dot-delimited field paths (e.g. --fields, --sort).
	ContextPath
)

// Entry defines one alias -> canonical JSON key mapping.
type Entry struct {
	Alias     string
	Canonical string
}

var entries = []Entry{
	{Alias: "i", Canonical: "id"},
	{Alias: "n", Canonical: "name"},
	{Alias: "e", Canonical: "email"},
	{Alias: "st", Canonical: "status"},
	{Alias: "cst", Canonical: "st"},
	{Alias: "pr", Canonical: "priority"},
	{Alias: "ty", Canonical: "type"},
	{Alias: "ct", Canonical: "content"},
	{Alias: "tl", Canonical: "title"},
	{Alias: "ds", Canonical: "description"},
	{Alias: "ur", Canonical: "url"},
	{Alias: "sg", Canonical: "slug"},
	{Alias: "en", Canonical: "enabled"},
	{Alias: "act", Canonical: "actions"},
	{Alias: "it", Canonical: "items"},
	{Alias: "im", Canonical: "item"},
	{Alias: "rs", Canonical: "results"},
	{Alias: "pl", Canonical: "payload"},
	{Alias: "mt", Canonical: "meta"},
	{Alias: "kd", Canonical: "kind"},
	{Alias: "qe", Canonical: "query"},
	{Alias: "dt", Canonical: "data"},
	{Alias: "er", Canonical: "error"},
	{Alias: "sm", Canonical: "summary"},
	{Alias: "hm", Canonical: "has_more"},
	{Alias: "la", Canonical: "last_activity_at"},
	{Alias: "ca", Canonical: "created_at"},
	{Alias: "ua", Canonical: "updated_at"},
	{Alias: "ii", Canonical: "inbox_id"},
	{Alias: "ib", Canonical: "inbox"},
	{Alias: "ci", Canonical: "contact_id"},
	{Alias: "ai", Canonical: "assignee_id"},
	{Alias: "ti", Canonical: "team_id"},
	{Alias: "di", Canonical: "display_id"},
	{Alias: "cv", Canonical: "conversation_id"},
	{Alias: "mi", Canonical: "message_id"},
	{Alias: "mty", Canonical: "message_type"},
	{Alias: "uc", Canonical: "unread_count"},
	{Alias: "mc", Canonical: "messages_count"},
	{Alias: "tm", Canonical: "total_messages"},
	{Alias: "pn", Canonical: "phone_number"},
	{Alias: "lb", Canonical: "labels"},
	{Alias: "cu", Canonical: "custom_attributes"},
	{Alias: "blk", Canonical: "blacklist"},
	{Alias: "mtr", Canonical: "membership_tier"},
	{Alias: "oc", Canonical: "open_conversations"},
	{Alias: "tcv", Canonical: "total_conversations"},
	{Alias: "fc", Canonical: "first_contact"},
	{Alias: "lac", Canonical: "last_activity"},
	{Alias: "sd", Canonical: "sender"},
	{Alias: "sdi", Canonical: "sender_id"},
	{Alias: "sty", Canonical: "sender_type"},
	{Alias: "snm", Canonical: "sender_name"},
	{Alias: "ctp", Canonical: "content_type"},
	{Alias: "att", Canonical: "attachments"},
	{Alias: "ft", Canonical: "file_type"},
	{Alias: "fs", Canonical: "file_size"},
	{Alias: "du", Canonical: "data_url"},
	{Alias: "tu", Canonical: "thumb_url"},
	{Alias: "pv", Canonical: "private"},
	{Alias: "msg", Canonical: "message"},
	{Alias: "mgs", Canonical: "messages"},
	{Alias: "mg", Canonical: "msgs"},
	{Alias: "cvn", Canonical: "conversation"},
	{Alias: "ctc", Canonical: "contact"},
	{Alias: "ps", Canonical: "position"},
	{Alias: "aci", Canonical: "account_id"},
	{Alias: "ot", Canonical: "order_total"},
	{Alias: "os", Canonical: "order_status"},
	{Alias: "tp", Canonical: "total_pages"},
	{Alias: "cp", Canonical: "current_page"},
	{Alias: "pp", Canonical: "per_page"},
	{Alias: "tc", Canonical: "total_count"},
}

var (
	aliasToCanonical         = buildAliasToCanonical(entries)
	functionAliasToCanonical = map[string]string{
		"sl": "select",
		"ts": "test",
	}
)

func init() {
	if err := validateEntries(entries); err != nil {
		panic(err)
	}
	if err := validateFunctionAliases(functionAliasToCanonical); err != nil {
		panic(err)
	}
}

func buildAliasToCanonical(values []Entry) map[string]string {
	m := make(map[string]string, len(values))
	for _, v := range values {
		m[v.Alias] = v.Canonical
	}
	return m
}

func validateEntries(values []Entry) error {
	seenAlias := make(map[string]string, len(values))
	seenCanonical := make(map[string]string, len(values))

	for _, v := range values {
		if !isLowerIdentifier(v.Alias) {
			return fmt.Errorf("invalid alias %q: must be lowercase [a-z_][a-z0-9_]*", v.Alias)
		}
		if len(v.Alias) > 3 {
			return fmt.Errorf("invalid alias %q: alias length must be <= 3", v.Alias)
		}
		if !isLowerSnake(v.Canonical) {
			return fmt.Errorf("invalid canonical key %q: must be lowercase snake_case", v.Canonical)
		}
		if v.Alias == v.Canonical {
			return fmt.Errorf("invalid alias %q: alias must differ from canonical key", v.Alias)
		}
		if prev, ok := seenAlias[v.Alias]; ok {
			return fmt.Errorf("alias collision: %q maps to both %q and %q", v.Alias, prev, v.Canonical)
		}
		seenAlias[v.Alias] = v.Canonical
		if prev, ok := seenCanonical[v.Canonical]; ok {
			return fmt.Errorf("canonical key %q has multiple aliases: %q and %q", v.Canonical, prev, v.Alias)
		}
		seenCanonical[v.Canonical] = v.Alias
	}
	return nil
}

func validateFunctionAliases(values map[string]string) error {
	seenCanonical := make(map[string]string, len(values))
	for alias, canonical := range values {
		if !isLowerIdentifier(alias) {
			return fmt.Errorf("invalid function alias %q: must be lowercase [a-z_][a-z0-9_]*", alias)
		}
		if len(alias) > 3 {
			return fmt.Errorf("invalid function alias %q: alias length must be <= 3", alias)
		}
		if !isLowerIdentifier(canonical) {
			return fmt.Errorf("invalid canonical function %q: must be lowercase identifier", canonical)
		}
		if alias == canonical {
			return fmt.Errorf("invalid function alias %q: alias must differ from canonical function", alias)
		}
		if prev, ok := seenCanonical[canonical]; ok {
			return fmt.Errorf("canonical function %q has multiple aliases: %q and %q", canonical, prev, alias)
		}
		seenCanonical[canonical] = alias
	}
	return nil
}

func isLowerIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if i == 0 {
			if !isLowerLetter(ch) && ch != '_' {
				return false
			}
			continue
		}
		if !isLowerLetter(ch) && !isDigit(ch) && ch != '_' {
			return false
		}
	}
	return true
}

func isLowerSnake(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if !isLowerLetter(ch) && !isDigit(ch) && ch != '_' {
			return false
		}
	}
	return true
}

func isLowerLetter(ch byte) bool {
	return ch >= 'a' && ch <= 'z'
}

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

func canonicalizeToken(token string) string {
	if !isLowerIdentifier(token) {
		return token
	}
	if canonical, ok := aliasToCanonical[token]; ok {
		return canonical
	}
	return token
}

func canonicalizeFunctionToken(token string) string {
	if !isLowerIdentifier(token) {
		return token
	}
	if canonical, ok := functionAliasToCanonical[token]; ok {
		return canonical
	}
	return token
}

// Canonical returns a canonical key for an alias, if configured.
func Canonical(alias string) (string, bool) {
	canonical, ok := aliasToCanonical[alias]
	return canonical, ok
}

// Entries returns alias mappings sorted by alias.
func Entries() []Entry {
	out := append([]Entry(nil), entries...)
	sort.Slice(out, func(i, j int) bool {
		return out[i].Alias < out[j].Alias
	})
	return out
}

// Normalize rewrites configured aliases for supported query/path contexts.
func Normalize(input string, context Context) string {
	switch context {
	case ContextQuery:
		return normalizeQuery(input)
	case ContextPath:
		return normalizePath(input)
	default:
		return input
	}
}

func normalizePath(path string) string {
	if path == "" {
		return path
	}
	parts := strings.Split(path, ".")
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = canonicalizeToken(part)
	}
	return strings.Join(parts, ".")
}

func normalizeQuery(expr string) string {
	if expr == "" {
		return expr
	}

	var out strings.Builder
	out.Grow(len(expr))

	inString := false
	escaped := false
	inComment := false
	braceDepth := 0

	for i := 0; i < len(expr); {
		ch := expr[i]

		if inComment {
			out.WriteByte(ch)
			i++
			if ch == '\n' {
				inComment = false
			}
			continue
		}

		if inString {
			out.WriteByte(ch)
			i++
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}

		switch ch {
		case '"':
			inString = true
			out.WriteByte(ch)
			i++
		case '#':
			inComment = true
			out.WriteByte(ch)
			i++
		case '{':
			braceDepth++
			out.WriteByte(ch)
			i++
		case '}':
			if braceDepth > 0 {
				braceDepth--
			}
			out.WriteByte(ch)
			i++
		case '.':
			out.WriteByte(ch)
			i++
			if i < len(expr) && isLowerIdentifierStart(expr[i]) {
				start := i
				i++
				for i < len(expr) && isLowerIdentifierPart(expr[i]) {
					i++
				}
				token := expr[start:i]
				out.WriteString(canonicalizeToken(token))
				continue
			}
		default:
			// Bare tokens (not preceded by '.') are NOT alias-rewritten for path access.
			// They may be jq keywords (and, or, not, null, as, def, etc.).
			// Only function aliases (followed by '(') and jq object construction
			// shorthands (inside { }, not followed by ':') are rewritten here.
			if isLowerIdentifierStart(ch) {
				start := i
				i++
				for i < len(expr) && isLowerIdentifierPart(expr[i]) {
					i++
				}
				token := expr[start:i]
				if shouldRewriteFunctionAlias(expr, start, i) {
					out.WriteString(canonicalizeFunctionToken(token))
				} else if braceDepth > 0 && isJqShorthandKey(expr, i) {
					out.WriteString(canonicalizeToken(token))
				} else {
					out.WriteString(token)
				}
				continue
			}
			out.WriteByte(ch)
			i++
		}
	}

	return out.String()
}

func isLowerIdentifierStart(ch byte) bool {
	return isLowerLetter(ch) || ch == '_'
}

func isLowerIdentifierPart(ch byte) bool {
	return isLowerLetter(ch) || isDigit(ch) || ch == '_'
}

func shouldRewriteFunctionAlias(expr string, start, end int) bool {
	if start > 0 && expr[start-1] == '$' {
		return false
	}
	for i := end; i < len(expr); i++ {
		switch expr[i] {
		case ' ', '\t', '\n', '\r':
			continue
		case '(':
			return true
		default:
			return false
		}
	}
	return false
}

// isJqShorthandKey checks if a bare token at position end in expr is used as a jq
// object construction shorthand (i.e., {foo} not {foo: bar}).
// Returns true if the next non-whitespace character is , or } (not :).
func isJqShorthandKey(expr string, end int) bool {
	for j := end; j < len(expr); j++ {
		switch expr[j] {
		case ' ', '\t', '\n', '\r':
			continue
		case ',', '}':
			return true
		default:
			return false
		}
	}
	// End of expression inside braces — treat as shorthand.
	return true
}
