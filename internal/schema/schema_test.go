package schema

import (
	"testing"
)

func TestRegisterAndGet(t *testing.T) {
	// Use a unique test schema name to avoid conflicts with registered resources
	s := Object("Test object", map[string]*Schema{
		"id":   Int("Identifier"),
		"name": String("Name"),
	}, "id")

	Register("_test_object", s)
	defer func() {
		// Clean up test schema
		ClearRegistry()
		// Re-register resource schemas
		registerAllResources()
	}()

	got, err := Get("_test_object")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if got.Type != "object" {
		t.Errorf("expected type 'object', got %q", got.Type)
	}
	if got.Description != "Test object" {
		t.Errorf("expected description 'Test object', got %q", got.Description)
	}
	if len(got.Required) != 1 || got.Required[0] != "id" {
		t.Errorf("expected required ['id'], got %v", got.Required)
	}
	if len(got.Properties) != 2 {
		t.Errorf("expected 2 properties, got %d", len(got.Properties))
	}
}

func TestGetNotFound(t *testing.T) {
	_, err := Get("_definitely_nonexistent_schema")
	if err == nil {
		t.Error("expected error for nonexistent schema")
	}
}

func TestListIsSorted(t *testing.T) {
	names := List()

	if len(names) == 0 {
		t.Fatal("expected at least some registered schemas")
	}

	// Verify names are sorted
	for i := 1; i < len(names); i++ {
		if names[i] < names[i-1] {
			t.Errorf("names not sorted: %v", names)
			break
		}
	}
}

// registerAllResources re-registers all resource schemas after ClearRegistry
func registerAllResources() {
	registerConversation()
	registerContact()
	registerMessage()
	registerInbox()
	registerAgent()
	registerTeam()
	registerLabel()
}

func TestBuilders(t *testing.T) {
	t.Run("String", func(t *testing.T) {
		s := String("A string field")
		if s.Type != "string" {
			t.Errorf("expected type 'string', got %q", s.Type)
		}
		if s.Description != "A string field" {
			t.Errorf("expected description 'A string field', got %q", s.Description)
		}
	})

	t.Run("Int", func(t *testing.T) {
		s := Int("An integer field")
		if s.Type != "integer" {
			t.Errorf("expected type 'integer', got %q", s.Type)
		}
	})

	t.Run("Bool", func(t *testing.T) {
		s := Bool("A boolean field")
		if s.Type != "boolean" {
			t.Errorf("expected type 'boolean', got %q", s.Type)
		}
	})

	t.Run("Enum", func(t *testing.T) {
		s := Enum("Status", "open", "closed", "pending")
		if s.Type != "string" {
			t.Errorf("expected type 'string', got %q", s.Type)
		}
		if len(s.Enum) != 3 {
			t.Errorf("expected 3 enum values, got %d", len(s.Enum))
		}
		if s.Enum[0] != "open" || s.Enum[1] != "closed" || s.Enum[2] != "pending" {
			t.Errorf("unexpected enum values: %v", s.Enum)
		}
	})

	t.Run("Array", func(t *testing.T) {
		s := Array(String("item"), "A list of strings")
		if s.Type != "array" {
			t.Errorf("expected type 'array', got %q", s.Type)
		}
		if s.Items == nil {
			t.Error("expected Items to be set")
		}
		if s.Items.Type != "string" {
			t.Errorf("expected Items.Type 'string', got %q", s.Items.Type)
		}
	})

	t.Run("Object", func(t *testing.T) {
		s := Object("An object", map[string]*Schema{
			"foo": String("Foo field"),
			"bar": Int("Bar field"),
		}, "foo")
		if s.Type != "object" {
			t.Errorf("expected type 'object', got %q", s.Type)
		}
		if len(s.Properties) != 2 {
			t.Errorf("expected 2 properties, got %d", len(s.Properties))
		}
		if len(s.Required) != 1 || s.Required[0] != "foo" {
			t.Errorf("expected required ['foo'], got %v", s.Required)
		}
	})

	t.Run("Timestamp", func(t *testing.T) {
		s := Timestamp("Created at")
		if s.Type != "integer" {
			t.Errorf("expected type 'integer', got %q", s.Type)
		}
		if s.Description != "Created at (Unix timestamp)" {
			t.Errorf("expected description with Unix timestamp suffix, got %q", s.Description)
		}
	})

	t.Run("Map", func(t *testing.T) {
		s := Map("Custom attributes")
		if s.Type != "object" {
			t.Errorf("expected type 'object', got %q", s.Type)
		}
	})
}

func TestResourceSchemasRegistered(t *testing.T) {
	// Verify all expected resource schemas are registered
	expectedSchemas := []string{
		"conversation",
		"contact",
		"message",
		"inbox",
		"agent",
		"team",
		"label",
	}

	for _, name := range expectedSchemas {
		s, err := Get(name)
		if err != nil {
			t.Errorf("schema %q not registered: %v", name, err)
			continue
		}
		if s.Type != "object" {
			t.Errorf("schema %q should be object type, got %q", name, s.Type)
		}
		if s.Description == "" {
			t.Errorf("schema %q should have a description", name)
		}
		if len(s.Properties) == 0 {
			t.Errorf("schema %q should have properties", name)
		}
	}
}

func TestConversationSchema(t *testing.T) {
	s, err := Get("conversation")
	if err != nil {
		t.Fatalf("Get conversation failed: %v", err)
	}

	// Check required fields
	requiredFields := map[string]bool{
		"id":         false,
		"inbox_id":   false,
		"status":     false,
		"created_at": false,
	}
	for _, req := range s.Required {
		if _, ok := requiredFields[req]; ok {
			requiredFields[req] = true
		}
	}
	for field, found := range requiredFields {
		if !found {
			t.Errorf("expected %q to be required", field)
		}
	}

	// Check status enum
	status := s.Properties["status"]
	if status == nil {
		t.Fatal("expected status property")
	}
	if len(status.Enum) != 4 {
		t.Errorf("expected 4 status enum values, got %d", len(status.Enum))
	}
}

func TestContactSchema(t *testing.T) {
	s, err := Get("contact")
	if err != nil {
		t.Fatalf("Get contact failed: %v", err)
	}

	// Verify expected properties exist
	expectedProps := []string{"id", "name", "email", "phone_number", "identifier", "created_at"}
	for _, prop := range expectedProps {
		if _, ok := s.Properties[prop]; !ok {
			t.Errorf("expected property %q in contact schema", prop)
		}
	}
}

func TestMessageSchema(t *testing.T) {
	s, err := Get("message")
	if err != nil {
		t.Fatalf("Get message failed: %v", err)
	}

	// Check message_type enum
	msgType := s.Properties["message_type"]
	if msgType == nil {
		t.Fatal("expected message_type property")
	}
	if len(msgType.Enum) != 4 {
		t.Errorf("expected 4 message_type enum values, got %d", len(msgType.Enum))
	}
}

func TestAgentSchema(t *testing.T) {
	s, err := Get("agent")
	if err != nil {
		t.Fatalf("Get agent failed: %v", err)
	}

	// Check role enum
	role := s.Properties["role"]
	if role == nil {
		t.Fatal("expected role property")
	}
	if len(role.Enum) != 2 {
		t.Errorf("expected 2 role enum values, got %d", len(role.Enum))
	}

	// Check availability enum
	availability := s.Properties["availability_status"]
	if availability == nil {
		t.Fatal("expected availability_status property")
	}
	if len(availability.Enum) != 3 {
		t.Errorf("expected 3 availability enum values, got %d", len(availability.Enum))
	}
}
