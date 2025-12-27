package api

import (
	"context"
	"fmt"
)

// ListAutomationRules returns all automation rules for the account
func (c *Client) ListAutomationRules(ctx context.Context) ([]AutomationRule, error) {
	var result AutomationRuleListResponse
	if err := c.Get(ctx, "/automation_rules", &result); err != nil {
		return nil, err
	}
	return result.Payload, nil
}

// GetAutomationRule returns a specific automation rule by ID
func (c *Client) GetAutomationRule(ctx context.Context, id int) (*AutomationRule, error) {
	var result AutomationRuleResponse
	path := fmt.Sprintf("/automation_rules/%d", id)
	if err := c.Get(ctx, path, &result); err != nil {
		return nil, err
	}
	return &result.Payload, nil
}

// CreateAutomationRule creates a new automation rule
func (c *Client) CreateAutomationRule(ctx context.Context, name, eventName string, conditions, actions []map[string]any) (*AutomationRule, error) {
	body := map[string]any{
		"name":       name,
		"event_name": eventName,
		"conditions": conditions,
		"actions":    actions,
	}
	var rule AutomationRule
	if err := c.Post(ctx, "/automation_rules", body, &rule); err != nil {
		return nil, err
	}
	return &rule, nil
}

// UpdateAutomationRule updates an existing automation rule
func (c *Client) UpdateAutomationRule(ctx context.Context, id int, name string, conditions, actions []map[string]any) (*AutomationRule, error) {
	body := map[string]any{}
	if name != "" {
		body["name"] = name
	}
	if conditions != nil {
		body["conditions"] = conditions
	}
	if actions != nil {
		body["actions"] = actions
	}

	var result AutomationRuleResponse
	path := fmt.Sprintf("/automation_rules/%d", id)
	if err := c.Patch(ctx, path, body, &result); err != nil {
		return nil, err
	}
	return &result.Payload, nil
}

// DeleteAutomationRule deletes an automation rule
func (c *Client) DeleteAutomationRule(ctx context.Context, id int) error {
	path := fmt.Sprintf("/automation_rules/%d", id)
	return c.Delete(ctx, path)
}
