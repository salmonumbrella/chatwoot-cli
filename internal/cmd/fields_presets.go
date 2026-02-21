package cmd

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/chatwoot/chatwoot-cli/internal/queryalias"
	"github.com/chatwoot/chatwoot-cli/internal/schema"
	"github.com/spf13/cobra"
)

const (
	fieldsPresetsAnnotation = "chatwoot.fields.presets"
	fieldsSchemaAnnotation  = "chatwoot.fields.schema"
)

func registerFieldPresets(cmd *cobra.Command, presets map[string][]string) {
	if cmd == nil || len(presets) == 0 {
		return
	}

	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}

	normalized := normalizeFieldPresets(presets)
	data, err := json.Marshal(normalized)
	if err != nil {
		return
	}
	cmd.Annotations[fieldsPresetsAnnotation] = string(data)
}

func registerFieldSchema(cmd *cobra.Command, schemaName string) {
	if cmd == nil {
		return
	}
	schemaName = strings.TrimSpace(schemaName)
	if schemaName == "" {
		return
	}
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	cmd.Annotations[fieldsSchemaAnnotation] = schemaName
}

func normalizeFieldPresets(presets map[string][]string) map[string][]string {
	out := make(map[string][]string, len(presets))
	for key, fields := range presets {
		name := strings.ToLower(strings.TrimSpace(key))
		if name == "" {
			continue
		}
		var cleaned []string
		for _, field := range fields {
			field = strings.TrimSpace(field)
			if field == "" {
				continue
			}
			cleaned = append(cleaned, field)
		}
		if len(cleaned) == 0 {
			continue
		}
		out[name] = cleaned
	}
	return out
}

func fieldPresetsForCommand(cmd *cobra.Command) (map[string][]string, error) {
	if cmd == nil || cmd.Annotations == nil {
		return nil, nil
	}
	raw := cmd.Annotations[fieldsPresetsAnnotation]
	var presets map[string][]string
	if raw != "" {
		if err := json.Unmarshal([]byte(raw), &presets); err != nil {
			return nil, fmt.Errorf("invalid fields presets: %w", err)
		}
	}

	schemaName := fieldSchemaForCommand(cmd)
	if schemaName == "" {
		return presets, nil
	}

	derived := schemaPresets(schemaName)
	if len(derived) > 0 {
		merged := make(map[string][]string, len(derived))
		for k, v := range derived {
			merged[k] = v
		}
		for k, v := range presets {
			merged[k] = v
		}
		presets = merged
	}

	if len(presets) == 0 {
		return presets, nil
	}

	if err := validatePresetsAgainstSchema(schemaName, presets); err != nil {
		return nil, err
	}

	return presets, nil
}

func parseFieldsWithPresets(cmd *cobra.Command, input string) ([]string, error) {
	presets, err := fieldPresetsForCommand(cmd)
	if err != nil {
		return nil, err
	}

	trimmed := strings.TrimSpace(input)
	if trimmed != "" && !strings.Contains(trimmed, ",") {
		key := strings.ToLower(trimmed)
		if preset, ok := presets[key]; ok {
			return normalizeFieldPaths(preset), nil
		}
	}

	fields, err := parseFields(input)
	if err != nil {
		return nil, err
	}
	fields = normalizeFieldPaths(fields)

	schemaName := fieldSchemaForCommand(cmd)
	if schemaName == "" {
		return fields, nil
	}
	if err := validateFieldsAgainstSchema(schemaName, fields); err != nil {
		return nil, err
	}
	return fields, nil
}

func normalizeFieldPaths(fields []string) []string {
	if len(fields) == 0 {
		return fields
	}
	out := make([]string, len(fields))
	for i, field := range fields {
		out[i] = queryalias.Normalize(field, queryalias.ContextPath)
	}
	return out
}

func fieldSchemaForCommand(cmd *cobra.Command) string {
	if cmd == nil || cmd.Annotations == nil {
		return ""
	}
	return strings.TrimSpace(cmd.Annotations[fieldsSchemaAnnotation])
}

func schemaPresets(schemaName string) map[string][]string {
	s, err := schema.Get(schemaName)
	if err != nil || s == nil || len(s.Properties) == 0 {
		return nil
	}

	propNames := make([]string, 0, len(s.Properties))
	for name := range s.Properties {
		propNames = append(propNames, name)
	}
	sort.Strings(propNames)

	required := append([]string{}, s.Required...)
	requiredSet := make(map[string]struct{}, len(required))
	for _, name := range required {
		requiredSet[name] = struct{}{}
	}

	var optional []string
	for _, name := range propNames {
		if _, ok := requiredSet[name]; ok {
			continue
		}
		optional = append(optional, name)
	}

	minimal := append([]string{}, required...)
	if len(minimal) == 0 {
		minimal = takeFields(optional, 3)
		if len(minimal) == 0 {
			minimal = append([]string{}, propNames...)
		}
	}

	defaultFields := append([]string{}, required...)
	if len(optional) > 0 {
		defaultFields = append(defaultFields, takeFields(optional, 5)...)
	}
	if len(defaultFields) == 0 {
		defaultFields = append([]string{}, propNames...)
	}

	debug := append([]string{}, propNames...)

	return normalizeFieldPresets(map[string][]string{
		"minimal": minimal,
		"default": defaultFields,
		"debug":   debug,
	})
}

func takeFields(fields []string, n int) []string {
	if n <= 0 || len(fields) == 0 {
		return nil
	}
	if len(fields) <= n {
		return append([]string{}, fields...)
	}
	return append([]string{}, fields[:n]...)
}

func validatePresetsAgainstSchema(schemaName string, presets map[string][]string) error {
	for name, fields := range presets {
		if err := validateFieldsAgainstSchema(schemaName, fields); err != nil {
			return fmt.Errorf("invalid fields preset %q: %w", name, err)
		}
	}
	return nil
}

func validateFieldsAgainstSchema(schemaName string, fields []string) error {
	s, err := schema.Get(schemaName)
	if err != nil || s == nil || len(s.Properties) == 0 {
		return nil
	}

	propNames := make([]string, 0, len(s.Properties))
	props := make(map[string]struct{}, len(s.Properties))
	for name := range s.Properties {
		propNames = append(propNames, name)
		props[name] = struct{}{}
	}
	sort.Strings(propNames)

	var invalid []string
	for _, field := range fields {
		root := strings.SplitN(field, ".", 2)[0]
		if _, ok := props[root]; !ok {
			invalid = append(invalid, field)
		}
	}
	if len(invalid) == 0 {
		return nil
	}

	return fmt.Errorf("unknown field(s) for %s: %s (available: %s)", schemaName, strings.Join(invalid, ", "), strings.Join(propNames, ", "))
}
