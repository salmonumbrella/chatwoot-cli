# 💬 Chatwoot CLI — Chatwoot in your terminal.

Chatwoot in your terminal. Manage conversations, contacts, campaigns, help center, and integrations.

## Features

- **Automations** - list and view automation rules, manage canned response templates
- **Bots** - create, update, delete bots
- **Campaigns** - create and manage SMS and messaging campaigns
- **Contacts** - create, update, search, filter, merge duplicates, bulk operations, manage labels and notes
- **Conversations** - list, filter, search, assign, status, priority, labels
- **Help Center** - manage portals, articles, and categories
- **Inboxes** - list and view inbox details, member access and roles, create and manage saved filter presets
- **Messages** - send, edit, delete messages and list attachments
- **Platform APIs** - manage accounts and users (self-hosted/managed)
- **Public Client APIs** - manage contacts and conversations via inbox identifiers
- **Profiles** - store multiple accounts/tokens and switch contexts quickly
- **Reports** - audit logs, customer satisfaction surveys, reports with metrics
- **Teams** - list teams and team members
- **Webhooks** - manage webhooks

## Installation

### Homebrew

```bash
brew install salmonumbrella/tap/chatwoot-cli
```

### Build from Source

```bash
git clone https://github.com/salmonumbrella/chatwoot-cli.git
cd chatwoot-cli
make build
# Binary at ./bin/chatwoot
```

## Quick Start

### 1. Authenticate

**Browser:**
```bash
chatwoot auth login
```

**Terminal:**
```bash
chatwoot auth login --browser=false --url https://chatwoot.example.com --token YOUR_API_TOKEN --account-id 1
```

### 2. Verify Setup

```bash
chatwoot profile get
```

### 3. List Conversations

```bash
chatwoot conversations list --status open
```

### 4. Search (Agent-Friendly)

```bash
# Search across contacts + conversations
chatwoot search "john"

# Auto-pick the best result (no interactive prompt)
chatwoot search "refund" --best

# Emit just an ID for chaining (no jq)
chatwoot search "refund" --best --emit id
```

## Configuration

### Authentication

Credentials are checked in this order:
1. Environment variables (`CHATWOOT_BASE_URL`, `CHATWOOT_API_TOKEN`, `CHATWOOT_ACCOUNT_ID`)
2. `CHATWOOT_PROFILE` (if set)
3. Current profile in keychain (defaults to `default`)

Check current configuration:
```bash
chatwoot auth status
chatwoot auth status --json
```

Remove stored credentials:
```bash
chatwoot auth logout
```

### Profiles

Manage multiple stored accounts:
```bash
chatwoot config profiles list
chatwoot config profiles use staging
chatwoot config profiles show --name staging
chatwoot config profiles delete staging
```

### Environment Variables

```bash
export CHATWOOT_BASE_URL=https://chatwoot.example.com
export CHATWOOT_API_TOKEN=your_api_token
export CHATWOOT_ACCOUNT_ID=1
export CHATWOOT_PROFILE=staging
export CHATWOOT_PLATFORM_TOKEN=your_platform_token
export CHATWOOT_ALLOW_PRIVATE=1
export CHATWOOT_OUTPUT=agent
export CHATWOOT_RESOLVE_NAMES=1
```

## Security

### Credential Storage

Credentials saved with `chatwoot auth login` are stored securely in your system's keychain:
- **macOS**: Keychain Access
- **Linux**: Secret Service (GNOME Keyring, KWallet)
- **Windows**: Credential Manager

## Commands

### Authentication

```bash
chatwoot auth login                                                 # Authenticate via browser
chatwoot auth login --no-browser --url <url> --token <t> --account-id <id>  # CLI login
chatwoot auth status                                                # Show current config
chatwoot auth logout                                                # Remove credentials
```

## Extensions

Executables on your PATH named `chatwoot-<name>` can be invoked as:

```bash
chatwoot <name> [args...]
```

### Shortcuts (Agent-Friendly)

Convenience commands designed for agent workflows:

```bash
# "get/show" are aliases for `open`
chatwoot get conversation 123
chatwoot show https://app.chatwoot.com/app/accounts/1/conversations/123

# One-ID actions (no extra lookups)
chatwoot comment 123 "Hello! How can I help?"
chatwoot note 123 "Internal note" --mention lily
chatwoot close 123 456
chatwoot reopen 123
chatwoot ctx 123 --output agent
```

### Conversations

```bash
# List and filter
chatwoot conversations list
chatwoot conversations list --status open --inbox-id 1
chatwoot conversations list --status open --assignee-type unassigned
chatwoot conversations list --status open --team-id 2 --labels "vip,urgent"
chatwoot conversations list --status open --search "refund"
chatwoot conversations list --all --max-pages 50
chatwoot conversations filter --payload '[{"attribute_key":"status","filter_operator":"equal_to","values":["open"]}]'
chatwoot conversations search --query "refund"

# Get details
chatwoot conversations get 123
chatwoot conversations counts                     # Get counts by status
chatwoot conversations meta                       # Get metadata
chatwoot conversations attachments 123            # List attachments

# Manage status and assignment
chatwoot conversations toggle-status 123 --status resolved
chatwoot conversations toggle-priority 123 --priority high
chatwoot conversations assign 123 --agent 5 --team 2
chatwoot conversations mark-unread 123

# Labels
chatwoot conversations labels 123
chatwoot conversations labels-add 123 --labels "urgent,vip"

# Custom attributes
chatwoot conversations custom-attributes 123 --payload '{"key":"value"}'

# AI context
chatwoot conversations context 123 --embed-images   # Get full context for AI
chatwoot conversations context 123 --output agent   # Agent-friendly envelope
chatwoot conversations context 123 --output agent --resolve-names

# Transcript
chatwoot conversations transcript 123               # Render transcript locally
chatwoot conversations transcript 123 --public-only # Exclude private notes
chatwoot conversations transcript 123 --limit 200   # Limit to most recent messages
chatwoot conversations transcript 123 --email user@example.com
```

### Messages

```bash
# List messages
chatwoot messages list 123
chatwoot messages list 123 --all
chatwoot messages list 123 --limit 500
chatwoot messages list 123 --output agent --resolve-names

# Create messages
chatwoot messages create 123 --content "Hello!"

# Update and delete
chatwoot messages update 123 456 --content "Updated text"
chatwoot messages delete 123 456
```

> **Note:** Messages are returned in chronological order (oldest first, most recent at end of array).
> To get the last N messages: `chatwoot messages list 123 --json | jq '.items[-N:]'`

### Private Notes & Mentions

Private notes are internal messages visible only to agents, not customers. You can mention/tag agents to notify them.

```bash
# Create a private note (internal, not visible to customer)
chatwoot messages create 123 --private --content "Internal note for the team"

# Mention an agent (they'll receive a notification)
chatwoot messages create 123 --private --mention lily --content "Can you follow up on this?"

# Mention multiple agents
chatwoot messages create 123 --private --mention lily --mention jack --content "Please review together"

# Mention by email
chatwoot messages create 123 --private --mention lily@example.com --content "Check this out"
```

The `--mention` flag:
- Accepts agent name (partial match) or email
- Automatically resolves to the agent's ID
- Formats the mention correctly so the agent receives a notification
- Requires the `--private` flag (mentions only work in private notes)

### Contacts

```bash
# List and search
chatwoot contacts list
chatwoot contacts list --sort name --order asc
chatwoot contacts list --sort -last_activity_at
chatwoot contacts search --query "john"
chatwoot contacts filter --payload '[{"attribute_key":"email","filter_operator":"contains","values":["@example.com"]}]'

# CRUD operations
chatwoot contacts get 123
chatwoot contacts show 123                        # Alias for 'get'
chatwoot contacts get +16042091231                # Lookup by phone number
chatwoot contacts create --name "John Doe" --email "john@example.com"
chatwoot contacts update 123 --phone "+1234567890"
chatwoot contacts update john@example.com --name "John Smith"
chatwoot contacts update +16042091231 --name "Wenqi Qu" --email "quwenqi@example.com"
chatwoot contacts delete 123

# Merge contacts (combine duplicates)
chatwoot contacts merge 123 456                   # Merge 456 INTO 123 (456 deleted)
chatwoot contacts merge 123 456 --force           # Skip confirmation

# Related data
chatwoot contacts conversations 123
chatwoot contacts contactable-inboxes 123

# Inbox association
chatwoot contacts create-inbox 123 --inbox-id 1 --source-id "+15551234567"

# Labels
chatwoot contacts labels 123
chatwoot contacts labels-add 123 --labels "customer,premium"

# Bulk label operations
chatwoot contacts bulk add-label --ids 1,2,3 --labels "vip,priority"
chatwoot contacts bulk remove-label --ids 1,2,3 --labels "old-tag"

# Bulk IDs can also come from stdin/files: @- or @path
printf "1\n2\n3\n" | chatwoot contacts bulk add-label --ids @- --labels vip

# Notes
chatwoot contacts notes 123
chatwoot contacts notes-add 123 --content "Called about refund"
chatwoot contacts notes-delete 123 456
```

### Campaigns

```bash
# List and get
chatwoot campaigns list
chatwoot campaigns list --all
chatwoot campaigns get 123

# Create campaign
chatwoot campaigns create --title "Welcome" --message "Hello!" --inbox-id 1 --labels 5,6

# Update and delete
chatwoot campaigns update 123 --enabled=false
chatwoot campaigns delete 123 --force
```

### Help Center (Portals)

```bash
# Portals
chatwoot portals list
chatwoot portals get help
chatwoot portals create --name "Help Center" --slug help
chatwoot portals update help --name "Support Center"
chatwoot portals delete help

# Articles
chatwoot portals articles list help
chatwoot portals articles create help --title "Getting Started" --content "..." --category-id 1
chatwoot portals articles update help 123 --status 1  # 0=draft, 1=published, 2=archived
chatwoot portals articles delete help 123

# Categories
chatwoot portals categories list help
chatwoot portals categories create help --name "FAQ" --slug faq
chatwoot portals categories update help faq --name "Frequently Asked Questions"
chatwoot portals categories delete help faq
```

### Inboxes

```bash
chatwoot inboxes list
chatwoot inboxes get 1
chatwoot inboxes create --name "Support" --channel-type api --greeting-enabled --greeting-message "Hi!"
chatwoot inboxes update 1 --timezone "America/New_York" --working-hours-enabled
```

### Inbox Members

```bash
chatwoot inbox-members list --inbox-id 1
chatwoot inbox-members add --inbox-id 1 --user-id 5
chatwoot inbox-members update --inbox-id 1 --user-id 5 --role administrator
chatwoot inbox-members remove --inbox-id 1 --user-id 5
```

### Agents & Teams

```bash
# Agents
chatwoot agents list
chatwoot agents get 5

# Teams
chatwoot teams list
chatwoot teams get 1
chatwoot teams members 1
```

### Canned Responses

```bash
chatwoot canned-responses list
chatwoot canned-responses get 123
chatwoot canned-responses create --short-code "greeting" --content "Hello! How can I help?"
chatwoot canned-responses update 123 --content "Updated response"
chatwoot canned-responses delete 123
```

### Webhooks

```bash
chatwoot webhooks list
chatwoot webhooks get 123
chatwoot webhooks create --url "https://example.com/webhook" --subscriptions "message_created,conversation_created"
chatwoot webhooks update 123 --url "https://new.example.com/hook"
chatwoot webhooks delete 123
```

### Automation Rules

```bash
chatwoot automation-rules list
chatwoot automation-rules get 123
```

### Agent Bots

```bash
chatwoot agent-bots list
chatwoot agent-bots get 123
chatwoot agent-bots create --name "Support Bot" --description "Handles FAQs"
chatwoot agent-bots update 123 --name "FAQ Bot"
chatwoot agent-bots delete 123
```

### Custom Attributes

```bash
chatwoot custom-attributes list --attribute-model contact_attribute
chatwoot custom-attributes get 123 --attribute-model contact_attribute
chatwoot custom-attributes create --attribute-model contact_attribute --attribute-key "vip_status" --attribute-display-name "VIP Status" --attribute-display-type list --attribute-values '["gold","silver","bronze"]'
chatwoot custom-attributes update 123 --attribute-model contact_attribute --attribute-display-name "VIP Level"
chatwoot custom-attributes delete 123 --attribute-model contact_attribute
```

### Custom Filters

```bash
chatwoot custom-filters list --filter-type conversation
chatwoot custom-filters get 123 --filter-type conversation
chatwoot custom-filters create --filter-type conversation --name "High Priority Open" --query '{"payload":[{"attribute_key":"status","filter_operator":"equal_to","values":["open"]},{"attribute_key":"priority","filter_operator":"equal_to","values":["high"]}]}'
chatwoot custom-filters update 123 --filter-type conversation --name "Updated Filter"
chatwoot custom-filters delete 123 --filter-type conversation
```

### Labels

```bash
chatwoot labels list
chatwoot labels get 123
chatwoot labels create --title "VIP"
chatwoot labels update 123 --title "Premium Customer"
chatwoot labels delete 123
```

### Integrations

```bash
chatwoot integrations list
chatwoot integrations hooks list --app-id slack
chatwoot integrations hooks create --app-id slack --inbox-id 1 --settings '{"webhook_url":"https://hooks.slack.com/..."}'
chatwoot integrations hooks update 123 --app-id slack --settings '{"webhook_url":"https://new.hooks.slack.com/..."}'
chatwoot integrations hooks delete 123 --app-id slack
```

### Platform APIs

```bash
# Accounts
chatwoot platform accounts create --name "Acme Inc" --domain acme.example.com --support-email support@acme.example.com
chatwoot platform accounts get 123
chatwoot platform accounts delete 123

# Users
chatwoot platform users create --name "Jane Doe" --email jane@example.com --password "secret"
chatwoot platform users update 45 --display-name "Jane D"
chatwoot platform users delete 45

# Account users
chatwoot platform account-users list 123
chatwoot platform account-users create 123 --user-id 45 --role agent
chatwoot platform account-users delete 123 --user-id 45
```

### Public Client APIs

```bash
# Contacts
chatwoot client contacts create --inbox <inbox-identifier> --name "Visitor" --email visitor@example.com
chatwoot client contacts get --inbox <inbox-identifier> --contact <contact-identifier>

# Conversations
chatwoot client conversations list --inbox <inbox-identifier> --contact <contact-identifier>
chatwoot client conversations create --inbox <inbox-identifier> --contact <contact-identifier>
chatwoot client conversations get 123 --inbox <inbox-identifier> --contact <contact-identifier>
chatwoot client conversations resolve 123 --inbox <inbox-identifier> --contact <contact-identifier>

# Messages & typing
chatwoot client messages create 123 --inbox <inbox-identifier> --contact <contact-identifier> --content "Hello!"
chatwoot client typing 123 --inbox <inbox-identifier> --contact <contact-identifier> --status on
chatwoot client last-seen update 123 --inbox <inbox-identifier> --contact <contact-identifier>
```

### Reports

```bash
chatwoot reports summary --since 2024-01-01 --until 2024-12-31
chatwoot reports summary --metric conversations_count --type account
```

### CSAT (Customer Satisfaction)

```bash
chatwoot csat list
chatwoot csat metrics --since 2024-01-01 --until 2024-12-31
```

### Audit Logs

```bash
chatwoot audit-logs list
chatwoot audit-logs list --user-id 5
```

### Profile & Account

```bash
chatwoot profile get                              # Get your user profile
chatwoot account get                              # Get account details
```

## Output Formats

### Text

Human-readable tables:

```bash
$ chatwoot conversations list
ID      STATUS    INBOX         CONTACT           MESSAGES    UPDATED
123     open      Support       John Doe          5           2 hours ago
124     pending   Sales         Jane Smith        3           1 day ago
```

### JSON

Machine-readable output:

```bash
$ chatwoot conversations list --output json
{
  "items": [
    {
      "id": 123,
      "status": "open",
      "inbox_id": 1,
      "contact": {"name": "John Doe"},
      "messages_count": 5
    }
  ]
}
```

Tip: `--json` is a shorthand for `--output json`.

### Agent JSON

Agent-optimized JSON output with consistent `kind` envelopes, compact summaries, and readable timestamps.
Enable it per command with `--output agent`, or set a default for your shell:

```bash
export CHATWOOT_OUTPUT=agent
```

Use `--resolve-names` to fetch inbox/contact names for conversation results (extra API calls).
Agent output is available for `conversations context` with message metadata and summary.

**Context commands return a single envelope**:
```bash
chatwoot conversations context 123 --output agent | jq '.item.summary'
chatwoot conversations context 123 --output agent --resolve-names | jq '.item.contact_labels'
```

**List commands return an object with an "items" array** (plus `has_more` and `meta`):
```bash
chatwoot contacts list --output agent | jq '.items[0]'
chatwoot conversations list --output agent | jq '.items[] | select(.status == "open")'
```

**Get commands return single objects**:
```bash
chatwoot contacts get 123 --output agent | jq '.item.email'
chatwoot conversations get 456 --output agent | jq '.item.messages_count'
```

Data goes to stdout, errors and progress to stderr for clean piping.

### JSONL

Streaming-friendly output (one JSON object per line):

```bash
$ chatwoot conversations list --output jsonl
{"id":123,"status":"open",...}
{"id":124,"status":"pending",...}
```

Tip: `--query` and `--template` apply per line in JSONL mode.

## Examples

### Triage open conversations

```bash
# List all open conversations
chatwoot conversations list --status open --output json | \
  jq '.items[] | {id, contact: .contact.name, messages: .messages_count}'

# Assign high-priority to agent
chatwoot conversations assign 123 --agent 5
chatwoot conversations toggle-priority 123 --priority high
```

### Send bulk campaign

```bash
# Create campaign for specific labels
chatwoot campaigns create \
  --title "Product Update" \
  --message "Check out our new feature!" \
  --inbox-id 1 \
  --labels 10,11,12
```

### Build AI support bot

```bash
# Get conversation context with images for AI vision
chatwoot conversations context 123 --embed-images --output json > context.json

# Process with AI (example using OpenAI or similar)
cat context.json | your-ai-tool --prompt "Draft a helpful response"

# Send the response
chatwoot messages create 123 --content "Your drafted response here"
```

### Search contacts by email domain

```bash
chatwoot contacts filter \
  --payload '[{"attribute_key":"email","filter_operator":"contains","values":["@example.com"]}]' \
  --output json | jq '.items[].email'
```

### Export conversations for analysis

```bash
# Get all resolved conversations from last month
chatwoot conversations filter \
  --payload '[{"attribute_key":"status","filter_operator":"equal_to","values":["resolved"]}]' \
  --output json > resolved-conversations.json

# Extract key metrics
jq '[.items[] | {id, resolved_at: .updated_at, messages: .messages_count, agent: .assignee.name}]' \
  resolved-conversations.json
```

### Manage help center content

```bash
# Create portal
chatwoot portals create --name "Knowledge Base" --slug kb

# Create category
chatwoot portals categories create kb --name "Getting Started" --slug getting-started

# Publish article
chatwoot portals articles create kb \
  --title "How to Get Started" \
  --content "..." \
  --category-id 1 \
  --status 1
```

### AI Context

Get complete conversation data optimized for use by LLMs:

```bash
# Text format with embedded images
chatwoot conversations context 123 --embed-images

# JSON format for programmatic access
chatwoot conversations context 123 --embed-images --output json
```

The `--embed-images` flag converts images to base64 data URIs that AI vision models can process directly.

### Pagination

List commands support pagination:

```bash
# Get all results (automatic pagination)
chatwoot conversations list --all

# Limit pagination depth
chatwoot conversations list --all --max-pages 50
```

### Filtering and Search

**Filter** uses Chatwoot's filter API for structured queries:
```bash
chatwoot conversations filter --payload '[{"attribute_key":"status","filter_operator":"equal_to","values":["open"]}]'
chatwoot contacts filter --payload '[{"attribute_key":"email","filter_operator":"contains","values":["@vip.com"]}]'
```

**Search** performs full-text search:
```bash
chatwoot conversations search --query "refund request"
chatwoot contacts search --query "john smith"
```

## Common Workflows

### Inbox → Conversation → Message

```bash
# Pick an inbox
chatwoot inboxes list

# List open conversations in an inbox
chatwoot conversations list --status open --inbox-id 1

# Send a message in a conversation
chatwoot messages create 123 --content "Hello! How can I help?"
```

### Contacts → Conversations

```bash
# Find a contact
chatwoot contacts search --query "john@example.com"

# Get the conversations for a contact
chatwoot contacts conversations 123
chatwoot contacts conversations 123 --output agent --resolve-names
```

## Troubleshooting

- **401 Unauthorized**: run `chatwoot auth login` and verify your token.
- **403 Forbidden**: check your account role and permissions.
- **404 Not Found**: verify the resource ID (it may have been deleted).
- **URL validation failed**: ensure the base URL is public, or use `--allow-private` only if you trust the target.
- **Base URL not configured**: set `CHATWOOT_BASE_URL` / `CHATWOOT_API_TOKEN` / `CHATWOOT_ACCOUNT_ID` or run `chatwoot auth login`.

## Global Flags

All commands support these flags:

- `--output <format>` - Output format: `text`, `json`, or `jsonl` (default: text)
- `--json` - Alias for `--output json`
- `--color <mode>` - Color mode: `auto`, `always`, or `never` (default: auto)
- `--allow-private` - Allow private/localhost URLs (unsafe)
- `--debug` - Enable verbose debug logging
- `--dry-run` - Preview changes without executing mutations
- `--timeout <duration>` - HTTP request timeout (default: 30s)
- `--idempotency-key <key|auto>` - Idempotency key for write requests (use `auto` for per-request keys)
- `--query <expr>` - JQ expression to filter JSON output
- `--jq <expr>` - Alias for `--query`
- `--fields <a,b,c>` - Select fields in JSON output (shorthand for `--query`; supports presets like `minimal`, `default`, `debug` on supported resources)
- `--quiet` - Suppress non-essential output
- `--silent` - Suppress non-error output to stderr
- `--no-input` - Disable interactive prompts
- `--yes`, `-y` - Assume yes for confirmations (desire path alias for `--force`)
- `--template <tmpl>` - Go template (or `@path`) to render JSON output
- `--utc` - Display timestamps in UTC
- `--time-zone <tz>` - Display timestamps in a specific time zone (e.g., `America/Los_Angeles`)
- `--max-rate-limit-retries <n>` - Max retries for HTTP 429 responses
- `--max-5xx-retries <n>` - Max retries for HTTP 5xx responses
- `--rate-limit-delay <duration>` - Base delay for 429 retries (e.g., 1s)
- `--server-error-delay <duration>` - Delay between 5xx retries (e.g., 1s)
- `--circuit-breaker-threshold <n>` - Failures before circuit opens
- `--circuit-breaker-reset-time <duration>` - Circuit breaker reset time (e.g., 30s)
- `--help` - Show help for any command

Note: `--utc` and `--time-zone` are mutually exclusive.

You can force interactive prompts in non-TTY environments by setting `CHATWOOT_FORCE_INTERACTIVE=true`.

### JQ Filtering

Filter JSON output with JQ expressions:

```bash
# Get only conversation IDs
chatwoot conversations list -o json --query '.items[].id'

# Filter by status
chatwoot conversations list -o json --query '.items[] | select(.status == "open")'
```

**Fields shorthand & templates**

```bash
# Select fields without writing JQ
chatwoot conversations list -o json --fields id,status,assignee_id

# Render custom output with a template
chatwoot conversations get 123 -o json --template '{{.id}} {{.status}}'
```

### Field Presets

Many commands support field presets for common use cases. Instead of listing individual fields, use a preset name:

```bash
# Minimal output - just essential identifiers
chatwoot contacts list -o json --fields minimal

# Default output - commonly needed fields
chatwoot conversations list -o json --fields default

# Debug output - all fields for troubleshooting
chatwoot contacts list -o json --fields debug
```

**Available presets:**

| Preset    | Purpose                                      |
|-----------|----------------------------------------------|
| `minimal` | Bare essentials for quick lookups            |
| `default` | Common fields for typical workflows          |
| `debug`   | All fields including metadata and timestamps |

The exact fields in each preset vary by resource. For example:

- **contacts minimal**: `id`, `name`, `email`
- **contacts debug**: includes `custom_attributes`, `thumbnail`, `last_activity_at`
- **conversations minimal**: `id`, `status`, `inbox_id`, `assignee_id`
- **conversations debug**: includes `labels`, `meta`, `custom_attributes`, `unread_count`

Field presets work with JSON output (`-o json` or `--json`). Commands without registered presets treat preset names as literal field names.

## Shell Completions

Completion commands are enabled. Generate shell completions for your preferred shell:

### Bash

```bash
chatwoot completion bash > /usr/local/etc/bash_completion.d/chatwoot
# Or for Linux:
chatwoot completion bash > /etc/bash_completion.d/chatwoot
```

### Zsh

```zsh
chatwoot completion zsh > "${fpath[1]}/_chatwoot"
# Or add to .zshrc:
echo 'eval "$(chatwoot completion zsh)"' >> ~/.zshrc
```

### Fish

```fish
chatwoot completion fish > ~/.config/fish/completions/chatwoot.fish
```

### PowerShell

```powershell
chatwoot completion powershell | Out-String | Invoke-Expression
# Or add to profile:
chatwoot completion powershell >> $PROFILE
```

## Version & Updates

```bash
chatwoot version
```

## Development

After cloning, install git hooks:

```bash
make setup
```

This installs [lefthook](https://github.com/evilmartians/lefthook) pre-commit and pre-push hooks for linting and testing.

### Build from Source

```bash
make build
```

### Run tests

```bash
make test
```

Golden output fixtures for CLI JSON snapshots live in `internal/cmd/testdata/golden`. To refresh them:

```bash
UPDATE_GOLDEN=1 go test ./internal/cmd -run TestGolden
```

## License

MIT

## Links

- [Chatwoot GitHub Repository](https://github.com/chatwoot)
- [Chatwoot API Documentation](https://www.chatwoot.com/developers/api)
