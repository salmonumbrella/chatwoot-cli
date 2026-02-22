// Package schema provides schema introspection for Chatwoot API resources.
// It enables agents to programmatically discover field structures for API resources.
package schema

import (
	"fmt"
	"sort"
	"sync"
)

// Schema represents a JSON Schema-like type definition for API resources.
type Schema struct {
	Type        string             `json:"type"`
	Description string             `json:"description,omitempty"`
	Required    []string           `json:"required,omitempty"`
	Properties  map[string]*Schema `json:"properties,omitempty"`
	Items       *Schema            `json:"items,omitempty"`
	Enum        []string           `json:"enum,omitempty"`
}

var (
	registry = make(map[string]*Schema)
	mu       sync.RWMutex
)

// Register adds a schema to the global registry.
func Register(name string, s *Schema) {
	mu.Lock()
	defer mu.Unlock()
	registry[name] = s
}

// Get retrieves a schema by name from the registry.
func Get(name string) (*Schema, error) {
	mu.RLock()
	defer mu.RUnlock()
	s, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("schema %q not found", name)
	}
	return s, nil
}

// List returns all registered schema names, sorted alphabetically.
func List() []string {
	mu.RLock()
	defer mu.RUnlock()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Object creates an object schema with properties.
func Object(desc string, props map[string]*Schema, required ...string) *Schema {
	return &Schema{
		Type:        "object",
		Description: desc,
		Properties:  props,
		Required:    required,
	}
}

// String creates a string schema.
func String(desc string) *Schema {
	return &Schema{
		Type:        "string",
		Description: desc,
	}
}

// Int creates an integer schema.
func Int(desc string) *Schema {
	return &Schema{
		Type:        "integer",
		Description: desc,
	}
}

// Bool creates a boolean schema.
func Bool(desc string) *Schema {
	return &Schema{
		Type:        "boolean",
		Description: desc,
	}
}

// Enum creates a string schema with enumerated values.
func Enum(desc string, values ...string) *Schema {
	return &Schema{
		Type:        "string",
		Description: desc,
		Enum:        values,
	}
}

// Array creates an array schema with items of a given type.
func Array(items *Schema, desc string) *Schema {
	return &Schema{
		Type:        "array",
		Description: desc,
		Items:       items,
	}
}

// Timestamp creates a schema for Unix timestamp fields.
func Timestamp(desc string) *Schema {
	return &Schema{
		Type:        "integer",
		Description: desc + " (Unix timestamp)",
	}
}

// Map creates a schema for key-value map fields.
func Map(desc string) *Schema {
	return &Schema{
		Type:        "object",
		Description: desc,
	}
}

// ClearRegistry removes all registered schemas (useful for testing).
func ClearRegistry() {
	mu.Lock()
	defer mu.Unlock()
	registry = make(map[string]*Schema)
}
