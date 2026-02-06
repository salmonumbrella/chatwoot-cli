package api

import (
	"context"
	"fmt"
	"net/http"
)

// List returns all automation rules for the account.
func (s AutomationRulesService) List(ctx context.Context) ([]AutomationRule, error) {
	return listAutomationRules(ctx, s)
}

func listAutomationRules(ctx context.Context, r Requester) ([]AutomationRule, error) {
	var result AutomationRuleListResponse
	if err := r.do(ctx, http.MethodGet, r.accountPath("/automation_rules"), nil, &result); err != nil {
		return nil, err
	}
	return result.Payload, nil
}

// Get returns a specific automation rule by ID.
func (s AutomationRulesService) Get(ctx context.Context, id int) (*AutomationRule, error) {
	return getAutomationRule(ctx, s, id)
}

func getAutomationRule(ctx context.Context, r Requester, id int) (*AutomationRule, error) {
	var result AutomationRuleResponse
	path := fmt.Sprintf("/automation_rules/%d", id)
	if err := r.do(ctx, http.MethodGet, r.accountPath(path), nil, &result); err != nil {
		return nil, err
	}
	return &result.Payload, nil
}

// Create creates a new automation rule.
func (s AutomationRulesService) Create(ctx context.Context, name, eventName string, conditions, actions []map[string]any) (*AutomationRule, error) {
	return createAutomationRule(ctx, s, name, eventName, conditions, actions)
}

func createAutomationRule(ctx context.Context, r Requester, name, eventName string, conditions, actions []map[string]any) (*AutomationRule, error) {
	body := map[string]any{
		"name":       name,
		"event_name": eventName,
		"conditions": conditions,
		"actions":    actions,
	}
	var rule AutomationRule
	if err := r.do(ctx, http.MethodPost, r.accountPath("/automation_rules"), body, &rule); err != nil {
		return nil, err
	}
	return &rule, nil
}

// Update updates an existing automation rule.
func (s AutomationRulesService) Update(ctx context.Context, id int, name string, conditions, actions []map[string]any, active *bool) (*AutomationRule, error) {
	return updateAutomationRule(ctx, s, id, name, conditions, actions, active)
}

func updateAutomationRule(ctx context.Context, r Requester, id int, name string, conditions, actions []map[string]any, active *bool) (*AutomationRule, error) {
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
	if active != nil {
		body["active"] = *active
	}

	var result AutomationRuleResponse
	path := fmt.Sprintf("/automation_rules/%d", id)
	if err := r.do(ctx, http.MethodPatch, r.accountPath(path), body, &result); err != nil {
		return nil, err
	}
	return &result.Payload, nil
}

// Delete deletes an automation rule.
func (s AutomationRulesService) Delete(ctx context.Context, id int) error {
	return deleteAutomationRule(ctx, s, id)
}

func deleteAutomationRule(ctx context.Context, r Requester, id int) error {
	path := fmt.Sprintf("/automation_rules/%d", id)
	return r.do(ctx, http.MethodDelete, r.accountPath(path), nil, nil)
}

// Clone clones an existing automation rule.
func (s AutomationRulesService) Clone(ctx context.Context, id int) (*AutomationRule, error) {
	return cloneAutomationRule(ctx, s, id)
}

func cloneAutomationRule(ctx context.Context, r Requester, id int) (*AutomationRule, error) {
	path := fmt.Sprintf("/automation_rules/%d/clone", id)
	var result AutomationRuleResponse
	if err := r.do(ctx, http.MethodPost, r.accountPath(path), nil, &result); err != nil {
		return nil, err
	}
	return &result.Payload, nil
}
