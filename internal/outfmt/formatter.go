package outfmt

import (
	"context"
	"fmt"
	"io"
	"text/tabwriter"
)

// Formatter handles output formatting for commands.
type Formatter struct {
	ctx       context.Context
	out       io.Writer
	errOut    io.Writer
	tabWriter *tabwriter.Writer
}

// NewFormatter creates a new Formatter
func NewFormatter(ctx context.Context, out, errOut io.Writer) *Formatter {
	return &Formatter{
		ctx:       ctx,
		out:       out,
		errOut:    errOut,
		tabWriter: tabwriter.NewWriter(out, 0, 4, 2, ' ', 0),
	}
}

// Output writes data as JSON or text based on context format.
func (f *Formatter) Output(data any) error {
	if IsJSON(f.ctx) {
		query := GetQuery(f.ctx)
		light := IsLight(f.ctx)
		if tmpl := GetTemplate(f.ctx); tmpl != "" {
			var filtered any
			var err error
			if light {
				filtered, err = ApplyQueryLiteral(data, query)
			} else {
				filtered, err = ApplyQuery(data, query)
			}
			if err != nil {
				return err
			}
			return WriteTemplate(f.out, filtered, tmpl)
		}
		if light {
			return WriteJSONFilteredLiteral(f.out, data, query, IsCompact(f.ctx))
		}
		return WriteJSONFiltered(f.out, data, query, IsCompact(f.ctx))
	}
	return nil
}

// StartTable writes table headers. Returns true if in text mode.
func (f *Formatter) StartTable(headers []string) bool {
	if IsJSON(f.ctx) {
		return false
	}

	for i, h := range headers {
		if i > 0 {
			_, _ = fmt.Fprint(f.tabWriter, "\t")
		}
		_, _ = fmt.Fprint(f.tabWriter, h)
	}
	_, _ = fmt.Fprintln(f.tabWriter)
	return true
}

// Row writes a single row to the table.
func (f *Formatter) Row(columns ...string) {
	for i, col := range columns {
		if i > 0 {
			_, _ = fmt.Fprint(f.tabWriter, "\t")
		}
		_, _ = fmt.Fprint(f.tabWriter, col)
	}
	_, _ = fmt.Fprintln(f.tabWriter)
}

// EndTable flushes the table output.
func (f *Formatter) EndTable() error {
	return f.tabWriter.Flush()
}

// Empty writes a message to stderr indicating no results.
func (f *Formatter) Empty(message string) {
	_, _ = fmt.Fprintln(f.errOut, message)
}
