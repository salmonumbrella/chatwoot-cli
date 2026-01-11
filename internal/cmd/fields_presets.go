package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

const fieldsPresetsAnnotation = "chatwoot.fields.presets"

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
	if raw == "" {
		return nil, nil
	}
	var presets map[string][]string
	if err := json.Unmarshal([]byte(raw), &presets); err != nil {
		return nil, fmt.Errorf("invalid fields presets: %w", err)
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
			return preset, nil
		}
	}

	return parseFields(input)
}
