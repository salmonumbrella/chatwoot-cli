# Chatwoot CLI

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
chatwoot au login
```

**Terminal:**
```bash
chatwoot au login --browser=false --url https://chatwoot.example.com --token YOUR_API_TOKEN --account-id 1
```

### 2. Verify Setup

```bash
chatwoot pr g
```

### 3. List Conversations

```bash
chatwoot c ls --st open
```

### 4. Search (Agent-Friendly)

```bash
# Search across contacts + conversations
chatwoot s "john"

# Auto-pick the best result (no interactive prompt)
chatwoot s "refund" --best

# Emit just an ID for chaining (no jq)
chatwoot s "refund" --best --emit id
```

## Configuration

### Authentication

Credentials are checked in this order:
1. Environment variables (`CHATWOOT_BASE_URL`, `CHATWOOT_API_TOKEN`, `CHATWOOT_ACCOUNT_ID`)
2. `CHATWOOT_PROFILE` (if set)
3. Current profile in keychain (defaults to `default`)

Check current configuration:
```bash
chatwoot au status
chatwoot au status --json
```

Remove stored credentials:
```bash
chatwoot au logout
```

### Profiles

Manage multiple stored accounts:
```bash
chatwoot cfg profiles ls
chatwoot cfg profiles use staging
chatwoot cfg profiles show --name staging
chatwoot cfg profiles del staging
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

Credentials saved with `chatwoot au login` are stored securely in your system's keychain:
- **macOS**: Keychain Access
- **Linux**: Secret Service (GNOME Keyring, KWallet)
- **Windows**: Credential Manager

## Commands

### Authentication

```bash
chatwoot au login                                                       # Authenticate via browser
chatwoot au login --no-browser --url <url> --token <t> --account-id <id>  # CLI login
chatwoot au status                                                      # Show current config
chatwoot au logout                                                      # Remove credentials
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
chatwoot o 123
chatwoot show https://app.chatwoot.com/app/accounts/1/conversations/123

# One-ID actions (no extra lookups)
chatwoot cmt 123 "Hello! How can I help?"
chatwoot n 123 "Internal note" --mention lily
chatwoot x 123 456
chatwoot ro 123
chatwoot ct 123 -o agent
```

### Conversations

```bash
# List and filter
chatwoot c ls
chatwoot c ls --st open --iid 1
chatwoot c ls --st open --at unassigned
chatwoot c ls --st open --tid 2 -L "vip,urgent"
chatwoot c ls --st open --search "refund"
chatwoot c ls -a --mp 50
chatwoot c filter --payload '[{"attribute_key":"status","filter_operator":"equal_to","values":["open"]}]'
chatwoot c search --query "refund"

# Get details
chatwoot c g 123
chatwoot c counts                              # Get counts by status
chatwoot c meta                                # Get metadata
chatwoot c attachments 123                     # List attachments

# Manage status and assignment
chatwoot c toggle-status 123 --st resolved
chatwoot c toggle-priority 123 --pri high
chatwoot c assign 123 --ag 5 --team 2
chatwoot c mark-unread 123

# Labels
chatwoot c labels 123
chatwoot c labels-add 123 --labels "urgent,vip"

# Custom attributes
chatwoot c custom-attributes 123 --payload '{"key":"value"}'

# AI context
chatwoot c context 123 --embed                 # Get full context for AI
chatwoot c context 123 -o agent                # Agent-friendly envelope
chatwoot c context 123 -o agent --rn

# Transcript
chatwoot c transcript 123                      # Render transcript locally
chatwoot c transcript 123 --public-only        # Exclude private notes
chatwoot c transcript 123 -l 200               # Limit to most recent messages
chatwoot c transcript 123 --email user@example.com
```

### Messages

```bash
# List messages
chatwoot m ls 123
chatwoot m ls 123 -a
chatwoot m ls 123 -l 500
chatwoot m ls 123 -o agent --rn

# Create messages
chatwoot m cr 123 -c "Hello!"

# Update and delete
chatwoot m up 123 456 -c "Updated text"
chatwoot m del 123 456
```

> **Note:** Messages are returned in chronological order (oldest first, most recent at end of array).
> To get the last N messages: `chatwoot m ls 123 --json | jq '.items[-N:]'`

### Private Notes & Mentions

Private notes are internal messages visible only to agents, not customers. You can mention/tag agents to notify them.

```bash
# Create a private note (internal, not visible to customer)
chatwoot m cr 123 -P -c "Internal note for the team"

# Mention an agent (they'll receive a notification)
chatwoot m cr 123 -P --mention lily -c "Can you follow up on this?"

# Mention multiple agents
chatwoot m cr 123 -P --mention lily --mention jack -c "Please review together"

# Mention by email
chatwoot m cr 123 -P --mention lily@example.com -c "Check this out"
```

The `--mention` flag:
- Accepts agent name (partial match) or email
- Automatically resolves to the agent's ID
- Formats the mention correctly so the agent receives a notification
- Requires the `-P` flag (mentions only work in private notes)

### Contacts

```bash
# List and search
chatwoot co ls
chatwoot co ls --sort name --order asc
chatwoot co ls --sort -last_activity_at
chatwoot co search --query "john"
chatwoot co filter --payload '[{"attribute_key":"email","filter_operator":"contains","values":["@example.com"]}]'

# CRUD operations
chatwoot co g 123
chatwoot co show 123                           # Alias for 'get'
chatwoot co g +16042091231                     # Lookup by phone number
chatwoot co cr -n "John Doe" -e "john@example.com"
chatwoot co up 123 --phone "+1234567890"
chatwoot co up john@example.com -n "John Smith"
chatwoot co up +16042091231 -n "Wenqi Qu" -e "quwenqi@example.com"
chatwoot co del 123

# Merge contacts (combine duplicates)
chatwoot co merge 123 456                      # Merge 456 INTO 123 (456 deleted)
chatwoot co merge 123 456 -y                   # Skip confirmation

# Related data
chatwoot co conversations 123
chatwoot co contactable-inboxes 123

# Inbox association
chatwoot co create-inbox 123 --iid 1 --source-id "+15551234567"

# Labels
chatwoot co labels 123
chatwoot co labels-add 123 --labels "customer,premium"

# Bulk label operations
chatwoot co bulk add-label --ids 1,2,3 --labels "vip,priority"
chatwoot co bulk remove-label --ids 1,2,3 --labels "old-tag"

# Bulk IDs can also come from stdin/files: @- or @path
printf "1\n2\n3\n" | chatwoot co bulk add-label --ids @- --labels vip

# Notes
chatwoot co notes 123
chatwoot co notes-add 123 --content "Called about refund"
chatwoot co notes-delete 123 456
```

### Campaigns

```bash
# List and get
chatwoot cm ls
chatwoot cm ls -a
chatwoot cm g 123

# Create campaign
chatwoot cm cr --title "Welcome" -m "Hello!" --iid 1 --labels 5,6

# Update and delete
chatwoot cm up 123 --enabled=false
chatwoot cm del 123 -y
```

### Help Center (Portals)

```bash
# Portals
chatwoot po ls
chatwoot po g help
chatwoot po cr -n "Help Center" --slug help
chatwoot po up help -n "Support Center"
chatwoot po del help

# Articles
chatwoot po articles ls help
chatwoot po articles cr help --title "Getting Started" --content "..." --category-id 1
chatwoot po articles up help 123 --status 1    # 0=draft, 1=published, 2=archived
chatwoot po articles del help 123

# Categories
chatwoot po categories ls help
chatwoot po categories cr help -n "FAQ" --slug faq
chatwoot po categories up help faq -n "Frequently Asked Questions"
chatwoot po categories del help faq
```

### Inboxes

```bash
chatwoot in ls
chatwoot in g 1
chatwoot in cr -n "Support" --channel-type api --greeting-enabled --greeting-message "Hi!"
chatwoot in up 1 --timezone "America/New_York" --working-hours-enabled
```

### Inbox Members

```bash
chatwoot inbox-members ls --iid 1
chatwoot inbox-members add --iid 1 --user-id 5
chatwoot inbox-members up --iid 1 --user-id 5 --role administrator
chatwoot inbox-members remove --iid 1 --user-id 5
```

### Agents & Teams

```bash
# Agents
chatwoot a ls
chatwoot a g 5

# Teams
chatwoot t ls
chatwoot t g 1
chatwoot t members 1
```

### Canned Responses

```bash
chatwoot canned-responses ls
chatwoot canned-responses g 123
chatwoot canned-responses cr --short-code "greeting" --content "Hello! How can I help?"
chatwoot canned-responses up 123 --content "Updated response"
chatwoot canned-responses del 123
```

### Webhooks

```bash
chatwoot wh ls
chatwoot wh g 123
chatwoot wh cr --url "https://example.com/webhook" --subscriptions "message_created,conversation_created"
chatwoot wh up 123 --url "https://new.example.com/hook"
chatwoot wh del 123
```

### Automation Rules

```bash
chatwoot automation-rules ls
chatwoot automation-rules g 123
```

### Agent Bots

```bash
chatwoot agent-bots ls
chatwoot agent-bots g 123
chatwoot agent-bots cr -n "Support Bot" --desc "Handles FAQs"
chatwoot agent-bots up 123 -n "FAQ Bot"
chatwoot agent-bots del 123
```

### Custom Attributes

```bash
chatwoot custom-attributes ls --attribute-model contact_attribute
chatwoot custom-attributes g 123 --attribute-model contact_attribute
chatwoot custom-attributes cr --attribute-model contact_attribute --attribute-key "vip_status" --attribute-display-name "VIP Status" --attribute-display-type list --attribute-values '["gold","silver","bronze"]'
chatwoot custom-attributes up 123 --attribute-model contact_attribute --attribute-display-name "VIP Level"
chatwoot custom-attributes del 123 --attribute-model contact_attribute
```

### Custom Filters

```bash
chatwoot custom-filters ls --filter-type conversation
chatwoot custom-filters g 123 --filter-type conversation
chatwoot custom-filters cr --filter-type conversation -n "High Priority Open" --query '{"payload":[{"attribute_key":"status","filter_operator":"equal_to","values":["open"]},{"attribute_key":"priority","filter_operator":"equal_to","values":["high"]}]}'
chatwoot custom-filters up 123 --filter-type conversation -n "Updated Filter"
chatwoot custom-filters del 123 --filter-type conversation
```

### Labels

```bash
chatwoot l ls
chatwoot l g 123
chatwoot l cr --title "VIP"
chatwoot l up 123 --title "Premium Customer"
chatwoot l del 123
```

### Integrations

```bash
chatwoot integrations ls
chatwoot integrations hooks ls --app-id slack
chatwoot integrations hooks cr --app-id slack --iid 1 --settings '{"webhook_url":"https://hooks.slack.com/..."}'
chatwoot integrations hooks up 123 --app-id slack --settings '{"webhook_url":"https://new.hooks.slack.com/..."}'
chatwoot integrations hooks del 123 --app-id slack
```

### Platform APIs

```bash
# Accounts
chatwoot pf accounts cr -n "Acme Inc" --domain acme.example.com --support-email support@acme.example.com
chatwoot pf accounts g 123
chatwoot pf accounts del 123

# Users
chatwoot pf users cr -n "Jane Doe" --email jane@example.com --password "secret"
chatwoot pf users up 45 --display-name "Jane D"
chatwoot pf users del 45

# Account users
chatwoot pf account-users ls 123
chatwoot pf account-users cr 123 --user-id 45 --role agent
chatwoot pf account-users del 123 --user-id 45
```

### Public Client APIs

```bash
# Contacts
chatwoot client contacts cr --inbox <inbox-identifier> -n "Visitor" -e visitor@example.com
chatwoot client contacts g --inbox <inbox-identifier> --contact <contact-identifier>

# Conversations
chatwoot client conversations ls --inbox <inbox-identifier> --contact <contact-identifier>
chatwoot client conversations cr --inbox <inbox-identifier> --contact <contact-identifier>
chatwoot client conversations g 123 --inbox <inbox-identifier> --contact <contact-identifier>
chatwoot client conversations resolve 123 --inbox <inbox-identifier> --contact <contact-identifier>

# Messages & typing
chatwoot client messages cr 123 --inbox <inbox-identifier> --contact <contact-identifier> -c "Hello!"
chatwoot client typing 123 --inbox <inbox-identifier> --contact <contact-identifier> --status on
chatwoot client last-seen up 123 --inbox <inbox-identifier> --contact <contact-identifier>
```

### Reports

```bash
chatwoot rp summary --since 2024-01-01 --until 2024-12-31
chatwoot rp summary --metric conversations_count --type account
```

### CSAT (Customer Satisfaction)

```bash
chatwoot cs ls
chatwoot cs metrics --since 2024-01-01 --until 2024-12-31
```

### Audit Logs

```bash
chatwoot audit-logs ls
chatwoot audit-logs ls --user-id 5
```

### Profile & Account

```bash
chatwoot pr g                                  # Get your user profile
chatwoot account g                             # Get account details
```

## Output Formats

### Text

Human-readable tables:

```bash
$ chatwoot c ls
ID      STATUS    INBOX         CONTACT           MESSAGES    UPDATED
123     open      Support       John Doe          5           2 hours ago
124     pending   Sales         Jane Smith        3           1 day ago
```

### JSON

Machine-readable output:

```bash
$ chatwoot c ls -o json
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

Tip: `--json` is a shorthand for `-o json`.

### Agent JSON

Agent-optimized JSON output with consistent `kind` envelopes, compact summaries, and readable timestamps.
Enable it per command with `-o agent`, or set a default for your shell:

```bash
export CHATWOOT_OUTPUT=agent
```

Use `--rn` to fetch inbox/contact names for conversation results (extra API calls).
Agent output is available for `c context` with message metadata and summary.

**Context commands return a single envelope**:
```bash
chatwoot c context 123 -o agent | jq '.item.summary'
chatwoot c context 123 -o agent --rn | jq '.item.contact_labels'
```

**List commands return an object with an "items" array** (plus `has_more` and `meta`):
```bash
chatwoot co ls -o agent | jq '.items[0]'
chatwoot c ls -o agent | jq '.items[] | select(.status == "open")'
```

**Get commands return single objects**:
```bash
chatwoot co g 123 -o agent | jq '.item.email'
chatwoot c g 456 -o agent | jq '.item.messages_count'
```

Data goes to stdout, errors and progress to stderr for clean piping.

### JSONL

Streaming-friendly output (one JSON object per line):

```bash
$ chatwoot c ls -o jsonl
{"id":123,"status":"open",...}
{"id":124,"status":"pending",...}
```

Tip: `--query` and `--template` apply per line in JSONL mode.

## Examples

### Triage open conversations

```bash
# List all open conversations
chatwoot c ls --st open -o json | \
  jq '.items[] | {id, contact: .contact.name, messages: .messages_count}'

# Assign high-priority to agent
chatwoot c assign 123 --ag 5
chatwoot c toggle-priority 123 --pri high
```

### Send bulk campaign

```bash
# Create campaign for specific labels
chatwoot cm cr \
  --title "Product Update" \
  -m "Check out our new feature!" \
  --iid 1 \
  --labels 10,11,12
```

### Build AI support bot

```bash
# Get conversation context with images for AI vision
chatwoot c context 123 --embed -o json > context.json

# Process with AI (example using OpenAI or similar)
cat context.json | your-ai-tool --prompt "Draft a helpful response"

# Send the response
chatwoot m cr 123 -c "Your drafted response here"
```

### Search contacts by email domain

```bash
chatwoot co filter \
  --payload '[{"attribute_key":"email","filter_operator":"contains","values":["@example.com"]}]' \
  -o json | jq '.items[].email'
```

### Export conversations for analysis

```bash
# Get all resolved conversations from last month
chatwoot c filter \
  --payload '[{"attribute_key":"status","filter_operator":"equal_to","values":["resolved"]}]' \
  -o json > resolved-conversations.json

# Extract key metrics
jq '[.items[] | {id, resolved_at: .updated_at, messages: .messages_count, agent: .assignee.name}]' \
  resolved-conversations.json
```

### Manage help center content

```bash
# Create portal
chatwoot po cr -n "Knowledge Base" --slug kb

# Create category
chatwoot po categories cr kb -n "Getting Started" --slug getting-started

# Publish article
chatwoot po articles cr kb \
  --title "How to Get Started" \
  --content "..." \
  --category-id 1 \
  --status 1
```

### AI Context

Get complete conversation data optimized for use by LLMs:

```bash
# Text format with embedded images
chatwoot c context 123 --embed

# JSON format for programmatic access
chatwoot c context 123 --embed -o json
```

The `--embed` flag converts images to base64 data URIs that AI vision models can process directly.

### Pagination

List commands support pagination:

```bash
# Get all results (automatic pagination)
chatwoot c ls -a

# Limit pagination depth
chatwoot c ls -a --mp 50
```

### Filtering and Search

**Filter** uses Chatwoot's filter API for structured queries:
```bash
chatwoot c filter --payload '[{"attribute_key":"status","filter_operator":"equal_to","values":["open"]}]'
chatwoot co filter --payload '[{"attribute_key":"email","filter_operator":"contains","values":["@vip.com"]}]'
```

**Search** performs full-text search:
```bash
chatwoot c search --query "refund request"
chatwoot co search --query "john smith"
```

## Common Workflows

### Inbox -> Conversation -> Message

```bash
# Pick an inbox
chatwoot in ls

# List open conversations in an inbox
chatwoot c ls --st open --iid 1

# Send a message in a conversation
chatwoot m cr 123 -c "Hello! How can I help?"
```

### Contacts -> Conversations

```bash
# Find a contact
chatwoot co search --query "john@example.com"

# Get the conversations for a contact
chatwoot co conversations 123
chatwoot co conversations 123 -o agent --rn
```

## Troubleshooting

- **401 Unauthorized**: run `chatwoot au login` and verify your token.
- **403 Forbidden**: check your account role and permissions.
- **404 Not Found**: verify the resource ID (it may have been deleted).
- **URL validation failed**: ensure the base URL is public, or use `--allow-private` only if you trust the target.
- **Base URL not configured**: set `CHATWOOT_BASE_URL` / `CHATWOOT_API_TOKEN` / `CHATWOOT_ACCOUNT_ID` or run `chatwoot au login`.

## Global Flags

All commands support these flags:

- `-o <format>` / `--output <format>` - Output format: `text`, `json`, or `jsonl` (default: text)
- `--json` - Alias for `-o json`
- `--color <mode>` - Color mode: `auto`, `always`, or `never` (default: auto)
- `--allow-private` - Allow private/localhost URLs (unsafe)
- `--debug` - Enable verbose debug logging
- `--dr` / `--dry-run` - Preview changes without executing mutations
- `--timeout <duration>` - HTTP request timeout (default: 30s)
- `--idem <key|auto>` / `--idempotency-key <key|auto>` - Idempotency key for write requests (use `auto` for per-request keys)
- `--query <expr>` / `--jq <expr>` - JQ expression to filter JSON output
- `--fields <a,b,c>` - Select fields in JSON output (shorthand for `--query`; supports presets like `minimal`, `default`, `debug` on supported resources)
- `-q` / `--quiet` - Suppress non-essential output
- `--silent` - Suppress non-error output to stderr
- `--no-input` - Disable interactive prompts
- `-y` / `--yes` - Assume yes for confirmations (desire path alias for `--force`)
- `--template <tmpl>` - Go template (or `@path`) to render JSON output
- `--utc` - Display timestamps in UTC
- `--tz <tz>` / `--time-zone <tz>` - Display timestamps in a specific time zone (e.g., `America/Los_Angeles`)
- `--max-rl <n>` / `--max-rate-limit-retries <n>` - Max retries for HTTP 429 responses
- `--max-5xx-retries <n>` - Max retries for HTTP 5xx responses
- `--rld <duration>` / `--rate-limit-delay <duration>` - Base delay for 429 retries (e.g., 1s)
- `--sed <duration>` / `--server-error-delay <duration>` - Delay between 5xx retries (e.g., 1s)
- `--circuit-breaker-threshold <n>` - Failures before circuit opens
- `--circuit-breaker-reset-time <duration>` - Circuit breaker reset time (e.g., 30s)
- `--help` - Show help for any command

Note: `--utc` and `--tz` are mutually exclusive.

You can force interactive prompts in non-TTY environments by setting `CHATWOOT_FORCE_INTERACTIVE=true`.

### Global Flag Aliases

Frequently used global flags have short aliases:

| Flag | Alias |
|------|-------|
| `--dry-run` | `--dr` |
| `--resolve-names` | `--rn` |
| `--time-zone` | `--tz` |
| `--help-json` | `--hj` |
| `--idempotency-key` | `--idem` |
| `--max-rate-limit-retries` | `--max-rl` |
| `--rate-limit-delay` | `--rld` |
| `--server-error-delay` | `--sed` |

## Command Aliases

Every command has 1-2 letter aliases for fast typing. Use `chatwoot <alias>` instead of the full command name:

| Command | Aliases |
|---------|---------|
| `conversations` | `conv`, `c` |
| `messages` | `msg`, `m` |
| `contacts` | `co` |
| `search` | `s` |
| `reply` | `r` |
| `note` | `n` |
| `comment` | `cmt` |
| `close` | `resolve`, `x` |
| `reopen` | `ro` |
| `assign` | `as` |
| `ctx` | `ct` |
| `open` | `get`, `show`, `o` |
| `agents` | `a` |
| `teams` | `t` |
| `labels` | `l` |
| `inboxes` | `in` |
| `campaigns` | `camp`, `cm` |
| `webhooks` | `wh` |
| `reports` | `rp` |
| `snooze` | `sn` |
| `handoff` | `ho` |
| `mentions` | `mn` |
| `csat` | `cs` |
| `portals` | `po` |
| `platform` | `pf` |
| `schema` | `sc` |
| `config` | `cfg` |
| `auth` | `au` |
| `profile` | `pr` |
| `version` | `v` |

Subcommands also have aliases (e.g., `list` -> `ls`, `get` -> `g`, `create` -> `cr`, `update` -> `up`, `delete` -> `del`).

### Examples

```bash
# These are equivalent:
chatwoot conversations list --status open
chatwoot c ls --st open

chatwoot messages list 123 --limit 50
chatwoot m ls 123 -l 50

chatwoot search "refund" --type conversations --limit 5
chatwoot s "refund" -t conversations -l 5

chatwoot contacts create --name "John" --email "john@example.com"
chatwoot co cr -n "John" -e "john@example.com"

chatwoot close 123
chatwoot x 123
```

## Flag Aliases

Commonly used flags have short aliases to reduce typing. Single-letter aliases appear in `--help` output. Multi-letter aliases (like `--st`, `--iid`) are hidden from help but work the same way.

### Single-Letter Flag Aliases

| Flag | Alias | Available on |
|------|-------|-------------|
| `--content` | `-c` | messages create/update, comment, note, reply |
| `--name` | `-n` | contacts create/update, campaigns, inboxes, platform |
| `--email` | `-e` | contacts create/update |
| `--limit` | `-l` | messages list, search, all list commands |
| `--type` | `-t` | search |
| `--page` | `-p` | contacts list, all list commands |
| `--all` | `-a` | all list commands |
| `--emit` | `-E` | agents, campaigns, contacts, conversations, inboxes, teams, search, webhooks, ref |
| `--labels` | `-L` | conversations list, campaigns |
| `--private` | `-P` | messages create, conversations typing, reply |
| `--since` | `-S` | conversations list, mentions, reports |
| `--resolve` | `-R` | comment, note, reply |
| `--message` | `-m` | campaigns create, inboxes |

### Multi-Letter Flag Aliases

| Flag | Alias | Available on |
|------|-------|-------------|
| `--status` | `--st` | conversations list/create/update |
| `--inbox-id` | `--iid` | conversations, campaigns, contacts, csat, integrations |
| `--contact-id` | `--cid` | conversations create, integrations, reply |
| `--team-id` | `--tid` | conversations list/create |
| `--priority` | `--pri` | conversations, comment, note, reply |
| `--agent` | `--ag` | assign, conversations, handoff |
| `--description` | `--desc` | campaigns, labels, platform, portals, teams |
| `--assignee-type` | `--at` | conversations list |
| `--unread-only` | `--unread` | conversations list |
| `--max-pages` | `--mp` | all list commands, conversations, messages |
| `--concurrency` | `--cc` | contacts bulk, conversations bulk, messages |
| `--since-last-agent` | `--sla` | messages list |
| `--transcript` | `--tr` | messages list |
| `--snooze-for` | `--for` | comment, note, reply |
| `--include-snippet` | `--snippet` | search |
| `--embed-images` | `--embed` | conversations context, ctx |
| `--context-messages` | `--cm` | conversations follow |
| `--only-unassigned` | `--unassigned` | conversations follow |
| `--exclude-private` | `--pub` | conversations follow |

### JQ Filtering

Filter JSON output with JQ expressions:

```bash
# Get only conversation IDs
chatwoot c ls -o json --query '.items[].id'

# Filter by status
chatwoot c ls -o json --query '.items[] | select(.status == "open")'
```

**Fields shorthand & templates**

```bash
# Select fields without writing JQ
chatwoot c ls -o json --fields id,status,assignee_id

# Render custom output with a template
chatwoot c g 123 -o json --template '{{.id}} {{.status}}'
```

### Field Presets

Many commands support field presets for common use cases. Instead of listing individual fields, use a preset name:

```bash
# Minimal output - just essential identifiers
chatwoot co ls -o json --fields minimal

# Default output - commonly needed fields
chatwoot c ls -o json --fields default

# Debug output - all fields for troubleshooting
chatwoot co ls -o json --fields debug
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
chatwoot v
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
