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
# Binary at ./bin/chatwoot
```

## Quick Start

### 1. Authenticate

**Browser:**
```bash
chatwoot auth login                            # Opens browser for interactive login
```

**Terminal:**
```bash
chatwoot auth login --browser=false --url https://chatwoot.example.com --token YOUR_API_TOKEN --account-id 1
```

### 2. Verify Setup

```bash
chatwoot pr g                                  # Get current user profile
```

### 3. List Conversations

```bash
chatwoot c ls --st open                        # List open conversations
```

### 4. Search (Agent-Friendly)

```bash
chatwoot s "john"                              # Search across contacts + conversations
chatwoot s "refund" --best                     # Auto-pick the best result (no interactive prompt)
chatwoot s "refund" --best --emit id           # Emit just an ID for chaining (no jq)
```

## Configuration

### Authentication

Credentials are checked in this order:
1. Environment variables (`CHATWOOT_BASE_URL`, `CHATWOOT_API_TOKEN`, `CHATWOOT_ACCOUNT_ID`)
2. `CHATWOOT_PROFILE` (if set)
3. Current profile in keychain (defaults to `default`)

Check current configuration:
```bash
chatwoot auth status                           # Show current config
chatwoot auth status --json                    # Show current config as JSON
```

Remove stored credentials:
```bash
chatwoot auth logout                           # Remove stored credentials from keychain
```

### Profiles

Manage multiple stored accounts:
```bash
chatwoot cfg profiles ls                       # List all profiles
chatwoot cfg profiles use staging              # Switch to staging profile
chatwoot cfg profiles show --name staging      # Show profile details
chatwoot cfg profiles del staging              # Delete a profile
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
chatwoot auth login                            # Authenticate via browser
chatwoot auth login --no-browser --url <url> --token <t> --account-id <id>  # CLI login
chatwoot auth status                           # Show current config
chatwoot auth logout                           # Remove credentials
```

## Extensions

Executables on your PATH named `chatwoot-<name>` can be invoked as:

```bash
chatwoot <name> [args...]
```

### Shortcuts (Agent-Friendly)

Convenience commands designed for agent workflows:

```bash
chatwoot o 123                                 # Open conversation details
chatwoot show https://app.chatwoot.com/app/accounts/1/conversations/123  # Open from URL
chatwoot cmt 123 "Hello! How can I help?"      # Send a public reply
chatwoot n 123 "Internal note" --mention lily   # Add private note with @mention
chatwoot r 123 "Thanks for contacting us!"     # Reply to conversation
chatwoot x 123 456                             # Close (resolve) conversations
chatwoot ro 123                                # Reopen a closed conversation
chatwoot sn 123 --for 2h                       # Snooze for 2 hours
chatwoot ho 123 --ag lily --reason "Needs billing"  # Escalate with reason
chatwoot as 123 --ag 5 --team 2                # Assign to agent and team
chatwoot ct 123 -o agent                       # Get AI context for conversation
chatwoot ref 123                               # Resolve ID to typed reference
```

### Conversations

```bash
chatwoot c ls                                  # List all conversations
chatwoot c ls --st open --iid 1                # Filter by status and inbox
chatwoot c ls --st open --at unassigned        # Filter by assignee type
chatwoot c ls --st open --tid 2 -L "vip,urgent"  # Filter by team and labels
chatwoot c ls --st open --search "refund"      # Full-text search within list
chatwoot c ls -a --mp 50                       # Paginate all, max 50 pages
chatwoot c filter --payload '[{"attribute_key":"status","filter_operator":"equal_to","values":["open"]}]'  # Structured filter
chatwoot c search --query "refund"             # Full-text search
chatwoot c g 123                               # Get conversation details
chatwoot c counts                              # Get counts by status
chatwoot c meta                                # Get account metadata
chatwoot c attachments 123                     # List attachments for conversation
chatwoot c toggle-status 123 --st resolved     # Change conversation status
chatwoot c toggle-priority 123 --pri high      # Set conversation priority
chatwoot c assign 123 --ag 5 --team 2          # Assign to agent and team
chatwoot c mark-unread 123                     # Mark conversation as unread
chatwoot c labels 123                          # List labels on conversation
chatwoot c labels-add 123 --labels "urgent,vip"  # Add labels to conversation
chatwoot c custom-attributes 123 --payload '{"key":"value"}'  # Set custom attributes
chatwoot c context 123 --embed                 # Get full AI context with embedded images
chatwoot c context 123 -o agent                # Agent-friendly context envelope
chatwoot c context 123 -o agent --rn           # Context with resolved names
chatwoot c transcript 123                      # Render transcript locally
chatwoot c transcript 123 --public-only        # Transcript excluding private notes
chatwoot c transcript 123 -l 200               # Transcript limited to 200 messages
chatwoot c transcript 123 --email user@example.com  # Email transcript to address
chatwoot c follow 123                          # Follow conversation via WebSocket
chatwoot c follow 123 --tail 50               # Show last 50 messages then stream
chatwoot c follow --all                        # Follow all account conversations
chatwoot c follow --all --events all           # All event types (status, assignments, etc.)
chatwoot c follow 123 --typing                 # Include typing indicators
chatwoot c follow 123 -o agent --rn            # Agent output with resolved names
chatwoot c follow --all --inbox 1 --status open  # Filter by inbox and status
chatwoot c follow --all --label vip --pri urgent  # Filter by label and priority
chatwoot c follow --all --assignee 5           # Only agent 5's conversations
chatwoot c follow --all --unassigned           # Only unassigned conversations
chatwoot c follow --all --pub                  # Exclude private messages
chatwoot c follow 123 --debounce 2s            # Batch rapid messages (2s window)
chatwoot c follow 123 --context --cm 20        # Emit snapshot with 20 context messages
chatwoot c follow 123 --cursor-file .cursor    # Resume from last position on restart
chatwoot c follow 123 --since-id 456           # Skip messages with id <= 456
chatwoot c follow 123 --since-time 24h         # Skip messages older than 24h
chatwoot c follow 123 --raw                    # Include raw WebSocket payload
chatwoot c follow 123 --exec './handler.sh'    # Pipe each event JSON to command
chatwoot c follow 123 --exec './handler.sh' --exec-fatal  # Abort on handler failure
```

> **Real-time streaming:** `follow` connects directly to Chatwoot's ActionCable WebSocket.
> No webhook setup required. Reconnects automatically with exponential backoff.
> In agent mode (`-o agent`), conversation snapshots are emitted automatically on first event.

### Messages

```bash
chatwoot m ls 123                              # List messages in conversation
chatwoot m ls 123 -a                           # List all messages (paginated)
chatwoot m ls 123 -l 500                       # List up to 500 messages
chatwoot m ls 123 -o agent --rn                # Agent output with resolved names
chatwoot m cr 123 -c "Hello!"                  # Send a message
chatwoot m up 123 456 -c "Updated text"        # Update message 456
chatwoot m del 123 456                         # Delete message 456
```

> **Note:** Messages are returned in chronological order (oldest first, most recent at end of array).
> To get the last N messages: `chatwoot m ls 123 --json | jq '.items[-N:]'`

### Private Notes & Mentions

Private notes are internal messages visible only to agents, not customers. You can mention/tag agents to notify them.

```bash
chatwoot m cr 123 -P -c "Internal note for the team"  # Create private note
chatwoot m cr 123 -P --mention lily -c "Can you follow up on this?"  # Mention an agent
chatwoot m cr 123 -P --mention lily --mention jack -c "Please review together"  # Mention multiple agents
chatwoot m cr 123 -P --mention lily@example.com -c "Check this out"  # Mention by email
```

The `--mention` flag:
- Accepts agent name (partial match) or email
- Automatically resolves to the agent's ID
- Formats the mention correctly so the agent receives a notification
- Requires the `-P` flag (mentions only work in private notes)

### Contacts

```bash
chatwoot co ls                                 # List all contacts
chatwoot co ls --sort name --order asc         # Sort by name ascending
chatwoot co ls --sort -last_activity_at        # Sort by recent activity
chatwoot co search --query "john"              # Full-text search contacts
chatwoot co filter --payload '[{"attribute_key":"email","filter_operator":"contains","values":["@example.com"]}]'  # Structured filter
chatwoot co g 123                              # Get contact by ID
chatwoot co show 123                           # Get contact (alias for get)
chatwoot co g +16042091231                     # Lookup contact by phone number
chatwoot co cr -n "John Doe" -e "john@example.com"  # Create a contact
chatwoot co up 123 --phone "+1234567890"       # Update phone number
chatwoot co up john@example.com -n "John Smith"  # Update by email lookup
chatwoot co up +16042091231 -n "Wenqi Qu" -e "quwenqi@example.com"  # Update by phone lookup
chatwoot co del 123                            # Delete a contact
chatwoot co merge 123 456                      # Merge 456 into 123 (456 deleted)
chatwoot co merge 123 456 -y                   # Merge without confirmation
chatwoot co conversations 123                  # List conversations for contact
chatwoot co contactable-inboxes 123            # List inboxes contact can reach
chatwoot co create-inbox 123 --iid 1 --source-id "+15551234567"  # Associate contact with inbox
chatwoot co labels 123                         # List labels on contact
chatwoot co labels-add 123 --labels "customer,premium"  # Add labels to contact
chatwoot co bulk add-label --ids 1,2,3 --labels "vip,priority"  # Bulk add labels
chatwoot co bulk remove-label --ids 1,2,3 --labels "old-tag"  # Bulk remove labels
printf "1\n2\n3\n" | chatwoot co bulk add-label --ids @- --labels vip  # Bulk IDs from stdin
chatwoot co notes 123                          # List notes on contact
chatwoot co notes-add 123 --content "Called about refund"  # Add a note
chatwoot co notes-delete 123 456               # Delete note 456
```

### Campaigns

```bash
chatwoot cm ls                                 # List all campaigns
chatwoot cm ls -a                              # List all (paginated)
chatwoot cm g 123                              # Get campaign details
chatwoot cm cr --title "Welcome" -m "Hello!" --iid 1 --labels 5,6  # Create campaign
chatwoot cm up 123 --enabled=false             # Disable campaign
chatwoot cm del 123 -y                         # Delete without confirmation
```

### Help Center (Portals)

```bash
chatwoot po ls                                 # List portals
chatwoot po g help                             # Get portal by slug
chatwoot po cr -n "Help Center" --slug help    # Create portal
chatwoot po up help -n "Support Center"        # Update portal name
chatwoot po del help                           # Delete portal
chatwoot po articles ls help                   # List articles in portal
chatwoot po articles cr help --title "Getting Started" --content "..." --category-id 1  # Create article
chatwoot po articles up help 123 --status 1    # Update article (0=draft, 1=published, 2=archived)
chatwoot po articles del help 123              # Delete article
chatwoot po categories ls help                 # List categories in portal
chatwoot po categories cr help -n "FAQ" --slug faq  # Create category
chatwoot po categories up help faq -n "Frequently Asked Questions"  # Update category
chatwoot po categories del help faq            # Delete category
```

### Inboxes

```bash
chatwoot in ls                                 # List all inboxes
chatwoot in g 1                                # Get inbox details
chatwoot in cr -n "Support" --channel-type api --greeting-enabled --greeting-message "Hi!"  # Create inbox
chatwoot in up 1 --timezone "America/New_York" --working-hours-enabled  # Update inbox settings
```

### Inbox Members

```bash
chatwoot inbox-members ls --iid 1              # List members of inbox
chatwoot inbox-members add --iid 1 --user-id 5  # Add agent to inbox
chatwoot inbox-members up --iid 1 --user-id 5 --role administrator  # Update member role
chatwoot inbox-members remove --iid 1 --user-id 5  # Remove agent from inbox
```

### Agents & Teams

```bash
chatwoot a ls                                  # List all agents
chatwoot a g 5                                 # Get agent details
chatwoot t ls                                  # List all teams
chatwoot t g 1                                 # Get team details
chatwoot t members 1                           # List team members
```

### Canned Responses

```bash
chatwoot canned-responses ls                   # List all canned responses
chatwoot canned-responses g 123                # Get canned response
chatwoot canned-responses cr --short-code "greeting" --content "Hello! How can I help?"  # Create template
chatwoot canned-responses up 123 --content "Updated response"  # Update template
chatwoot canned-responses del 123              # Delete template
```

### Webhooks

```bash
chatwoot wh ls                                 # List all webhooks
chatwoot wh g 123                              # Get webhook details
chatwoot wh cr --url "https://example.com/webhook" --subscriptions "message_created,conversation_created"  # Create webhook
chatwoot wh up 123 --url "https://new.example.com/hook"  # Update webhook URL
chatwoot wh del 123                            # Delete webhook
```

### Automation Rules

```bash
chatwoot automation-rules ls                   # List all automation rules
chatwoot automation-rules g 123                # Get automation rule details
```

### Agent Bots

```bash
chatwoot agent-bots ls                         # List all bots
chatwoot agent-bots g 123                      # Get bot details
chatwoot agent-bots cr -n "Support Bot" --desc "Handles FAQs"  # Create bot
chatwoot agent-bots up 123 -n "FAQ Bot"        # Update bot name
chatwoot agent-bots del 123                    # Delete bot
```

### Custom Attributes

```bash
chatwoot custom-attributes ls --attribute-model contact_attribute  # List contact attributes
chatwoot custom-attributes g 123 --attribute-model contact_attribute  # Get attribute
chatwoot custom-attributes cr --attribute-model contact_attribute --attribute-key "vip_status" --attribute-display-name "VIP Status" --attribute-display-type list --attribute-values '["gold","silver","bronze"]'  # Create attribute
chatwoot custom-attributes up 123 --attribute-model contact_attribute --attribute-display-name "VIP Level"  # Update attribute
chatwoot custom-attributes del 123 --attribute-model contact_attribute  # Delete attribute
```

### Custom Filters

```bash
chatwoot custom-filters ls --filter-type conversation  # List conversation filters
chatwoot custom-filters g 123 --filter-type conversation  # Get filter details
chatwoot custom-filters cr --filter-type conversation -n "High Priority Open" --query '{"payload":[{"attribute_key":"status","filter_operator":"equal_to","values":["open"]},{"attribute_key":"priority","filter_operator":"equal_to","values":["high"]}]}'  # Create filter
chatwoot custom-filters up 123 --filter-type conversation -n "Updated Filter"  # Update filter
chatwoot custom-filters del 123 --filter-type conversation  # Delete filter
```

### Labels

```bash
chatwoot l ls                                  # List all labels
chatwoot l g 123                               # Get label details
chatwoot l cr --title "VIP"                    # Create label
chatwoot l up 123 --title "Premium Customer"   # Update label title
chatwoot l del 123                             # Delete label
```

### Integrations

```bash
chatwoot integrations ls                       # List all integrations
chatwoot integrations hooks ls --app-id slack   # List hooks for an app
chatwoot integrations hooks cr --app-id slack --iid 1 --settings '{"webhook_url":"https://hooks.slack.com/..."}'  # Create hook
chatwoot integrations hooks up 123 --app-id slack --settings '{"webhook_url":"https://new.hooks.slack.com/..."}'  # Update hook
chatwoot integrations hooks del 123 --app-id slack  # Delete hook
```

### Platform APIs

```bash
chatwoot pf accounts cr -n "Acme Inc" --domain acme.example.com --support-email support@acme.example.com  # Create account
chatwoot pf accounts g 123                     # Get account details
chatwoot pf accounts del 123                   # Delete account
chatwoot pf users cr -n "Jane Doe" --email jane@example.com --password "secret"  # Create user
chatwoot pf users up 45 --display-name "Jane D"  # Update user display name
chatwoot pf users del 45                       # Delete user
chatwoot pf account-users ls 123               # List users in account
chatwoot pf account-users cr 123 --user-id 45 --role agent  # Add user to account
chatwoot pf account-users del 123 --user-id 45  # Remove user from account
```

### Public Client APIs

```bash
chatwoot client contacts cr --inbox <inbox-identifier> -n "Visitor" -e visitor@example.com  # Create contact via inbox
chatwoot client contacts g --inbox <inbox-identifier> --contact <contact-identifier>  # Get contact
chatwoot client conversations ls --inbox <inbox-identifier> --contact <contact-identifier>  # List conversations
chatwoot client conversations cr --inbox <inbox-identifier> --contact <contact-identifier>  # Create conversation
chatwoot client conversations g 123 --inbox <inbox-identifier> --contact <contact-identifier>  # Get conversation
chatwoot client conversations resolve 123 --inbox <inbox-identifier> --contact <contact-identifier>  # Resolve conversation
chatwoot client messages cr 123 --inbox <inbox-identifier> --contact <contact-identifier> -c "Hello!"  # Send message
chatwoot client typing 123 --inbox <inbox-identifier> --contact <contact-identifier> --status on  # Send typing indicator
chatwoot client last-seen up 123 --inbox <inbox-identifier> --contact <contact-identifier>  # Update last seen
```

### Reports

```bash
chatwoot rp summary --since 2024-01-01 --until 2024-12-31  # Get summary report
chatwoot rp summary --metric conversations_count --type account  # Specific metric
```

### CSAT (Customer Satisfaction)

```bash
chatwoot cs ls                                 # List CSAT responses
chatwoot cs metrics --since 2024-01-01 --until 2024-12-31  # Get CSAT metrics
```

### Audit Logs

```bash
chatwoot audit-logs ls                         # List all audit logs
chatwoot audit-logs ls --user-id 5             # Filter by user
```

### Profile & Account

```bash
chatwoot pr g                                  # Get your user profile
chatwoot account g                             # Get account details
```

### Status

```bash
chatwoot st                                    # Show config and auth status
chatwoot st -o json                            # Status as JSON
chatwoot st --check                            # Exit code 1 if not authenticated
```

### Raw API Requests

Make direct API calls to any Chatwoot endpoint (like `gh api` for GitHub):

```bash
chatwoot ap /conversations/123                 # GET request (default)
chatwoot ap /conversations -X POST -f inbox_id=1 -f contact_id=5  # POST with fields
chatwoot ap /conversations/123 -X PATCH -F 'labels=["bug","urgent"]'  # PATCH with JSON array
chatwoot ap /automation_rules/14 -X PATCH -d '{"automation_rule":{"active":true}}'  # Inline JSON body
chatwoot ap /contacts -X POST -i body.json     # Body from file
echo '{"name":"Test"}' | chatwoot ap /contacts -X POST -i -  # Body from stdin
chatwoot ap /contacts -o json --jq '.payload[0].name'  # Filter response with jq
chatwoot ap /conversations/123 -X DELETE -s    # Silent mode (no output)
chatwoot ap /conversations/123 --include       # Show response headers
```

### Snooze

```bash
chatwoot sn 123 --for 2h                       # Snooze for 2 hours
chatwoot sn 123 --for 24h                      # Snooze for 1 day
chatwoot sn 123 --for 2h --note "Waiting for customer reply"  # Snooze with private note
```

### Handoff (Escalate / Transfer)

Composite command that sends a private note + assigns + sets priority in one step:

```bash
chatwoot ho 123 --ag 5 --reason "Refund request, needs billing approval"  # Handoff to agent
chatwoot ho 123 --team billing --reason "Technical issue beyond L1"  # Handoff to team
chatwoot ho 123 --ag 5 --team billing --priority urgent --reason "VIP customer, SLA at risk"  # Full escalation
```

### Mentions

```bash
chatwoot mn ls                                 # List all recent mentions
chatwoot mn ls -S 24h                          # Mentions from last 24 hours
chatwoot mn ls -S 7d -l 20                     # Last 7 days, limit 20
chatwoot mn ls --conversation-id 123           # Mentions in a specific conversation
```

### Reference Resolver

Normalize IDs, URLs, and typed prefixes into canonical typed IDs for agent workflows:

```bash
chatwoot ref 123                               # Probe conversation + contact
chatwoot ref conversation:123                  # Explicit type (no probe)
chatwoot ref https://app.chatwoot.com/.../123  # Parse from URL
chatwoot ref 123 --type contact                # Force resource type
chatwoot ref 123 --try conversation --try inbox  # Custom probe order
chatwoot ref 123 -E url                        # Emit just the UI URL
chatwoot ref 123 -E id                         # Emit just the typed ID
```

### Dashboard

Query external dashboard APIs for contact data (requires configuration via `chatwoot cfg dashboard add`):

```bash
chatwoot dash orders --contact 180712            # Query orders for contact
chatwoot dash orders --conversation 24445        # Auto-resolve contact from conversation
chatwoot dash orders --contact 180712 --page 2 --per-page 20  # Paginate results
```

### Survey

```bash
chatwoot sv g <conversation-uuid>              # Get survey response for a conversation
```

### Cache

```bash
chatwoot ch clear                              # Clear all cached data
chatwoot ch path                               # Show cache directory and files
```

### Public API (Unauthenticated)

Widget/client-side API using inbox identifiers instead of account auth:

```bash
chatwoot pub inboxes g <inbox-id>              # Get inbox info
chatwoot pub contacts mk <inbox-id> --name "Visitor" --email visitor@example.com  # Create contact
chatwoot pub contacts g <inbox-id> <contact-id>  # Get contact
chatwoot pub contacts up <inbox-id> <contact-id> --name "Updated Name"  # Update contact
chatwoot pub conversations ls <inbox-id> <contact-id>  # List conversations
chatwoot pub conversations mk <inbox-id> <contact-id>  # Create conversation
chatwoot pub conversations g <inbox-id> <contact-id> 123  # Get conversation
chatwoot pub conversations resolve <inbox-id> <contact-id> 123  # Resolve conversation
chatwoot pub messages ls <inbox-id> <contact-id> 123  # List messages
chatwoot pub messages mk <inbox-id> <contact-id> 123 --content "Hello!"  # Send message
chatwoot pub messages up <inbox-id> <contact-id> 123 456 --content "Updated"  # Update message
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
chatwoot c context 123 -o agent | jq '.item.summary'  # Extract summary from context
chatwoot c context 123 -o agent --rn | jq '.item.contact_labels'  # Extract labels with resolved names
```

**List commands return an object with an "items" array** (plus `has_more` and `meta`):
```bash
chatwoot co ls -o agent | jq '.items[0]'       # Get first contact
chatwoot c ls -o agent | jq '.items[] | select(.status == "open")'  # Filter open conversations
```

**Get commands return single objects**:
```bash
chatwoot co g 123 -o agent | jq '.item.email'  # Extract contact email
chatwoot c g 456 -o agent | jq '.item.messages_count'  # Extract message count
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
chatwoot c ls --st open -o json | \
  jq '.items[] | {id, contact: .contact.name, messages: .messages_count}'  # Extract triage summary
chatwoot c assign 123 --ag 5                   # Assign to agent
chatwoot c toggle-priority 123 --pri high      # Set high priority
```

### Send bulk campaign

```bash
chatwoot cm cr \
  --title "Product Update" \
  -m "Check out our new feature!" \
  --iid 1 \
  --labels 10,11,12                            # Create campaign for specific labels
```

### Build AI support bot

```bash
chatwoot c context 123 --embed -o json > context.json  # Get context with images for AI vision
cat context.json | your-ai-tool --prompt "Draft a helpful response"  # Process with AI
chatwoot m cr 123 -c "Your drafted response here"  # Send the AI response
```

### Search contacts by email domain

```bash
chatwoot co filter \
  --payload '[{"attribute_key":"email","filter_operator":"contains","values":["@example.com"]}]' \
  -o json | jq '.items[].email'                # Extract matching emails
```

### Export conversations for analysis

```bash
chatwoot c filter \
  --payload '[{"attribute_key":"status","filter_operator":"equal_to","values":["resolved"]}]' \
  -o json > resolved-conversations.json        # Export resolved conversations
jq '[.items[] | {id, resolved_at: .updated_at, messages: .messages_count, agent: .assignee.name}]' \
  resolved-conversations.json                  # Extract key metrics
```

### Manage help center content

```bash
chatwoot po cr -n "Knowledge Base" --slug kb   # Create portal
chatwoot po categories cr kb -n "Getting Started" --slug getting-started  # Create category
chatwoot po articles cr kb \
  --title "How to Get Started" \
  --content "..." \
  --category-id 1 \
  --status 1                                   # Publish article
```

### AI Context

Get complete conversation data optimized for use by LLMs:

```bash
chatwoot c context 123 --embed                 # Text format with embedded images
chatwoot c context 123 --embed -o json         # JSON format for programmatic access
```

The `--embed` flag converts images to base64 data URIs that AI vision models can process directly.

### Pagination

List commands support pagination:

```bash
chatwoot c ls -a                               # Get all results (automatic pagination)
chatwoot c ls -a --mp 50                       # Limit pagination depth to 50 pages
```

### Filtering and Search

**Filter** uses Chatwoot's filter API for structured queries:
```bash
chatwoot c filter --payload '[{"attribute_key":"status","filter_operator":"equal_to","values":["open"]}]'  # Filter conversations
chatwoot co filter --payload '[{"attribute_key":"email","filter_operator":"contains","values":["@vip.com"]}]'  # Filter contacts
```

**Search** performs full-text search:
```bash
chatwoot c search --query "refund request"     # Search conversations
chatwoot co search --query "john smith"        # Search contacts
```

## Common Workflows

### Inbox -> Conversation -> Message

```bash
chatwoot in ls                                 # Pick an inbox
chatwoot c ls --st open --iid 1                # List open conversations in inbox
chatwoot m cr 123 -c "Hello! How can I help?"  # Send a message
```

### Contacts -> Conversations

```bash
chatwoot co search --query "john@example.com"  # Find a contact
chatwoot co conversations 123                  # List conversations for contact
chatwoot co conversations 123 -o agent --rn    # Agent output with resolved names
```

## Troubleshooting

- **401 Unauthorized**: run `chatwoot auth login` and verify your token.
- **403 Forbidden**: check your account role and permissions.
- **404 Not Found**: verify the resource ID (it may have been deleted).
- **URL validation failed**: ensure the base URL is public, or use `--allow-private` only if you trust the target.
- **Base URL not configured**: set `CHATWOOT_BASE_URL` / `CHATWOOT_API_TOKEN` / `CHATWOOT_ACCOUNT_ID` or run `chatwoot auth login`.

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
| `dashboard` | `dash` |
| `survey` | `sv` |
| `cache` | `ch` |
| `public` | `pub` |
| `api` | `ap` |
| `status` | `st` |
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
chatwoot c ls -o json --query '.items[].id'    # Get only conversation IDs
chatwoot c ls -o json --query '.items[] | select(.status == "open")'  # Filter by status
```

**Fields shorthand & templates**

```bash
chatwoot c ls -o json --fields id,status,assignee_id  # Select fields without writing JQ
chatwoot c g 123 -o json --template '{{.id}} {{.status}}'  # Render custom output
```

### Field Presets

Many commands support field presets for common use cases. Instead of listing individual fields, use a preset name:

```bash
chatwoot co ls -o json --fields minimal        # Bare essentials (id, name, email)
chatwoot c ls -o json --fields default         # Common fields for typical workflows
chatwoot co ls -o json --fields debug          # All fields for troubleshooting
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
chatwoot v                                     # Show CLI version
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
