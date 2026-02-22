package queryalias

import "testing"

func TestEntriesValidity(t *testing.T) {
	values := Entries()
	if len(values) == 0 {
		t.Fatal("Entries() must not be empty")
	}

	aliasSeen := make(map[string]struct{}, len(values))
	canonicalSeen := make(map[string]struct{}, len(values))
	for _, entry := range values {
		if entry.Alias == "" || entry.Canonical == "" {
			t.Fatalf("empty alias entry: %+v", entry)
		}
		if len(entry.Alias) > 3 {
			t.Fatalf("alias %q exceeds 3 characters", entry.Alias)
		}
		if entry.Alias == entry.Canonical {
			t.Fatalf("alias %q must differ from canonical %q", entry.Alias, entry.Canonical)
		}
		if _, ok := aliasSeen[entry.Alias]; ok {
			t.Fatalf("duplicate alias %q", entry.Alias)
		}
		aliasSeen[entry.Alias] = struct{}{}
		if _, ok := canonicalSeen[entry.Canonical]; ok {
			t.Fatalf("duplicate canonical key %q", entry.Canonical)
		}
		canonicalSeen[entry.Canonical] = struct{}{}
	}
}

func TestCanonical(t *testing.T) {
	tests := []struct {
		alias string
		want  string
		ok    bool
	}{
		{alias: "st", want: "status", ok: true},
		{alias: "cst", want: "st", ok: true},
		{alias: "ib", want: "inbox", ok: true},
		{alias: "mg", want: "msgs", ok: true},
		{alias: "la", want: "last_activity_at", ok: true},
		{alias: "sd", want: "sender", ok: true},
		{alias: "mty", want: "message_type", ok: true},
		{alias: "blk", want: "blacklist", ok: true},
		{alias: "mtr", want: "membership_tier", ok: true},
		{alias: "tp", want: "total_pages", ok: true},
		{alias: "cp", want: "current_page", ok: true},
		{alias: "pp", want: "per_page", ok: true},
		{alias: "tc", want: "total_count", ok: true},
		{alias: "missing", want: "", ok: false},
	}

	for _, tt := range tests {
		got, ok := Canonical(tt.alias)
		if ok != tt.ok {
			t.Fatalf("Canonical(%q) ok=%v, want %v", tt.alias, ok, tt.ok)
		}
		if got != tt.want {
			t.Fatalf("Canonical(%q)=%q, want %q", tt.alias, got, tt.want)
		}
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "single alias", in: "st", want: "status"},
		{name: "light status alias", in: "cst", want: "st"},
		{name: "light inbox alias", in: "ib", want: "inbox"},
		{name: "light messages alias", in: "mg", want: "msgs"},
		{name: "nested path", in: "cu.plan", want: "custom_attributes.plan"},
		{name: "custom attr aliases", in: "cu.blk", want: "custom_attributes.blacklist"},
		{name: "multiple aliases", in: "ci.la", want: "contact_id.last_activity_at"},
		{name: "meta sender id", in: "mt.sd.i", want: "meta.sender.id"},
		{name: "meta sender name", in: "mt.sd.n", want: "meta.sender.name"},
		{name: "pagination total pages", in: "mt.tp", want: "meta.total_pages"},
		{name: "pagination current page", in: "mt.cp", want: "meta.current_page"},
		{name: "long form unchanged", in: "last_activity_at", want: "last_activity_at"},
		{name: "mixed case unchanged", in: "St", want: "St"},
		{name: "unknown unchanged", in: "unknown_key", want: "unknown_key"},
		{name: "empty input", in: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Normalize(tt.in, ContextPath)
			if got != tt.want {
				t.Fatalf("Normalize(path, %q)=%q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestNormalizeQuery(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "basic dot paths",
			in:   `.it[] | select(.st == "open") | .i`,
			want: `.items[] | select(.status == "open") | .id`,
		},
		{
			name: "nested path aliases",
			in:   `.it[0].cu.plan`,
			want: `.items[0].custom_attributes.plan`,
		},
		{
			name: "message aliases with nested sender",
			in:   `.it[] | select(.mty == 1) | .sd.n`,
			want: `.items[] | select(.message_type == 1) | .sender.name`,
		},
		{
			name: "custom attribute aliases",
			in:   `.it[] | sl(.cu.blk == true and .cu.mtr != null) | .i`,
			want: `.items[] | select(.custom_attributes.blacklist == true and .custom_attributes.membership_tier != null) | .id`,
		},
		{
			name: "function aliases select and test",
			in:   `.it[] | sl(.mty == 1) | sl(.ct | ts("refund"; "i"))`,
			want: `.items[] | select(.message_type == 1) | select(.content | test("refund"; "i"))`,
		},
		{
			name: "recursive descent",
			in:   `..it | .la`,
			want: `..items | .last_activity_at`,
		},
		{
			name: "quoted bracket key preserved",
			in:   `.it[0]["st"]`,
			want: `.items[0]["st"]`,
		},
		{
			name: "mixed case token preserved",
			in:   `.St | .IT | .st`,
			want: `.St | .IT | .status`,
		},
		{
			name: "strings and comments preserved",
			in:   ".st as $x | \"keep .st and #comment\" # .st alias here\n.it",
			want: ".status as $x | \"keep .st and #comment\" # .st alias here\n.items",
		},
		{
			name: "unknown token unchanged",
			in:   `.unknown_key | .st`,
			want: `.unknown_key | .status`,
		},
		{
			name: "quoted keys only",
			in:   `.["st"] | .["it"]`,
			want: `.["st"] | .["it"]`,
		},
		{
			name: "variables are not rewritten as function aliases",
			in:   `.it[] | $sl | .st`,
			want: `.items[] | $sl | .status`,
		},
		{
			name: "del builtin preserved as bare token",
			in:   `.payload | del(.temp)`,
			want: `.payload | del(.temp)`,
		},
		{
			name: "shorthand single key",
			in:   `{i}`,
			want: `{id}`,
		},
		{
			name: "shorthand multiple keys",
			in:   `{i, n}`,
			want: `{id, name}`,
		},
		{
			name: "shorthand mixed with dot path",
			in:   `{i, s: .st}`,
			want: `{id, s: .status}`,
		},
		{
			name: "shorthand in pipeline",
			in:   `.it[] | {i, st, ct}`,
			want: `.items[] | {id, status, content}`,
		},
		{
			name: "light payload aliases",
			in:   `{i, cst, ib, ctc, ls: .mg[-1]}`,
			want: `{id, st, inbox, contact, ls: .msgs[-1]}`,
		},
		{
			name: "key-value pair key not rewritten",
			in:   `{i: .st}`,
			want: `{i: .status}`,
		},
		{
			name: "key-value pair string value",
			in:   `{n: "hello"}`,
			want: `{n: "hello"}`,
		},
		{
			name: "shorthand nested braces",
			in:   `{a: {i}}`,
			want: `{a: {id}}`,
		},
		{
			name: "shorthand unknown token unchanged",
			in:   `{foo}`,
			want: `{foo}`,
		},
		{
			name: "meta sender id and name",
			in:   `.mt.sd.i, .mt.sd.n`,
			want: `.meta.sender.id, .meta.sender.name`,
		},
		{
			name: "pagination metadata",
			in:   `.mt.tp, .mt.cp, .mt.pp, .mt.tc`,
			want: `.meta.total_pages, .meta.current_page, .meta.per_page, .meta.total_count`,
		},
		{
			name: "empty input",
			in:   "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Normalize(tt.in, ContextQuery)
			if got != tt.want {
				t.Fatalf("Normalize(query, %q)=%q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestNormalizeUnknownContext(t *testing.T) {
	in := `.st`
	got := Normalize(in, Context(999))
	if got != in {
		t.Fatalf("Normalize with unknown context rewrote input: got %q want %q", got, in)
	}
}
