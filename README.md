# Chatwoot CLI

Chatwoot in your terminal. Manage conversations, contacts, campaigns, help center, and integrations.

## Features

- **Automations** - list and view automation rules, manage canned response templates
- **Bots** - create, update, delete bots
- **Campaigns** - create and manage SMS and messaging campaigns
- **Contacts** - create, update, search, filter, merge duplicates, bulk operations, manage labels and notes
- **Conversations** - list, filter, search, assign, status, priority, labels
- **Dashboards** - query external dashboard APIs for contact data
- **Help Center** - manage portals, articles, and categories
- **Inboxes** - list and view inbox details, member access and roles, create and manage saved filter presets
- **Mentions** - view @mentions of the current user across conversations
- **Messages** - send, edit, delete messages and list attachments
- **Platform APIs** - manage accounts and users (self-hosted/managed)
- **Public APIs** - unauthenticated widget/client-side operations via inbox identifiers
- **Profiles** - store multiple accounts/tokens and switch contexts quickly
- **Raw API** - make direct API calls to any Chatwoot endpoint
- **Real-time** - follow conversations via WebSocket with filtering, debouncing, and exec hooks
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
# Binary at ./bin/cw
```

## Quick Start

### 1. Authenticate

**Browser:**
```bash
cw auth login                            # Opens browser for interactive login
```

**Terminal:**
```bash
cw auth login --browser=false --url https://chatwoot.example.com --token YOUR_API_TOKEN --account-id 1

# Load credentials from a .env file
cw auth login --env-file .env
```

### 2. Verify Setup

```bash
cw pr                                    # Get current user profile (default: profile get)
```

### 3. List Conversations

```bash
cw c ls --st open                        # List open conversations
```

### 4. Search (Agent-Friendly)

```bash
cw s "john"                              # Search across contacts + conversations
cw s "refund" --best                     # Auto-pick the best result (no interactive prompt)
cw s "refund" --best --emit id           # Emit just an ID for chaining (no jq)
```

## Configuration

### Authentication

Credentials are checked in this order:
1. Environment variables (`CHATWOOT_BASE_URL`, `CHATWOOT_API_TOKEN`, `CHATWOOT_ACCOUNT_ID`)
2. `CHATWOOT_PROFILE` (if set)
3. Current profile in keychain (defaults to `default`)

Check current configuration:
```bash
cw auth status                           # Show current config
cw auth status --json                    # Show current config as JSON
```

Remove stored credentials:
```bash
cw auth logout                           # Remove stored credentials from keychain
```

### Profiles

Manage multiple stored accounts:
```bash
cw cfg profiles ls                       # List all profiles
cw cfg profiles use staging              # Switch to staging profile
cw cfg profiles show --name staging      # Show profile details
cw cfg profiles del staging              # Delete a profile
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

# Optional keyring controls (useful for headless Linux/CI)
export CW_KEYRING_BACKEND=auto            # auto | file | system
export CW_KEYRING_PASSWORD=strong-secret  # required for non-interactive file backend
export CW_CREDENTIALS_DIR=~/.config/chatwoot-cli

# Optional contact --light custom-attribute mapping (for tier/store IDs)
export CW_CONTACT_LIGHT_TIER_KEY=membership_tier
export CW_CONTACT_LIGHT_STORE_KEYS='store_a:store_key_1,store_b:store_key_2'
# or JSON:
# export CW_CONTACT_LIGHT_STORE_KEYS='{"store_a":"store_key_1","store_b":"store_key_2"}'
# In contact --li output, each alias becomes a top-level key (flat JSON).
```

## Security

### Credential Storage

Credentials saved with `cw auth login` are stored securely in your system's keychain:
- **macOS**: Keychain Access
- **Linux**: Secret Service (GNOME Keyring, KWallet)
- **Windows**: Credential Manager

Headless Linux fallback:
- If `DBUS_SESSION_BUS_ADDRESS` is missing, `cw` automatically uses the encrypted file backend.
- Default file location: `~/.config/chatwoot-cli/keyring/`
- For non-interactive environments (CI/systemd), set `CW_KEYRING_PASSWORD`.

## Commands

### Authentication

```bash
cw auth login                            # Authenticate via browser
cw auth login --no-browser --url <url> --token <t> --account-id <id>  # CLI login
cw auth login --env-file .env           # Load CHATWOOT_* (and CW_KEYRING_*) vars from .env
cw auth status                           # Show current config
cw auth logout                           # Remove credentials
```

## Extensions

Executables on your PATH named `cw-<name>` can be invoked as:

```bash
cw <name> [args...]
```

Extension shortcuts are also supported where configured:

```bash
cw view-images [args...]
cw vi [args...]                         # alias for cw-view-images
```

If you invoke extension executables directly, a shell-level alias works too:

```bash
ln -sf "$(command -v cw-view-images)" ~/.local/bin/cw-vi
```

### Shortcuts (Agent-Friendly)

Convenience commands designed for agent workflows:

```bash
cw o 123                                 # Open conversation details
cw show https://app.chatwoot.com/app/accounts/1/conversations/123  # Open from URL
cw cmt 123 "Hello! How can I help?"      # Send a public reply
cw n 123 "Internal note" --mention lily   # Add private note with @mention
cw r 123 "Thanks for contacting us!"     # Reply to conversation
cw x 123 456                             # Close (resolve) conversations
cw ro 123                                # Reopen a closed conversation
cw sn 123 --for 2h                       # Snooze for 2 hours
cw ho 123 --ag lily --reason "Needs billing"  # Escalate with reason
cw as 123 --ag 5 --team 2                # Assign to agent and team
cw ct 123 -o agent                       # Get AI context for conversation
cw ref 123                               # Resolve ID to typed reference
```

### Conversations

```bash
cw c ls                                  # List all conversations
cw c ls --st open --iid 1                # Filter by status and inbox
cw c ls --st open --at unassigned        # Filter by assignee type
cw c ls --st open --tid 2 -L "vip,urgent"  # Filter by team and labels
cw c ls --st open --search "refund"      # Full-text search within list
cw c ls -a --mp 50                       # Paginate all, max 50 pages
cw c filter --payload '[{"attribute_key":"status","filter_operator":"equal_to","values":["open"]}]'  # Structured filter
cw c search --query "refund"             # Full-text search
cw c g 123                               # Get conversation details
cw c counts                              # Get counts by status
cw c meta                                # Get account metadata
cw c attachments 123                     # List attachments for conversation
cw c toggle-status 123 --st resolved     # Change conversation status
cw c toggle-priority 123 --pri high      # Set conversation priority
cw c assign 123 --ag 5 --team 2          # Assign to agent and team
cw c mark-unread 123                     # Mark conversation as unread
cw c labels 123                          # List labels on conversation
cw c labels-add 123 --labels "urgent,vip"  # Add labels to conversation
cw c custom-attributes 123 --payload '{"key":"value"}'  # Set custom attributes
cw c context 123 --embed                 # Get full AI context with embedded images
cw c context 123 -o agent                # Agent-friendly context envelope
cw c context 123 -o agent --rn           # Context with resolved names
cw c transcript 123                      # Render transcript locally
cw c transcript 123 --public-only        # Transcript excluding private notes
cw c transcript 123 -l 200               # Transcript limited to 200 messages
cw c transcript 123 --email user@example.com  # Email transcript to address
cw c follow 123                          # Follow conversation via WebSocket
cw c follow 123 --tail 50               # Show last 50 messages then stream
cw c follow --all                        # Follow all account conversations
cw c follow --all --events all           # All event types (status, assignments, etc.)
cw c follow 123 --typing                 # Include typing indicators
cw c follow 123 -o agent --rn            # Agent output with resolved names
cw c follow --all --inbox 1 --status open  # Filter by inbox and status
cw c follow --all --label vip --pri urgent  # Filter by label and priority
cw c follow --all --assignee 5           # Only agent 5's conversations
cw c follow --all --unassigned           # Only unassigned conversations
cw c follow --all --pub                  # Exclude private messages
cw c follow 123 --debounce 2s            # Batch rapid messages (2s window)
cw c follow 123 --context --cm 20        # Emit snapshot with 20 context messages
cw c follow 123 --cursor-file .cursor    # Resume from last position on restart
cw c follow 123 --since-id 456           # Skip messages with id <= 456
cw c follow 123 --since-time 24h         # Skip messages older than 24h
cw c follow 123 --raw                    # Include raw WebSocket payload
cw c follow 123 --exec './handler.sh'    # Pipe each event JSON to command
cw c follow 123 --exec './handler.sh' --exec-fatal  # Abort on handler failure
```

> **Real-time streaming:** `follow` connects directly to Chatwoot's ActionCable WebSocket.
> No webhook setup required. Reconnects automatically with exponential backoff.
> In agent mode (`-o agent`), conversation snapshots are emitted automatically on first event.

### Messages

```bash
cw m ls 123                              # List messages in conversation
cw m ls 123 -a                           # List all messages (paginated)
cw m ls 123 -l 500                       # List up to 500 messages
cw m ls 123 --li                         # Light: minimal message payload for quick lookup
cw m ls 123 -o agent --rn                # Agent output with resolved names
cw m cr 123 -c "Hello!"                  # Send a message
cw m up 123 456 -c "Updated text"        # Update message 456
cw m del 123 456                         # Delete message 456
```

> **Note:** Messages are returned in chronological order (oldest first, most recent at end of array).
> To get the last N messages: `cw m ls 123 --json | jq '.items[-N:]'`

### Private Notes & Mentions

Private notes are internal messages visible only to agents, not customers. You can mention/tag agents to notify them.

```bash
cw m cr 123 -P -c "Internal note for the team"  # Create private note
cw m cr 123 -P --mention lily -c "Can you follow up on this?"  # Mention an agent
cw m cr 123 -P --mention lily --mention jack -c "Please review together"  # Mention multiple agents
cw m cr 123 -P --mention lily@example.com -c "Check this out"  # Mention by email
```

The `--mention` flag:
- Accepts agent name (partial match) or email
- Automatically resolves to the agent's ID
- Formats the mention correctly so the agent receives a notification
- Requires the `-P` flag (mentions only work in private notes)

### Contacts

```bash
cw co ls                                 # List all contacts
cw co ls --sort name --order asc         # Sort by name ascending
cw co ls --sort -last_activity_at        # Sort by recent activity
cw co ls --sort la                       # Same as --sort last_activity_at
cw co search --query "john"              # Full-text search contacts
cw co filter --payload '[{"attribute_key":"email","filter_operator":"contains","values":["@example.com"]}]'  # Structured filter
cw co g 123                              # Get contact by ID
cw co show 123                           # Get contact (alias for get)
cw co g +16042091231                     # Lookup contact by phone number
cw co g 123 --li                         # Light: minimal contact payload + active conversations
cw co ls --li                            # Light list: compact contacts
cw co cr -n "John Doe" -e "john@example.com"  # Create a contact
cw co up 123 --phone "+1234567890"       # Update phone number
cw co up john@example.com -n "John Smith"  # Update by email lookup
cw co up +16042091231 -n "Wenqi Qu" -e "quwenqi@example.com"  # Update by phone lookup
cw co del 123                            # Delete a contact
cw co merge 123 456                      # Merge 456 into 123 (456 deleted)
cw co merge 123 456 -y                   # Merge without confirmation
cw co conversations 123                  # List conversations for contact
cw co conversations 123 --li             # Light: minimal conversation payloads for contact lookups
cw co contactable-inboxes 123            # List inboxes contact can reach
cw co create-inbox 123 --iid 1 --source-id "+15551234567"  # Associate contact with inbox
cw co labels 123                         # List labels on contact
cw co labels-add 123 --labels "customer,premium"  # Add labels to contact
cw co bulk add-label --ids 1,2,3 --labels "vip,priority"  # Bulk add labels
cw co bulk remove-label --ids 1,2,3 --labels "old-tag"  # Bulk remove labels
printf "1\n2\n3\n" | cw co bulk add-label --ids @- --labels vip  # Bulk IDs from stdin
cw co notes 123                          # List notes on contact
cw co notes-add 123 --content "Called about refund"  # Add a note
cw co notes-delete 123 456               # Delete note 456
```

### Campaigns

```bash
cw cm ls                                 # List all campaigns
cw cm ls -a                              # List all (paginated)
cw cm g 123                              # Get campaign details
cw cm cr --title "Welcome" -m "Hello!" --iid 1 --labels 5,6  # Create campaign
cw cm up 123 --enabled=false             # Disable campaign
cw cm del 123 -y                         # Delete without confirmation
```

### Help Center (Portals)

```bash
cw po ls                                 # List portals
cw po g help                             # Get portal by slug
cw po cr -n "Help Center" --slug help    # Create portal
cw po up help -n "Support Center"        # Update portal name
cw po del help                           # Delete portal
cw po articles ls help                   # List articles in portal
cw po articles cr help --title "Getting Started" --content "..." --category-id 1  # Create article
cw po articles up help 123 --status 1    # Update article (0=draft, 1=published, 2=archived)
cw po articles del help 123              # Delete article
cw po categories ls help                 # List categories in portal
cw po categories cr help -n "FAQ" --slug faq  # Create category
cw po categories up help faq -n "Frequently Asked Questions"  # Update category
cw po categories del help faq            # Delete category
```

### Inboxes

```bash
cw in ls                                 # List all inboxes
cw in g 1                                # Get inbox details
cw in cr -n "Support" --channel-type api --greeting-enabled --greeting-message "Hi!"  # Create inbox
cw in up 1 --timezone "America/New_York" --working-hours-enabled  # Update inbox settings
```

### Inbox Members

```bash
cw inbox-members ls --iid 1              # List members of inbox
cw inbox-members add --iid 1 --user-id 5  # Add agent to inbox
cw inbox-members up --iid 1 --user-id 5 --role administrator  # Update member role
cw inbox-members remove --iid 1 --user-id 5  # Remove agent from inbox
```

### Agents & Teams

```bash
cw a ls                                  # List all agents
cw a g 5                                 # Get agent details
cw t ls                                  # List all teams
cw t g 1                                 # Get team details
cw t members 1                           # List team members
```

### Canned Responses

```bash
cw canned-responses ls                   # List all canned responses
cw canned-responses g 123                # Get canned response
cw canned-responses cr --short-code "greeting" --content "Hello! How can I help?"  # Create template
cw canned-responses up 123 --content "Updated response"  # Update template
cw canned-responses del 123              # Delete template
```

### Webhooks

```bash
cw wh ls                                 # List all webhooks
cw wh g 123                              # Get webhook details
cw wh cr --url "https://example.com/webhook" --subscriptions "message_created,conversation_created"  # Create webhook
cw wh up 123 --url "https://new.example.com/hook"  # Update webhook URL
cw wh del 123                            # Delete webhook
```

### Automation Rules

```bash
cw automation-rules ls                   # List all automation rules
cw automation-rules g 123                # Get automation rule details
```

### Agent Bots

```bash
cw agent-bots ls                         # List all bots
cw agent-bots g 123                      # Get bot details
cw agent-bots cr -n "Support Bot" --desc "Handles FAQs"  # Create bot
cw agent-bots up 123 -n "FAQ Bot"        # Update bot name
cw agent-bots del 123                    # Delete bot
```

### Custom Attributes

```bash
cw custom-attributes ls --attribute-model contact_attribute  # List contact attributes
cw custom-attributes g 123 --attribute-model contact_attribute  # Get attribute
cw custom-attributes cr --attribute-model contact_attribute --attribute-key "vip_status" --attribute-display-name "VIP Status" --attribute-display-type list --attribute-values '["gold","silver","bronze"]'  # Create attribute
cw custom-attributes up 123 --attribute-model contact_attribute --attribute-display-name "VIP Level"  # Update attribute
cw custom-attributes del 123 --attribute-model contact_attribute  # Delete attribute
```

### Custom Filters

```bash
cw custom-filters ls --filter-type conversation  # List conversation filters
cw custom-filters g 123 --filter-type conversation  # Get filter details
cw custom-filters cr --filter-type conversation -n "High Priority Open" --query '{"payload":[{"attribute_key":"status","filter_operator":"equal_to","values":["open"]},{"attribute_key":"priority","filter_operator":"equal_to","values":["high"]}]}'  # Create filter
cw custom-filters up 123 --filter-type conversation -n "Updated Filter"  # Update filter
cw custom-filters del 123 --filter-type conversation  # Delete filter
```

### Labels

```bash
cw l ls                                  # List all labels
cw l g 123                               # Get label details
cw l cr --title "VIP"                    # Create label
cw l up 123 --title "Premium Customer"   # Update label title
cw l del 123                             # Delete label
```

### Integrations

```bash
cw integrations ls                       # List all integrations
cw integrations hooks ls --app-id slack   # List hooks for an app
cw integrations hooks cr --app-id slack --iid 1 --settings '{"webhook_url":"https://hooks.slack.com/..."}'  # Create hook
cw integrations hooks up 123 --app-id slack --settings '{"webhook_url":"https://new.hooks.slack.com/..."}'  # Update hook
cw integrations hooks del 123 --app-id slack  # Delete hook
```

### Platform APIs

```bash
cw pf accounts cr -n "Acme Inc" --domain acme.example.com --support-email support@acme.example.com  # Create account
cw pf accounts g 123                     # Get account details
cw pf accounts del 123                   # Delete account
cw pf users cr -n "Jane Doe" --email jane@example.com --password "secret"  # Create user
cw pf users up 45 --display-name "Jane D"  # Update user display name
cw pf users del 45                       # Delete user
cw pf account-users ls 123               # List users in account
cw pf account-users cr 123 --user-id 45 --role agent  # Add user to account
cw pf account-users del 123 --user-id 45  # Remove user from account
```

### Public Client APIs

```bash
cw client contacts cr --inbox <inbox-identifier> -n "Visitor" -e visitor@example.com  # Create contact via inbox
cw client contacts g --inbox <inbox-identifier> --contact <contact-identifier>  # Get contact
cw client conversations ls --inbox <inbox-identifier> --contact <contact-identifier>  # List conversations
cw client conversations cr --inbox <inbox-identifier> --contact <contact-identifier>  # Create conversation
cw client conversations g 123 --inbox <inbox-identifier> --contact <contact-identifier>  # Get conversation
cw client conversations resolve 123 --inbox <inbox-identifier> --contact <contact-identifier>  # Resolve conversation
cw client messages cr 123 --inbox <inbox-identifier> --contact <contact-identifier> -c "Hello!"  # Send message
cw client typing 123 --inbox <inbox-identifier> --contact <contact-identifier> --status on  # Send typing indicator
cw client last-seen up 123 --inbox <inbox-identifier> --contact <contact-identifier>  # Update last seen
```

### Reports

```bash
cw rp summary --since 2024-01-01 --until 2024-12-31  # Get summary report
cw rp summary --metric conversations_count --type account  # Specific metric
```

### CSAT (Customer Satisfaction)

```bash
cw cs ls                                 # List CSAT responses
cw cs metrics --since 2024-01-01 --until 2024-12-31  # Get CSAT metrics
```

### Audit Logs

```bash
cw audit-logs ls                         # List all audit logs
cw audit-logs ls --user-id 5             # Filter by user
```

### Profile & Account

```bash
cw pr g                                  # Get your user profile
cw account g                             # Get account details
```

### Status

```bash
cw st                                    # Show config and auth status
cw st -o json                            # Status as JSON
cw st --check                            # Exit code 1 if not authenticated
```

### Raw API Requests

Make direct API calls to any Chatwoot endpoint (like `gh api` for GitHub):

```bash
cw ap /conversations/123                 # GET request (default)
cw ap /conversations -X POST -f inbox_id=1 -f contact_id=5  # POST with fields
cw ap /conversations/123 -X PATCH -F 'labels=["bug","urgent"]'  # PATCH with JSON array
cw ap /automation_rules/14 -X PATCH -d '{"automation_rule":{"active":true}}'  # Inline JSON body
cw ap /contacts -X POST -i body.json     # Body from file
echo '{"name":"Test"}' | cw ap /contacts -X POST -i -  # Body from stdin
cw ap /contacts -o json --jq '.payload[0].name'  # Filter response with jq
cw ap /conversations/123 -X DELETE -s    # Silent mode (no output)
cw ap /conversations/123 --include       # Show response headers
```

### Snooze

```bash
cw sn 123 --for 2h                       # Snooze for 2 hours
cw sn 123 --for 24h                      # Snooze for 1 day
cw sn 123 --for 2h --note "Waiting for customer reply"  # Snooze with private note
```

### Handoff (Escalate / Transfer)

Composite command that sends a private note + assigns + sets priority in one step:

```bash
cw ho 123 --ag 5 --reason "Refund request, needs billing approval"  # Handoff to agent
cw ho 123 --team billing --reason "Technical issue beyond L1"  # Handoff to team
cw ho 123 --ag 5 --team billing --priority urgent --reason "VIP customer, SLA at risk"  # Full escalation
```

### Mentions

```bash
cw mn ls                                 # List all recent mentions
cw mn ls -S 24h                          # Mentions from last 24 hours
cw mn ls -S 7d -l 20                     # Last 7 days, limit 20
cw mn ls --conversation-id 123           # Mentions in a specific conversation
```

### Reference Resolver

Normalize IDs, URLs, and typed prefixes into canonical typed IDs for agent workflows:

```bash
cw ref 123                               # Probe conversation + contact
cw ref conversation:123                  # Explicit type (no probe)
cw ref https://app.chatwoot.com/.../123  # Parse from URL
cw ref 123 --type contact                # Force resource type
cw ref 123 --try conversation --try inbox  # Custom probe order
cw ref 123 -E url                        # Emit just the UI URL
cw ref 123 -E id                         # Emit just the typed ID
```

### Dashboard

Query external dashboard APIs for contact data (requires configuration via `cw cfg dashboard add`):

```bash
cw dh ods --ct 180712                     # Short form: dashboard alias + fuzzy name + contact alias
cw dh ods --cv 24445                      # Resolve contact from conversation alias
cw dh ods --ct 180712 --pg 2 --pp 20      # Paginate with short aliases
cw dh ods --ct 180712 -o json -q '[.it[-3:] | .[] | {n: .number, ot}]'
# `ods` must uniquely match a configured dashboard (e.g., "orders"). `ord` also works.
# Full forms still work: `cw dash orders --contact ...`
```

### Survey

```bash
cw sv g <conversation-uuid>              # Get survey response for a conversation
```

### Cache

```bash
cw ch clear                              # Clear all cached data
cw ch path                               # Show cache directory and files
```

### Public API (Unauthenticated)

Widget/client-side API using inbox identifiers instead of account auth:

```bash
cw pub inboxes g <inbox-id>              # Get inbox info
cw pub contacts mk <inbox-id> --name "Visitor" --email visitor@example.com  # Create contact
cw pub contacts g <inbox-id> <contact-id>  # Get contact
cw pub contacts up <inbox-id> <contact-id> --name "Updated Name"  # Update contact
cw pub conversations ls <inbox-id> <contact-id>  # List conversations
cw pub conversations mk <inbox-id> <contact-id>  # Create conversation
cw pub conversations g <inbox-id> <contact-id> 123  # Get conversation
cw pub conversations resolve <inbox-id> <contact-id> 123  # Resolve conversation
cw pub messages ls <inbox-id> <contact-id> 123  # List messages
cw pub messages mk <inbox-id> <contact-id> 123 --content "Hello!"  # Send message
cw pub messages up <inbox-id> <contact-id> 123 456 --content "Updated"  # Update message
```

## Output Formats

### Text

Human-readable tables:

```bash
$ cw c ls
ID      STATUS    INBOX         CONTACT           MESSAGES    UPDATED
123     open      Support       John Doe          5           2 hours ago
124     pending   Sales         Jane Smith        3           1 day ago
```

### JSON

Machine-readable output:

```bash
$ cw c ls -o json
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
cw c context 123 -o agent | jq '.item.summary'  # Extract summary from context
cw c context 123 -o agent --rn | jq '.item.contact_labels'  # Extract labels with resolved names
```

**List commands return an object with an "items" array** (plus `has_more` and `meta`):
```bash
cw co ls -o agent | jq '.items[0]'       # Get first contact
cw c ls -o agent | jq '.items[] | select(.status == "open")'  # Filter open conversations
```

**Get commands return single objects**:
```bash
cw co g 123 -o agent | jq '.item.email'  # Extract contact email
cw c g 456 -o agent | jq '.item.messages_count'  # Extract message count
```

Data goes to stdout, errors and progress to stderr for clean piping.

### JSONL

Streaming-friendly output (one JSON object per line):

```bash
$ cw c ls -o jsonl
{"id":123,"status":"open",...}
{"id":124,"status":"pending",...}
```

Tip: `--query` and `--template` apply per line in JSONL mode.

## Examples

### Triage open conversations

```bash
cw c ls --st open -o json | \
  jq '.items[] | {id, contact: .contact.name, messages: .messages_count}'  # Extract triage summary
cw c ls --st open --li                   # Light: minimal conversation payload for triage
cw c assign 123 --ag 5                   # Assign to agent
cw c toggle-priority 123 --pri high      # Set high priority
```

### Send bulk campaign

```bash
cw cm cr \
  --title "Product Update" \
  -m "Check out our new feature!" \
  --iid 1 \
  --labels 10,11,12                            # Create campaign for specific labels
```

### Build AI support bot

```bash
cw c context 123 --embed -o json > context.json  # Get context with images for AI vision
cat context.json | your-ai-tool --prompt "Draft a helpful response"  # Process with AI
cw m cr 123 -c "Your drafted response here"  # Send the AI response
```

### Search contacts by email domain

```bash
cw co filter \
  --payload '[{"attribute_key":"email","filter_operator":"contains","values":["@example.com"]}]' \
  -o json | jq '.items[].email'                # Extract matching emails
```

### Export conversations for analysis

```bash
cw c filter \
  --payload '[{"attribute_key":"status","filter_operator":"equal_to","values":["resolved"]}]' \
  -o json > resolved-conversations.json        # Export resolved conversations
jq '[.items[] | {id, resolved_at: .updated_at, messages: .messages_count, agent: .assignee.name}]' \
  resolved-conversations.json                  # Extract key metrics
```

### Manage help center content

```bash
cw po cr -n "Knowledge Base" --slug kb   # Create portal
cw po categories cr kb -n "Getting Started" --slug getting-started  # Create category
cw po articles cr kb \
  --title "How to Get Started" \
  --content "..." \
  --category-id 1 \
  --status 1                                   # Publish article
```

### AI Context

Get complete conversation data optimized for use by LLMs:

```bash
cw c context 123 --embed                 # Text format with embedded images
cw c context 123 --embed -o json         # JSON format for programmatic access
```

The `--embed` flag converts images to base64 data URIs that AI vision models can process directly.

### Pagination

List commands support pagination:

```bash
cw c ls -a                               # Get all results (automatic pagination)
cw c ls -a --mp 50                       # Limit pagination depth to 50 pages
```

### Filtering and Search

**Filter** uses Chatwoot's filter API for structured queries:
```bash
cw c filter --payload '[{"attribute_key":"status","filter_operator":"equal_to","values":["open"]}]'  # Filter conversations
cw c filter --payload '[{"attribute_key":"status","filter_operator":"equal_to","values":["open"]}]' --li  # Light filter results
cw co filter --payload '[{"attribute_key":"email","filter_operator":"contains","values":["@vip.com"]}]'  # Filter contacts
```

**Search** performs full-text search:
```bash
cw c search --query "refund request"     # Search conversations
cw c search "refund request" --li        # Light conversation search payload
cw co search --query "john smith"        # Search contacts
cw s "john smith" --li                   # Light global search payload
```

## Common Workflows

### Inbox -> Conversation -> Message

```bash
cw in ls                                 # Pick an inbox
cw c ls --st open --iid 1                # List open conversations in inbox
cw m cr 123 -c "Hello! How can I help?"  # Send a message
```

### Contacts -> Conversations

```bash
cw co search --query "john@example.com"  # Find a contact
cw co conversations 123                  # List conversations for contact
cw co conversations 123 -o agent --rn    # Agent output with resolved names
```

## Troubleshooting

- **401 Unauthorized**: run `cw auth login` and verify your token.
- **403 Forbidden**: check your account role and permissions.
- **404 Not Found**: verify the resource ID (it may have been deleted).
- **URL validation failed**: ensure the base URL is public, or use `--allow-private` only if you trust the target.
- **Base URL not configured**: set `CHATWOOT_BASE_URL` / `CHATWOOT_API_TOKEN` / `CHATWOOT_ACCOUNT_ID` or run `cw auth login`.

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
- `-q <expr>` / `--query <expr>` / `--jq <expr>` - JQ expression to filter JSON output (supports key aliases in path contexts)
- `--fields <a,b,c>` - Select fields in JSON output (shorthand for `--query`; supports presets like `minimal`, `default`, `debug` on supported resources, and key aliases in paths)
- `-Q` / `--quiet` - Suppress non-essential output
- `--silent` - Suppress non-error output to stderr
- `--no-input` - Disable interactive prompts
- `-y` / `--yes` - Assume yes for confirmations (desire path alias for `--force`)
- `--template <tmpl>` - Go template (or `@path`) to render JSON output
- `--utc` - Display timestamps in UTC
- `--tz <tz>` / `--time-zone <tz>` - Display timestamps in a specific time zone (e.g., `America/Los_Angeles`)
- `--max-rl <n>` / `--max-rate-limit-retries <n>` - Max retries for HTTP 429 responses
- `--max-5xx-retries <n>` - Max retries for HTTP 5xx responses
- `--rld <duration>` / `--rate-limit-delay <duration>` - Base delay for 429 retries (e.g., 1s)
- `--sedly <duration>` / `--server-error-delay <duration>` - Delay between 5xx retries (e.g., 1s)
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
| `--json` | `--j` (also `-j`) |
| `--idempotency-key` | `--idem` |
| `--max-rate-limit-retries` | `--max-rl` |
| `--rate-limit-delay` | `--rld` |
| `--server-error-delay` | `--sedly` |
| `--output` | `--out` |
| `--query` | `--qr` |
| `--query-file` | `--qf` |
| `--items-only` | `--io`, `--results-only`, `--ro` |
| `--compact-json` | `--cj` |
| `--color` | `--clr` |
| `--debug` | `--dbg` |
| `--fields` | `--fi` |
| `--silent` | `--sil` |
| `--no-input` | `--ni` |
| `--template` | `--tpl` |
| `--timeout` | `--to` |
| `--wait` | `--wai` |
| `--utc` | `--ut` |
| `--max-5xx-retries` | `--m5x` |
| `--allow-private` | `--ap` |
| `--circuit-breaker-threshold` | `--cbt` |
| `--circuit-breaker-reset-time` | `--cbr` |

## Command Aliases

Most commands provide short aliases for fast typing. Use `cw <alias>` instead of the full command name:

| Command | Aliases |
|---------|---------|
| `account` | `acc`, `ac` |
| `agent-bots` | `bots`, `ab` |
| `agents` | `agent`, `a` |
| `api` | `ap` |
| `assign` | `reassign`, `as` |
| `audit-logs` | `audit`, `al` |
| `auth` | `au` |
| `automation-rules` | `automation`, `rules`, `ar` |
| `cache` | `ch` |
| `campaigns` | `campaign`, `camp`, `cm` |
| `canned-responses` | `cr`, `canned` |
| `client` | `cl` |
| `close` | `close-conversation`, `resolve-conversation`, `resolve`, `x` |
| `comment` | `cmt` |
| `completions` | `-` |
| `config` | `cfg` |
| `contacts` | `contact`, `customers`, `co` |
| `conversations` | `conv`, `c` |
| `csat` | `satisfaction`, `cs` |
| `ctx` | `context`, `ct` |
| `custom-attributes` | `attrs`, `ca` |
| `custom-filters` | `filters`, `cf` |
| `dashboard` | `dash`, `dh` |
| `handoff` | `escalate`, `transfer`, `ho` |
| `inbox-members` | `inbox_members`, `im` |
| `inboxes` | `inbox`, `in` |
| `integrations` | `integration`, `int`, `ig` |
| `labels` | `label`, `l` |
| `mentions` | `mn` |
| `messages` | `message`, `msg`, `m` |
| `note` | `internal-note`, `n` |
| `open` | `get`, `show`, `o` |
| `platform` | `pf` |
| `portals` | `portal`, `po` |
| `profile` | `pr` |
| `public` | `pub` |
| `ref` | `-` |
| `reopen` | `open-conversation`, `ro` |
| `reply` | `respond`, `r` |
| `reports` | `report`, `rpt`, `rp` |
| `schema` | `sc` |
| `search` | `find`, `s` |
| `snooze` | `pause`, `defer`, `sn` |
| `status` | `st` |
| `survey` | `sv` |
| `teams` | `team`, `t` |
| `version` | `v` |
| `webhooks` | `webhook`, `wh` |

Subcommands also have aliases (e.g., `list` -> `ls`, `get` -> `g`, `create` -> `mk`/`cr`, `update` -> `up`, `delete` -> `rm`/`del`).

### Examples

```bash
# These are equivalent:
cw conversations list --status open
cw c ls --st open

cw messages list 123 --limit 50
cw m ls 123 -l 50

cw search "refund" --type conversations --limit 5
cw s "refund" -t conversations -l 5

cw contacts create --name "John" --email "john@example.com"
cw co cr -n "John" -e "john@example.com"

cw close 123
cw x 123
```

## Flag Aliases

Commonly used flags have short aliases to reduce typing. Single-letter aliases appear in `--help` output. Multi-letter aliases (like `--st`, `--iid`) are hidden from help but work the same way.

### Single-Letter Flag Aliases

| Flag | Alias | Available on |
|------|-------|-------------|
| `--content` | `-c` | messages create/update, comment, note, reply |
| `--compact` | `-c` | dashboard (compact field selection, distinct from `--compact-json`) |
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
| `--contact` | `--ct` | dashboard |
| `--conversation` | `--cv` | dashboard |
| `--team-id` | `--tid` | conversations list/create |
| `--priority` | `--pri` | conversations, comment, note, reply |
| `--page` | `--pg` | dashboard |
| `--per-page` | `--pp` | dashboard |
| `--compact` | `--brief`, `--summary` | dashboard |
| `--agent` | `--ag` | assign, conversations, handoff |
| `--description` | `--desc` | campaigns, labels, platform, portals, teams |
| `--assignee-type` | `--at` | conversations list |
| `--unread-only` | `--unread` | conversations list |
| `--waiting` | `--wt` | conversations list |
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
cw c ls -o json --query '.items[].id'    # Get only conversation IDs
cw c ls -o json --query '.items[] | select(.status == "open")'  # Filter by status
```

### JSON Key Aliases (Query/Path Contexts)

To reduce typing, `--query`/`--jq`, `--fields`, and path-style `--sort` values accept lowercase key aliases.
Aliases rewrite only path tokens (for example `.it[].st`), and do **not** rewrite:

- quoted bracket literals like `.["st"]`
- mixed-case tokens like `.St`
- strings/comments inside jq expressions

Long-form keys remain fully supported.

Supported jq function aliases (only when called with parentheses):

| Alias | Canonical function |
|------|---------------------|
| `sl` | `select` |
| `ts` | `test` |

Inventory basis: aliases were selected from a scan of docs/tests plus JSON tags and response maps, ranked by frequency and character savings.

Top key candidates by savings score (`frequency * (len-2)`):

| Key | Score |
|-----|------:|
| `account_id` | 400 |
| `description` | 270 |
| `last_activity_at` | 252 |
| `created_at` | 248 |
| `items` | 231 |
| `conversation_id` | 195 |
| `name` | 186 |
| `custom_attributes` | 165 |
| `open_conversations` | 160 |
| `payload` | 155 |

| Alias | Canonical key |
|------|----------------|
| `aci` | `account_id` |
| `act` | `actions` |
| `ai` | `assignee_id` |
| `att` | `attachments` |
| `blk` | `blacklist` |
| `ca` | `created_at` |
| `ci` | `contact_id` |
| `ct` | `content` |
| `ctc` | `contact` |
| `ctp` | `content_type` |
| `cu` | `custom_attributes` |
| `cv` | `conversation_id` |
| `cvn` | `conversation` |
| `di` | `display_id` |
| `ds` | `description` |
| `dt` | `data` |
| `du` | `data_url` |
| `e` | `email` |
| `en` | `enabled` |
| `er` | `error` |
| `fc` | `first_contact` |
| `fs` | `file_size` |
| `ft` | `file_type` |
| `hm` | `has_more` |
| `i` | `id` |
| `ii` | `inbox_id` |
| `im` | `item` |
| `it` | `items` |
| `kd` | `kind` |
| `la` | `last_activity_at` |
| `lac` | `last_activity` |
| `lb` | `labels` |
| `mc` | `messages_count` |
| `mgs` | `messages` |
| `mi` | `message_id` |
| `msg` | `message` |
| `mt` | `meta` |
| `mtr` | `membership_tier` |
| `mty` | `message_type` |
| `n` | `name` |
| `oc` | `open_conversations` |
| `pl` | `payload` |
| `pn` | `phone_number` |
| `pr` | `priority` |
| `ps` | `position` |
| `pv` | `private` |
| `qe` | `query` |
| `rs` | `results` |
| `sd` | `sender` |
| `sdi` | `sender_id` |
| `sg` | `slug` |
| `sm` | `summary` |
| `snm` | `sender_name` |
| `st` | `status` |
| `sty` | `sender_type` |
| `tcv` | `total_conversations` |
| `ti` | `team_id` |
| `tl` | `title` |
| `tm` | `total_messages` |
| `tu` | `thumb_url` |
| `ty` | `type` |
| `ua` | `updated_at` |
| `uc` | `unread_count` |
| `ur` | `url` |

For agent workflows, prefer these aliases in query/path contexts to minimize command length.

Before/after examples:

```bash
# --query / --jq
cw c ls -o json --query '.items[] | select(.status == "open") | .id'
cw c ls -o json --query '.it[] | select(.st == "open") | .i'

# --fields projection
cw c ls -o json --fields id,status,last_activity_at,custom_attributes.plan
cw c ls -o json --fields i,st,la,cu.plan

# --sort path
cw co ls --sort last_activity_at
cw co ls --sort la
```

Message-focused shortest forms:

```bash
# Last N messages with key fields
cw m ls CONV_ID --jq '.it[-6:] | .[] | {id: .i, content: .ct, sender: .sd.n}'

# Only incoming messages
cw m ls CONV_ID --jq '[.it[] | sl(.mty == 0)]'

# Only outgoing messages
cw m ls CONV_ID --jq '[.it[] | sl(.mty == 1)]'

# Filter by content
cw m ls CONV_ID --jq '[.it[] | sl(.ct != null) | sl(.ct | ts("keyword"; "i"))]'
```

Literal-key preservation example:

```bash
cw schema list -o json --query '.["it"]'   # looks up literal key "it" (not alias-rewritten)
```

**Fields shorthand & templates**

```bash
cw c ls -o json --fields id,status,assignee_id  # Select fields without writing JQ
cw c g 123 -o json --template '{{.id}} {{.status}}'  # Render custom output
```

### Field Presets

Many commands support field presets for common use cases. Instead of listing individual fields, use a preset name:

```bash
cw co ls -o json --fields minimal        # Bare essentials (id, name, email)
cw c ls -o json --fields default         # Common fields for typical workflows
cw co ls -o json --fields debug          # All fields for troubleshooting
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
cw completion bash > /usr/local/etc/bash_completion.d/cw
# Or for Linux:
cw completion bash > /etc/bash_completion.d/cw
```

### Zsh

```zsh
cw completion zsh > "${fpath[1]}/_cw"
# Or add to .zshrc:
echo 'eval "$(cw completion zsh)"' >> ~/.zshrc
```

### Fish

```fish
cw completion fish > ~/.config/fish/completions/cw.fish
```

### PowerShell

```powershell
cw completion powershell | Out-String | Invoke-Expression
# Or add to profile:
cw completion powershell >> $PROFILE
```

## Version & Updates

```bash
cw v                                     # Show CLI version
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
