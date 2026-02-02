# Agent Mode Phase 2 Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add 7 agent-efficiency features to reduce API round-trips and provide richer context for AI assistants.

**Architecture:** Each feature follows existing patterns: add flags to commands, extend agentfmt types where needed, add API methods as required. All features are independent and can be implemented in parallel.

**Tech Stack:** Go, Cobra CLI, existing api/agentfmt packages

---

## Task 1: Add `--with-messages` flag to `conversations get`

Include recent messages inline when fetching a conversation to save a round-trip.

**Files:**
- Modify: `internal/cmd/conversations.go` (newConversationsGetCmd function)
- Modify: `internal/agentfmt/agentfmt.go` (add ConversationDetailWithMessages type)
- Test: `internal/cmd/conversations_cmd_test.go`
- Test: `internal/agentfmt/agentfmt_test.go`

**Step 1: Add ConversationDetailWithMessages type to agentfmt**

Add to `internal/agentfmt/agentfmt.go` after the `ConversationDetail` struct:

```go
// ConversationDetailWithMessages includes recent messages inline.
type ConversationDetailWithMessages struct {
	ConversationDetail
	Messages []MessageSummary `json:"messages,omitempty"`
}
```

**Step 2: Write the failing test**

Add to `internal/cmd/conversations_cmd_test.go`:

```go
func TestConversationsGetWithMessages(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/123", jsonResponse(200, `{
			"id": 123,
			"inbox_id": 1,
			"status": "open",
			"messages_count": 2,
			"unread_count": 1,
			"created_at": 1700000000,
			"last_activity_at": 1700001000
		}`)).
		On("GET", "/api/v1/accounts/1/conversations/123/messages", jsonResponse(200, `{
			"payload": [
				{"id": 1, "content": "Hello", "message_type": 0, "created_at": 1700000000},
				{"id": 2, "content": "Hi there", "message_type": 1, "created_at": 1700000500}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "get", "123", "--with-messages", "--output", "agent"})
		require.NoError(t, err)
	})

	var result map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &result))

	item := result["item"].(map[string]any)
	messages := item["messages"].([]any)
	assert.Len(t, messages, 2)
}
```

**Step 3: Run test to verify it fails**

Run: `go test -run TestConversationsGetWithMessages ./internal/cmd/ -v`
Expected: FAIL with "unknown flag: --with-messages"

**Step 4: Implement the flag in conversations.go**

Find `newConversationsGetCmd` and add the flag variable and logic:

```go
func newConversationsGetCmd() *cobra.Command {
	var withMessages bool
	var messageLimit int

	cmd := &cobra.Command{
		// ... existing Use, Short, Long, Args ...
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			// ... existing id parsing and client creation ...

			conv, err := client.Conversations().Get(ctx, id)
			if err != nil {
				return fmt.Errorf("failed to get conversation %d: %w", id, err)
			}

			if isAgent(cmd) {
				detail := agentfmt.ConversationDetailFromConversation(*conv)

				if withMessages {
					messages, err := client.Messages().List(ctx, id)
					if err != nil {
						return fmt.Errorf("failed to fetch messages: %w", err)
					}
					// Limit messages if specified
					if messageLimit > 0 && len(messages) > messageLimit {
						messages = messages[len(messages)-messageLimit:]
					}
					return printJSON(cmd, agentfmt.ItemEnvelope{
						Kind: agentfmt.KindFromCommandPath(cmd.CommandPath()),
						Item: agentfmt.ConversationDetailWithMessages{
							ConversationDetail: detail,
							Messages:           agentfmt.MessageSummaries(messages),
						},
					})
				}

				return printJSON(cmd, agentfmt.ItemEnvelope{
					Kind: agentfmt.KindFromCommandPath(cmd.CommandPath()),
					Item: detail,
				})
			}

			// ... existing JSON and text output ...
		}),
	}

	cmd.Flags().BoolVar(&withMessages, "with-messages", false, "Include recent messages in output (agent mode)")
	cmd.Flags().IntVar(&messageLimit, "message-limit", 20, "Maximum messages to include with --with-messages")

	// ... existing flags ...
	return cmd
}
```

**Step 5: Run test to verify it passes**

Run: `go test -run TestConversationsGetWithMessages ./internal/cmd/ -v`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/cmd/conversations.go internal/agentfmt/agentfmt.go internal/cmd/conversations_cmd_test.go
git commit -m "feat(conversations): add --with-messages flag for inline message fetching"
```

---

## Task 2: Add `--waiting` flag to `conversations list`

Sort conversations by customer wait time (time since last customer message without agent reply).

**Files:**
- Modify: `internal/cmd/conversations.go` (newConversationsListCmd function)
- Test: `internal/cmd/conversations_cmd_test.go`

**Step 1: Write the failing test**

Add to `internal/cmd/conversations_cmd_test.go`:

```go
func TestConversationsListWaiting(t *testing.T) {
	now := time.Now().Unix()
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations", jsonResponse(200, fmt.Sprintf(`{
			"data": {
				"meta": {"total_pages": 1},
				"payload": [
					{"id": 1, "inbox_id": 1, "status": "open", "last_activity_at": %d, "agent_last_seen_at": %d},
					{"id": 2, "inbox_id": 1, "status": "open", "last_activity_at": %d, "agent_last_seen_at": %d},
					{"id": 3, "inbox_id": 1, "status": "open", "last_activity_at": %d, "agent_last_seen_at": %d}
				]
			}
		}`, now-3600, now-3600, now-7200, now-1800, now-1800, now-1800)))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "list", "--waiting", "--output", "json"})
		require.NoError(t, err)
	})

	var result struct {
		Items []struct {
			ID int `json:"id"`
		} `json:"items"`
	}
	require.NoError(t, json.Unmarshal([]byte(output), &result))

	// Should be sorted by wait time descending (longest waiting first)
	// Conv 2: last_activity 2h ago, agent seen 30m ago = 1.5h wait
	// Conv 1: last_activity 1h ago, agent seen 1h ago = 0h wait (agent replied)
	// Conv 3: last_activity 30m ago, agent seen 30m ago = 0h wait
	assert.Equal(t, 2, result.Items[0].ID, "longest waiting should be first")
}
```

**Step 2: Run test to verify it fails**

Run: `go test -run TestConversationsListWaiting ./internal/cmd/ -v`
Expected: FAIL with "unknown flag: --waiting"

**Step 3: Implement the flag**

In `newConversationsListCmd`, add after the existing flags:

```go
var waiting bool

// In the Fetch function, after fetching items:
if waiting {
	// Sort by wait time (last_activity_at - agent_last_seen_at) descending
	sort.Slice(items, func(i, j int) bool {
		waitI := items[i].LastActivityAt - items[i].AgentLastSeenAt
		waitJ := items[j].LastActivityAt - items[j].AgentLastSeenAt
		// If agent hasn't seen, they've been waiting since last activity
		if items[i].AgentLastSeenAt == 0 {
			waitI = items[i].LastActivityAt
		}
		if items[j].AgentLastSeenAt == 0 {
			waitJ = items[j].LastActivityAt
		}
		return waitI > waitJ // Longest wait first
	})
}

// Add flag:
cmd.Flags().BoolVar(&waiting, "waiting", false, "Sort by customer wait time (longest first)")
```

**Step 4: Run test to verify it passes**

Run: `go test -run TestConversationsListWaiting ./internal/cmd/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/cmd/conversations.go internal/cmd/conversations_cmd_test.go
git commit -m "feat(conversations): add --waiting flag to sort by customer wait time"
```

---

## Task 3: Add `--context` flag to `conversations get`

One-shot "give me everything" - conversation details, recent messages, contact info, relationship summary.

**Files:**
- Modify: `internal/cmd/conversations.go` (newConversationsGetCmd function)
- Modify: `internal/agentfmt/agentfmt.go` (add ConversationContext type)
- Test: `internal/cmd/conversations_cmd_test.go`
- Test: `internal/agentfmt/agentfmt_test.go`

**Step 1: Add ConversationContext type to agentfmt**

Add to `internal/agentfmt/agentfmt.go`:

```go
// ConversationContext provides comprehensive context for a conversation.
type ConversationContext struct {
	Conversation ConversationDetail              `json:"conversation"`
	Messages     []MessageSummary                `json:"messages,omitempty"`
	Contact      *ContactDetailWithRelationship  `json:"contact,omitempty"`
}
```

**Step 2: Write the failing test**

Add to `internal/cmd/conversations_cmd_test.go`:

```go
func TestConversationsGetContext(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/123", jsonResponse(200, `{
			"id": 123,
			"inbox_id": 1,
			"status": "open",
			"contact_id": 456,
			"messages_count": 2,
			"created_at": 1700000000,
			"last_activity_at": 1700001000
		}`)).
		On("GET", "/api/v1/accounts/1/conversations/123/messages", jsonResponse(200, `{
			"payload": [
				{"id": 1, "content": "Hello", "message_type": 0, "created_at": 1700000000}
			]
		}`)).
		On("GET", "/api/v1/accounts/1/contacts/456", jsonResponse(200, `{
			"id": 456,
			"name": "John Doe",
			"email": "john@example.com"
		}`)).
		On("GET", "/api/v1/accounts/1/contacts/456/conversations", jsonResponse(200, `{
			"payload": [
				{"id": 123, "status": "open", "created_at": 1700000000},
				{"id": 100, "status": "resolved", "created_at": 1690000000}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "get", "123", "--context", "--output", "agent"})
		require.NoError(t, err)
	})

	var result map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &result))

	item := result["item"].(map[string]any)
	assert.NotNil(t, item["conversation"])
	assert.NotNil(t, item["messages"])
	assert.NotNil(t, item["contact"])

	contact := item["contact"].(map[string]any)
	assert.NotNil(t, contact["relationship"])
}
```

**Step 3: Run test to verify it fails**

Run: `go test -run TestConversationsGetContext ./internal/cmd/ -v`
Expected: FAIL with "unknown flag: --context"

**Step 4: Implement the flag**

In `newConversationsGetCmd`, add:

```go
var withContext bool

// In RunE, after getting conversation:
if withContext && isAgent(cmd) {
	detail := agentfmt.ConversationDetailFromConversation(*conv)

	// Fetch messages
	messages, _ := client.Messages().List(ctx, id)
	if messageLimit > 0 && len(messages) > messageLimit {
		messages = messages[len(messages)-messageLimit:]
	}

	// Fetch contact with relationship
	var contactWithRel *agentfmt.ContactDetailWithRelationship
	if conv.ContactID > 0 {
		contact, err := client.Contacts().Get(ctx, conv.ContactID)
		if err == nil && contact != nil {
			contactDetail := agentfmt.ContactDetailFromContact(*contact)
			convs, _ := client.Contacts().Conversations(ctx, conv.ContactID)
			relationship := agentfmt.ComputeRelationshipSummary(convs)
			contactWithRel = &agentfmt.ContactDetailWithRelationship{
				ContactDetail: contactDetail,
				Relationship:  relationship,
			}
		}
	}

	return printJSON(cmd, agentfmt.ItemEnvelope{
		Kind: agentfmt.KindFromCommandPath(cmd.CommandPath()),
		Item: agentfmt.ConversationContext{
			Conversation: detail,
			Messages:     agentfmt.MessageSummaries(messages),
			Contact:      contactWithRel,
		},
	})
}

// Add flag:
cmd.Flags().BoolVar(&withContext, "context", false, "Include messages, contact, and relationship (agent mode)")
```

**Step 5: Run test to verify it passes**

Run: `go test -run TestConversationsGetContext ./internal/cmd/ -v`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/cmd/conversations.go internal/agentfmt/agentfmt.go internal/cmd/conversations_cmd_test.go
git commit -m "feat(conversations): add --context flag for comprehensive context"
```

---

## Task 4: Add `--since-last-agent` flag to `messages list`

Only show messages since the last agent reply (what's new that needs response).

**Files:**
- Modify: `internal/cmd/messages.go` (messages list command)
- Test: `internal/cmd/messages_test.go`

**Step 1: Write the failing test**

Add to `internal/cmd/messages_test.go`:

```go
func TestMessagesListSinceLastAgent(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/conversations/123/messages", jsonResponse(200, `{
			"payload": [
				{"id": 1, "content": "Customer question", "message_type": 0, "created_at": 1700000000},
				{"id": 2, "content": "Agent reply", "message_type": 1, "created_at": 1700001000},
				{"id": 3, "content": "Customer follow-up", "message_type": 0, "created_at": 1700002000},
				{"id": 4, "content": "Another question", "message_type": 0, "created_at": 1700003000}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"messages", "list", "123", "--since-last-agent", "--output", "json"})
		require.NoError(t, err)
	})

	var result struct {
		Items []struct {
			ID int `json:"id"`
		} `json:"items"`
	}
	require.NoError(t, json.Unmarshal([]byte(output), &result))

	// Should only include messages after the last agent reply (id 2)
	assert.Len(t, result.Items, 2)
	assert.Equal(t, 3, result.Items[0].ID)
	assert.Equal(t, 4, result.Items[1].ID)
}
```

**Step 2: Run test to verify it fails**

Run: `go test -run TestMessagesListSinceLastAgent ./internal/cmd/ -v`
Expected: FAIL with "unknown flag: --since-last-agent"

**Step 3: Implement the flag**

In the messages list command in `internal/cmd/messages.go`:

```go
var sinceLastAgent bool

// After fetching messages:
if sinceLastAgent {
	// Find the last outgoing (agent) message
	lastAgentIdx := -1
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].MessageType == api.MessageTypeOutgoing {
			lastAgentIdx = i
			break
		}
	}

	// Only keep messages after the last agent message
	if lastAgentIdx >= 0 && lastAgentIdx < len(messages)-1 {
		messages = messages[lastAgentIdx+1:]
	} else if lastAgentIdx == len(messages)-1 {
		// Agent message is the last one, no new customer messages
		messages = nil
	}
	// If no agent message found, keep all messages
}

// Add flag:
cmd.Flags().BoolVar(&sinceLastAgent, "since-last-agent", false, "Only show messages since the last agent reply")
```

**Step 4: Run test to verify it passes**

Run: `go test -run TestMessagesListSinceLastAgent ./internal/cmd/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/cmd/messages.go internal/cmd/messages_test.go
git commit -m "feat(messages): add --since-last-agent flag to show new customer messages"
```

---

## Task 5: Add `--with-open-conversations` flag to `contacts get`

Include the contact's open conversations inline.

**Files:**
- Modify: `internal/cmd/contacts.go` (contactGetRunE function)
- Modify: `internal/agentfmt/agentfmt.go` (extend ContactDetailWithRelationship)
- Test: `internal/cmd/contacts_test.go`

**Step 1: Add OpenConversations field to ContactDetailWithRelationship**

Modify `internal/agentfmt/agentfmt.go`:

```go
// ContactDetailWithRelationship extends ContactDetail with relationship data.
type ContactDetailWithRelationship struct {
	ContactDetail
	Relationship      *RelationshipSummary  `json:"relationship,omitempty"`
	OpenConversations []ConversationSummary `json:"open_conversations,omitempty"`
}
```

**Step 2: Write the failing test**

Add to `internal/cmd/contacts_test.go`:

```go
func TestContactsGetWithOpenConversations(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/contacts/123", jsonResponse(200, `{
			"id": 123,
			"name": "John Doe",
			"email": "john@example.com"
		}`)).
		On("GET", "/api/v1/accounts/1/contacts/123/conversations", jsonResponse(200, `{
			"payload": [
				{"id": 1, "status": "open", "inbox_id": 1, "created_at": 1700000000},
				{"id": 2, "status": "resolved", "inbox_id": 1, "created_at": 1690000000},
				{"id": 3, "status": "pending", "inbox_id": 2, "created_at": 1695000000}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"contacts", "get", "123", "--with-open-conversations", "--output", "agent"})
		require.NoError(t, err)
	})

	var result map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &result))

	item := result["item"].(map[string]any)
	openConvs := item["open_conversations"].([]any)

	// Should only include open and pending conversations (not resolved)
	assert.Len(t, openConvs, 2)
}
```

**Step 3: Run test to verify it fails**

Run: `go test -run TestContactsGetWithOpenConversations ./internal/cmd/ -v`
Expected: FAIL with "unknown flag: --with-open-conversations"

**Step 4: Implement the flag**

Modify `contactGetRunE` in `internal/cmd/contacts.go`:

```go
// Add flag variable at command level
var withOpenConversations bool

// In contactGetRunE, modify the agent output section:
if isAgent(cmd) {
	detail := agentfmt.ContactDetailFromContact(*contact)

	// Fetch conversations for relationship summary
	convs, err := client.Contacts().Conversations(ctx, id)
	if err == nil && convs != nil {
		relationship := agentfmt.ComputeRelationshipSummary(convs)

		result := agentfmt.ContactDetailWithRelationship{
			ContactDetail: detail,
			Relationship:  relationship,
		}

		// Include open conversations if requested
		if withOpenConversations {
			var openConvs []api.Conversation
			for _, conv := range convs {
				if conv.Status == "open" || conv.Status == "pending" {
					openConvs = append(openConvs, conv)
				}
			}
			result.OpenConversations = agentfmt.ConversationSummaries(openConvs)
		}

		return printJSON(cmd, agentfmt.ItemEnvelope{
			Kind: agentfmt.KindFromCommandPath(cmd.CommandPath()),
			Item: result,
		})
	}

	return printJSON(cmd, agentfmt.ItemEnvelope{
		Kind: agentfmt.KindFromCommandPath(cmd.CommandPath()),
		Item: detail,
	})
}

// Add flag to both get and show commands:
cmd.Flags().BoolVar(&withOpenConversations, "with-open-conversations", false, "Include open conversations (agent mode)")
```

**Step 5: Run test to verify it passes**

Run: `go test -run TestContactsGetWithOpenConversations ./internal/cmd/ -v`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/cmd/contacts.go internal/agentfmt/agentfmt.go internal/cmd/contacts_test.go
git commit -m "feat(contacts): add --with-open-conversations flag"
```

---

## Task 6: Add bulk operations - `conversations resolve/assign` with multiple IDs

Allow `conversations resolve 123 456 789` and `conversations assign 123,456 --agent 5`.

**Files:**
- Modify: `internal/cmd/conversations.go` (add new bulk-style commands)
- Test: `internal/cmd/conversations_cmd_test.go`

**Step 1: Write the failing test**

Add to `internal/cmd/conversations_cmd_test.go`:

```go
func TestConversationsResolveMultiple(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/conversations/123/toggle_status", jsonResponse(200, `{"id": 123, "status": "resolved"}`)).
		On("POST", "/api/v1/accounts/1/conversations/456/toggle_status", jsonResponse(200, `{"id": 456, "status": "resolved"}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "resolve", "123", "456", "--output", "json"})
		require.NoError(t, err)
	})

	var result map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &result))
	assert.Equal(t, float64(2), result["success_count"])
}

func TestConversationsAssignMultiple(t *testing.T) {
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/conversations/123/assignments", jsonResponse(200, `{"id": 123}`)).
		On("POST", "/api/v1/accounts/1/conversations/456/assignments", jsonResponse(200, `{"id": 456}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"conversations", "assign", "123,456", "--agent", "5", "--output", "json"})
		require.NoError(t, err)
	})

	var result map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &result))
	assert.Equal(t, float64(2), result["success_count"])
}
```

**Step 2: Run test to verify it fails**

Run: `go test -run "TestConversationsResolveMultiple|TestConversationsAssignMultiple" ./internal/cmd/ -v`
Expected: FAIL

**Step 3: Add top-level resolve command**

Add to `internal/cmd/conversations.go`:

```go
func newConversationsResolveCmd() *cobra.Command {
	var concurrency int

	cmd := &cobra.Command{
		Use:   "resolve <id> [id...]",
		Short: "Resolve one or more conversations",
		Long:  "Mark conversations as resolved. Accepts multiple IDs as arguments or comma-separated.",
		Example: `  # Resolve single conversation
  chatwoot conversations resolve 123

  # Resolve multiple conversations
  chatwoot conversations resolve 123 456 789
  chatwoot conversations resolve 123,456,789`,
		Args: cobra.MinimumNArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			ids, err := parseIDArgs(args)
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			ctx := cmdContext(cmd)

			if len(ids) == 1 {
				// Single ID - simple output
				result, err := client.Conversations().ToggleStatus(ctx, ids[0], "resolved", 0)
				if err != nil {
					return fmt.Errorf("failed to resolve conversation %d: %w", ids[0], err)
				}
				if isJSON(cmd) {
					return printJSON(cmd, result)
				}
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Resolved conversation %d\n", ids[0])
				return nil
			}

			// Multiple IDs - bulk operation
			results := runBulkOperation(ctx, ids, int64(concurrency), false, cmd.ErrOrStderr(),
				func(ctx context.Context, id int) (any, error) {
					return client.Conversations().ToggleStatus(ctx, id, "resolved", 0)
				})

			successCount, failCount := countResults(results)

			if isJSON(cmd) {
				return printJSON(cmd, map[string]any{
					"success_count": successCount,
					"fail_count":    failCount,
				})
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Resolved %d conversations (%d failed)\n", successCount, failCount)
			return nil
		}),
	}

	cmd.Flags().IntVar(&concurrency, "concurrency", DefaultConcurrency, "Max concurrent operations")
	return cmd
}

// parseIDArgs parses IDs from command args (supports both space-separated and comma-separated)
func parseIDArgs(args []string) ([]int, error) {
	var ids []int
	for _, arg := range args {
		parts := strings.Split(arg, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			id, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("invalid ID %q: %w", part, err)
			}
			if id <= 0 {
				return nil, fmt.Errorf("ID must be positive: %d", id)
			}
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("at least one ID is required")
	}
	return ids, nil
}
```

**Step 4: Add top-level assign command**

```go
func newConversationsAssignCmd() *cobra.Command {
	var agentID int
	var teamID int
	var concurrency int

	cmd := &cobra.Command{
		Use:   "assign <id> [id...]",
		Short: "Assign one or more conversations",
		Long:  "Assign conversations to an agent or team. Accepts multiple IDs.",
		Example: `  # Assign to agent
  chatwoot conversations assign 123 --agent 5

  # Assign multiple to agent
  chatwoot conversations assign 123 456 --agent 5
  chatwoot conversations assign 123,456,789 --agent 5

  # Assign to team
  chatwoot conversations assign 123 --team 2`,
		Args: cobra.MinimumNArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			if agentID == 0 && teamID == 0 {
				return fmt.Errorf("either --agent or --team is required")
			}

			ids, err := parseIDArgs(args)
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			ctx := cmdContext(cmd)

			if len(ids) == 1 {
				result, err := client.Conversations().Assign(ctx, ids[0], agentID, teamID)
				if err != nil {
					return fmt.Errorf("failed to assign conversation %d: %w", ids[0], err)
				}
				if isJSON(cmd) {
					return printJSON(cmd, result)
				}
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Assigned conversation %d\n", ids[0])
				return nil
			}

			results := runBulkOperation(ctx, ids, int64(concurrency), false, cmd.ErrOrStderr(),
				func(ctx context.Context, id int) (any, error) {
					return client.Conversations().Assign(ctx, id, agentID, teamID)
				})

			successCount, failCount := countResults(results)

			if isJSON(cmd) {
				return printJSON(cmd, map[string]any{
					"success_count": successCount,
					"fail_count":    failCount,
				})
			}

			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Assigned %d conversations (%d failed)\n", successCount, failCount)
			return nil
		}),
	}

	cmd.Flags().IntVar(&agentID, "agent", 0, "Agent ID to assign to")
	cmd.Flags().IntVar(&teamID, "team", 0, "Team ID to assign to")
	cmd.Flags().IntVar(&concurrency, "concurrency", DefaultConcurrency, "Max concurrent operations")
	return cmd
}
```

**Step 5: Register the commands**

In `newConversationsCmd()`, add:

```go
cmd.AddCommand(newConversationsResolveCmd())
cmd.AddCommand(newConversationsAssignCmd())  // Note: this replaces existing assign if there's a conflict
```

**Step 6: Run test to verify it passes**

Run: `go test -run "TestConversationsResolveMultiple|TestConversationsAssignMultiple" ./internal/cmd/ -v`
Expected: PASS

**Step 7: Commit**

```bash
git add internal/cmd/conversations.go internal/cmd/conversations_cmd_test.go
git commit -m "feat(conversations): add resolve and assign commands with multi-ID support"
```

---

## Task 7: Add `inboxes stats` command

Quick view of inbox health: open count, avg wait time, unread count.

**Files:**
- Modify: `internal/cmd/inboxes.go` (add newInboxesStatsCmd)
- Modify: `internal/api/inboxes.go` (add Stats method if needed)
- Test: `internal/cmd/inboxes_test.go`

**Step 1: Write the failing test**

Add to `internal/cmd/inboxes_test.go`:

```go
func TestInboxesStats(t *testing.T) {
	now := time.Now().Unix()
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/inboxes/1", jsonResponse(200, `{
			"id": 1,
			"name": "Support Inbox"
		}`)).
		On("GET", "/api/v1/accounts/1/conversations", jsonResponse(200, fmt.Sprintf(`{
			"data": {
				"meta": {"total_pages": 1},
				"payload": [
					{"id": 1, "inbox_id": 1, "status": "open", "unread_count": 2, "last_activity_at": %d, "agent_last_seen_at": %d},
					{"id": 2, "inbox_id": 1, "status": "open", "unread_count": 0, "last_activity_at": %d, "agent_last_seen_at": %d},
					{"id": 3, "inbox_id": 1, "status": "pending", "unread_count": 1, "last_activity_at": %d, "agent_last_seen_at": 0}
				]
			}
		}`, now-3600, now-1800, now-7200, now-7200, now-1800)))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"inboxes", "stats", "1", "--output", "json"})
		require.NoError(t, err)
	})

	var result map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &result))

	assert.Equal(t, float64(1), result["inbox_id"])
	assert.Equal(t, "Support Inbox", result["inbox_name"])
	assert.Equal(t, float64(2), result["open_count"])
	assert.Equal(t, float64(1), result["pending_count"])
	assert.Equal(t, float64(3), result["unread_count"])
}
```

**Step 2: Run test to verify it fails**

Run: `go test -run TestInboxesStats ./internal/cmd/ -v`
Expected: FAIL with "unknown command \"stats\""

**Step 3: Implement the command**

Add to `internal/cmd/inboxes.go`:

```go
func newInboxesStatsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stats <id>",
		Short: "Get inbox statistics",
		Long:  "Returns inbox health metrics: open count, pending count, unread messages, average wait time",
		Args:  cobra.ExactArgs(1),
		RunE: RunE(func(cmd *cobra.Command, args []string) error {
			id, err := validation.ParsePositiveInt(args[0], "ID")
			if err != nil {
				return err
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			ctx := cmdContext(cmd)

			// Get inbox info
			inbox, err := client.Inboxes().Get(ctx, id)
			if err != nil {
				return fmt.Errorf("failed to get inbox: %w", err)
			}

			// Get conversations for this inbox
			result, err := client.Conversations().List(ctx, api.ListConversationsParams{
				InboxID: fmt.Sprintf("%d", id),
				Status:  "all",
			})
			if err != nil {
				return fmt.Errorf("failed to list conversations: %w", err)
			}

			// Calculate stats
			var openCount, pendingCount, unresolvedCount, totalUnread int
			var totalWaitTime int64
			var waitingCount int
			now := time.Now().Unix()

			for _, conv := range result.Data.Payload {
				switch conv.Status {
				case "open":
					openCount++
					unresolvedCount++
				case "pending":
					pendingCount++
					unresolvedCount++
				}
				totalUnread += conv.Unread

				// Calculate wait time for open/pending conversations
				if conv.Status == "open" || conv.Status == "pending" {
					if conv.AgentLastSeenAt > 0 && conv.LastActivityAt > conv.AgentLastSeenAt {
						totalWaitTime += conv.LastActivityAt - conv.AgentLastSeenAt
						waitingCount++
					} else if conv.AgentLastSeenAt == 0 {
						totalWaitTime += now - conv.LastActivityAt
						waitingCount++
					}
				}
			}

			avgWaitSeconds := int64(0)
			if waitingCount > 0 {
				avgWaitSeconds = totalWaitTime / int64(waitingCount)
			}

			stats := map[string]any{
				"inbox_id":        inbox.ID,
				"inbox_name":      inbox.Name,
				"open_count":      openCount,
				"pending_count":   pendingCount,
				"unresolved_count": unresolvedCount,
				"unread_count":    totalUnread,
				"avg_wait_seconds": avgWaitSeconds,
				"waiting_count":   waitingCount,
			}

			if isJSON(cmd) {
				return printJSON(cmd, stats)
			}

			// Text output
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Inbox: %s (ID: %d)\n", inbox.Name, inbox.ID)
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Open: %d | Pending: %d | Unread: %d\n", openCount, pendingCount, totalUnread)
			if waitingCount > 0 {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Avg wait: %s (%d waiting)\n", formatDuration(avgWaitSeconds), waitingCount)
			}
			return nil
		}),
	}

	return cmd
}

func formatDuration(seconds int64) string {
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	if seconds < 3600 {
		return fmt.Sprintf("%dm", seconds/60)
	}
	hours := seconds / 3600
	mins := (seconds % 3600) / 60
	if mins == 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dh%dm", hours, mins)
}
```

**Step 4: Register the command**

In `newInboxesCmd()`, add:

```go
cmd.AddCommand(newInboxesStatsCmd())
```

**Step 5: Run test to verify it passes**

Run: `go test -run TestInboxesStats ./internal/cmd/ -v`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/cmd/inboxes.go internal/cmd/inboxes_test.go
git commit -m "feat(inboxes): add stats command for inbox health metrics"
```

---

## Summary

| Task | Feature | Files |
|------|---------|-------|
| 1 | `conversations get --with-messages` | conversations.go, agentfmt.go |
| 2 | `conversations list --waiting` | conversations.go |
| 3 | `conversations get --context` | conversations.go, agentfmt.go |
| 4 | `messages list --since-last-agent` | messages.go |
| 5 | `contacts get --with-open-conversations` | contacts.go, agentfmt.go |
| 6 | `conversations resolve/assign` multi-ID | conversations.go |
| 7 | `inboxes stats` | inboxes.go |

All tasks are independent and can be implemented in parallel by separate subagents.
