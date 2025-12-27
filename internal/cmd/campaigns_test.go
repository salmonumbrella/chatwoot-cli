package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/chatwoot/chatwoot-cli/internal/api"
)

func TestLabelsParsingCreate(t *testing.T) {
	tests := []struct {
		name          string
		labelsFlag    string
		expectedCount int
		expectedIDs   []int
		expectError   bool
	}{
		{
			name:          "single label",
			labelsFlag:    "1",
			expectedCount: 1,
			expectedIDs:   []int{1},
			expectError:   false,
		},
		{
			name:          "multiple labels",
			labelsFlag:    "1,2,3",
			expectedCount: 3,
			expectedIDs:   []int{1, 2, 3},
			expectError:   false,
		},
		{
			name:          "labels with spaces",
			labelsFlag:    "1, 2, 3",
			expectedCount: 3,
			expectedIDs:   []int{1, 2, 3},
			expectError:   false,
		},
		{
			name:          "many labels",
			labelsFlag:    "10,20,30,40,50",
			expectedCount: 5,
			expectedIDs:   []int{10, 20, 30, 40, 50},
			expectError:   false,
		},
		{
			name:          "empty string",
			labelsFlag:    "",
			expectedCount: 0,
			expectedIDs:   []int{},
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var audience []api.CampaignAudience

			if tt.labelsFlag != "" {
				for _, idStr := range strings.Split(tt.labelsFlag, ",") {
					idStr = strings.TrimSpace(idStr)
					var id int
					if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
						if tt.expectError {
							return // Expected error occurred
						}
						t.Fatalf("Failed to parse label ID %q: %v", idStr, err)
					}
					audience = append(audience, api.CampaignAudience{Type: "Label", ID: id})
				}
			}

			if tt.expectError {
				t.Error("Expected error but got none")
				return
			}

			if len(audience) != tt.expectedCount {
				t.Errorf("Expected %d audience items, got %d", tt.expectedCount, len(audience))
			}

			for i, expectedID := range tt.expectedIDs {
				if i >= len(audience) {
					t.Errorf("Missing audience item at index %d", i)
					continue
				}
				if audience[i].Type != "Label" {
					t.Errorf("Expected audience[%d].Type to be 'Label', got %s", i, audience[i].Type)
				}
				if audience[i].ID != expectedID {
					t.Errorf("Expected audience[%d].ID to be %d, got %d", i, expectedID, audience[i].ID)
				}
			}
		})
	}
}

func TestLabelsParsingUpdate(t *testing.T) {
	tests := []struct {
		name          string
		labelsFlag    string
		expectedCount int
		expectedIDs   []int
		expectError   bool
	}{
		{
			name:          "update with single label",
			labelsFlag:    "5",
			expectedCount: 1,
			expectedIDs:   []int{5},
			expectError:   false,
		},
		{
			name:          "update with multiple labels",
			labelsFlag:    "5,6,7",
			expectedCount: 3,
			expectedIDs:   []int{5, 6, 7},
			expectError:   false,
		},
		{
			name:          "update with whitespace",
			labelsFlag:    " 1 , 2 , 3 ",
			expectedCount: 3,
			expectedIDs:   []int{1, 2, 3},
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var audience []api.CampaignAudience

			if tt.labelsFlag != "" {
				for _, idStr := range strings.Split(tt.labelsFlag, ",") {
					idStr = strings.TrimSpace(idStr)
					var id int
					if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
						if tt.expectError {
							return
						}
						t.Fatalf("Failed to parse label ID %q: %v", idStr, err)
					}
					audience = append(audience, api.CampaignAudience{Type: "Label", ID: id})
				}
			}

			if len(audience) != tt.expectedCount {
				t.Errorf("Expected %d audience items, got %d", tt.expectedCount, len(audience))
			}

			for i, expectedID := range tt.expectedIDs {
				if audience[i].ID != expectedID {
					t.Errorf("Expected audience[%d].ID to be %d, got %d", i, expectedID, audience[i].ID)
				}
			}
		})
	}
}

func TestLabelsMutualExclusivityCreate(t *testing.T) {
	tests := []struct {
		name         string
		labelsFlag   string
		audienceFlag string
		expectError  bool
		errorMessage string
	}{
		{
			name:         "both flags set",
			labelsFlag:   "1,2,3",
			audienceFlag: `[{"type":"Label","id":1}]`,
			expectError:  true,
			errorMessage: "--labels and --audience are mutually exclusive",
		},
		{
			name:         "only labels set",
			labelsFlag:   "1,2,3",
			audienceFlag: "",
			expectError:  false,
		},
		{
			name:         "only audience set",
			labelsFlag:   "",
			audienceFlag: `[{"type":"Label","id":1}]`,
			expectError:  false,
		},
		{
			name:         "neither flag set",
			labelsFlag:   "",
			audienceFlag: "",
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the mutual exclusivity check from campaigns.go
			if tt.labelsFlag != "" && tt.audienceFlag != "" {
				if !tt.expectError {
					t.Error("Expected error for mutual exclusivity but expectError is false")
				}
				// Verify error message matches
				expectedErr := "--labels and --audience are mutually exclusive"
				if tt.errorMessage != expectedErr {
					t.Errorf("Expected error message %q, got %q", expectedErr, tt.errorMessage)
				}
				return
			}

			if tt.expectError {
				t.Error("Expected error but got none")
			}
		})
	}
}

func TestLabelsMutualExclusivityUpdate(t *testing.T) {
	tests := []struct {
		name         string
		labelsFlag   string
		audienceFlag string
		expectError  bool
	}{
		{
			name:         "update with both flags",
			labelsFlag:   "5,6",
			audienceFlag: `[{"type":"Label","id":5}]`,
			expectError:  true,
		},
		{
			name:         "update with only labels",
			labelsFlag:   "5,6",
			audienceFlag: "",
			expectError:  false,
		},
		{
			name:         "update with only audience",
			labelsFlag:   "",
			audienceFlag: `[{"type":"Label","id":5}]`,
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.labelsFlag != "" && tt.audienceFlag != "" {
				if !tt.expectError {
					t.Error("Expected error for mutual exclusivity")
				}
				return
			}

			if tt.expectError {
				t.Error("Expected error but got none")
			}
		})
	}
}

func TestAudienceJSONParsing(t *testing.T) {
	tests := []struct {
		name          string
		audienceJSON  string
		expectedCount int
		expectError   bool
	}{
		{
			name:          "valid single audience",
			audienceJSON:  `[{"type":"Label","id":1}]`,
			expectedCount: 1,
			expectError:   false,
		},
		{
			name:          "valid multiple audience",
			audienceJSON:  `[{"type":"Label","id":1},{"type":"Label","id":2}]`,
			expectedCount: 2,
			expectError:   false,
		},
		{
			name:          "empty array",
			audienceJSON:  `[]`,
			expectedCount: 0,
			expectError:   false,
		},
		{
			name:         "invalid JSON",
			audienceJSON: `{invalid}`,
			expectError:  true,
		},
		{
			name:         "not an array",
			audienceJSON: `{"type":"Label","id":1}`,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var audience []api.CampaignAudience
			err := json.Unmarshal([]byte(tt.audienceJSON), &audience)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if !tt.expectError && len(audience) != tt.expectedCount {
				t.Errorf("Expected %d audience items, got %d", tt.expectedCount, len(audience))
			}
		})
	}
}

func TestCampaignAudienceStructure(t *testing.T) {
	tests := []struct {
		name         string
		audience     api.CampaignAudience
		expectedType string
		expectedID   int
	}{
		{
			name:         "label audience",
			audience:     api.CampaignAudience{Type: "Label", ID: 1},
			expectedType: "Label",
			expectedID:   1,
		},
		{
			name:         "label with higher ID",
			audience:     api.CampaignAudience{Type: "Label", ID: 999},
			expectedType: "Label",
			expectedID:   999,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.audience.Type != tt.expectedType {
				t.Errorf("Expected type %s, got %s", tt.expectedType, tt.audience.Type)
			}
			if tt.audience.ID != tt.expectedID {
				t.Errorf("Expected ID %d, got %d", tt.expectedID, tt.audience.ID)
			}

			// Test JSON marshaling
			data, err := json.Marshal(tt.audience)
			if err != nil {
				t.Fatalf("Failed to marshal audience: %v", err)
			}

			var unmarshaled api.CampaignAudience
			if err := json.Unmarshal(data, &unmarshaled); err != nil {
				t.Fatalf("Failed to unmarshal audience: %v", err)
			}

			if unmarshaled.Type != tt.expectedType {
				t.Errorf("After unmarshal, expected type %s, got %s", tt.expectedType, unmarshaled.Type)
			}
			if unmarshaled.ID != tt.expectedID {
				t.Errorf("After unmarshal, expected ID %d, got %d", tt.expectedID, unmarshaled.ID)
			}
		})
	}
}

func TestLabelsToAudienceConversion(t *testing.T) {
	tests := []struct {
		name        string
		labelIDs    []int
		expectedLen int
	}{
		{
			name:        "convert three labels",
			labelIDs:    []int{1, 2, 3},
			expectedLen: 3,
		},
		{
			name:        "convert single label",
			labelIDs:    []int{42},
			expectedLen: 1,
		},
		{
			name:        "convert many labels",
			labelIDs:    []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			expectedLen: 10,
		},
		{
			name:        "empty labels",
			labelIDs:    []int{},
			expectedLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var audience []api.CampaignAudience
			for _, id := range tt.labelIDs {
				audience = append(audience, api.CampaignAudience{Type: "Label", ID: id})
			}

			if len(audience) != tt.expectedLen {
				t.Errorf("Expected %d audience items, got %d", tt.expectedLen, len(audience))
			}

			for i, id := range tt.labelIDs {
				if audience[i].Type != "Label" {
					t.Errorf("Expected audience[%d].Type to be 'Label', got %s", i, audience[i].Type)
				}
				if audience[i].ID != id {
					t.Errorf("Expected audience[%d].ID to be %d, got %d", i, id, audience[i].ID)
				}
			}
		})
	}
}

func TestLabelsParsingErrors(t *testing.T) {
	tests := []struct {
		name       string
		labelsFlag string
		shouldFail bool // whether parsing should fail
	}{
		{
			name:       "non-numeric labels",
			labelsFlag: "abc,def",
			shouldFail: true,
		},
		{
			name:       "float labels",
			labelsFlag: "1.5,2.3",
			shouldFail: false, // fmt.Sscanf("%d") stops at decimal point, successfully parses "1" and "2"
		},
		{
			name:       "empty values",
			labelsFlag: "1,,3",
			shouldFail: true,
		},
		{
			name:       "mixed valid and invalid",
			labelsFlag: "1,abc,3",
			shouldFail: true,
		},
		{
			name:       "negative numbers",
			labelsFlag: "-1,-2",
			shouldFail: false, // negative numbers parse successfully with fmt.Sscanf
		},
		{
			name:       "whitespace only",
			labelsFlag: "  ,  ",
			shouldFail: true,
		},
		{
			name:       "special characters",
			labelsFlag: "1,@#$,3",
			shouldFail: true,
		},
		{
			name:       "mixed numbers and text",
			labelsFlag: "123abc",
			shouldFail: false, // fmt.Sscanf("%d") stops at first non-digit, successfully parses "123"
		},
		{
			name:       "text before numbers",
			labelsFlag: "abc123",
			shouldFail: true, // fmt.Sscanf("%d") fails if text comes first
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var hadError bool

			// Simulate the label parsing logic from campaigns.go
			for _, idStr := range strings.Split(tt.labelsFlag, ",") {
				idStr = strings.TrimSpace(idStr)

				// Check for empty string after trimming
				if idStr == "" {
					hadError = true
					continue
				}

				var id int
				if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
					hadError = true
					continue
				}
				// Successfully parsed id (not storing audience for this validation test)
				_ = id
			}

			if tt.shouldFail && !hadError {
				t.Errorf("Expected parsing to fail for input %q but it succeeded", tt.labelsFlag)
			}

			if !tt.shouldFail && hadError {
				t.Errorf("Expected parsing to succeed for input %q but it failed", tt.labelsFlag)
			}
		})
	}
}

func TestCampaignTitleEdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		title string
	}{
		{"empty title", ""},
		{"title with quotes", `Campaign "Special"`},
		{"title with format specifiers", "Get %d%% off today!"},
		{"title with newline", "Line1\nLine2"},
		{"title with tabs", "Tab\there"},
		{"very long title", strings.Repeat("a", 200)},
		{"unicode title", "Campaign 日本語 🎉"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Using %q should safely handle all these cases
			result := fmt.Sprintf("Delete campaign %q (ID: %d)? (y/N): ", tt.title, 123)

			// Verify the result is a valid string (no panic)
			if result == "" {
				t.Error("Expected non-empty result")
			}

			// Verify the ID is present
			if !strings.Contains(result, "123") {
				t.Errorf("Expected ID 123 in result: %s", result)
			}
		})
	}
}
