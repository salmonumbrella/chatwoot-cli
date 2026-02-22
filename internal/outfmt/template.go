package outfmt

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"text/template"
)

type templateKey struct{}

// WithTemplate adds a template string to the context
func WithTemplate(ctx context.Context, tmpl string) context.Context {
	return context.WithValue(ctx, templateKey{}, tmpl)
}

// GetTemplate retrieves the template string from context
func GetTemplate(ctx context.Context) string {
	if tmpl, ok := ctx.Value(templateKey{}).(string); ok {
		return tmpl
	}
	return ""
}

// WriteTemplate renders data using a Go text/template string
func WriteTemplate(w io.Writer, v any, tmpl string) error {
	funcs := template.FuncMap{
		"json": func(val any) (string, error) {
			buf := &bytes.Buffer{}
			enc := json.NewEncoder(buf)
			enc.SetIndent("", "  ")
			if err := enc.Encode(val); err != nil {
				return "", err
			}
			return buf.String(), nil
		},
	}

	t, err := template.New("output").Funcs(funcs).Option("missingkey=zero").Parse(tmpl)
	if err != nil {
		return formatTemplateError("invalid template", err)
	}
	if err := t.Execute(w, v); err != nil {
		return formatTemplateError("template execution error", err)
	}
	return nil
}

var templateLocationPattern = regexp.MustCompile(`:(\d+):(\d+):`)

func formatTemplateError(kind string, err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	if matches := templateLocationPattern.FindStringSubmatch(msg); len(matches) == 3 {
		return fmt.Errorf("%s at line %s, column %s: %s", kind, matches[1], matches[2], msg)
	}
	return fmt.Errorf("%s: %w", kind, err)
}
