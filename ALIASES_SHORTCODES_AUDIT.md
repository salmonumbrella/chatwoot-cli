# Alias and Shortcode Audit

Generated: 2026-02-14 22:58:11Z
Repo: `chatwoot-cli`

## Scope
- Command and subcommand aliases (Cobra `Aliases`).
- Flag shortcodes (single-dash shorthands, e.g. `-q`).
- Hidden long-form flag aliases registered via `flagAlias(...)`.

## Summary
- Command contexts scanned: **311**
- Commands with aliases: **230**
- Command alias instances: **274** (unique tokens: **135**)
- Commands with at least one shorthand in scope: **311**
- Shorthand instances in command contexts: **1325** (unique tokens: **25**)

## Collision Checks
- Sibling command token collisions (name+alias in same parent scope): **0**
- Flag shorthand collisions in effective command scope (local + inherited): **0**
- Unit tests run: `go test ./internal/cmd -run 'TestNoAliasCollisions|TestNoFlagShorthandCollisions|TestAliasReuseHasConsistentCommandName|TestPreferredAliasMappings'` => **pass**

## Mixed-Semantics Alias Reuse
No alias tokens are reused with different command names.

## High-Consistency Alias Families
These are reused heavily with consistent meaning:

| Alias | Reuse Count | Command Name |
|---|---:|---|
| `ls` | 30 | `list` |
| `g` | 29 | `get` |
| `mk` | 27 | `create` |
| `up` | 25 | `update` |
| `rm` | 23 | `delete` |
| `q` | 4 | `search` |
| `ab` | 2 | `agent-bots` |
| `add` | 2 | `add-label` |
| `bk` | 2 | `bulk` |
| `ca` | 2 | `custom-attributes` |
| `da` | 2 | `delete-avatar` |
| `f` | 2 | `filter` |
| `la` | 2 | `labels-add` |

## Root Persistent Flag Shortcodes
| Flag | Shorthand |
|---|---|
| `--json` | `-j` |
| `--output` | `-o` |
| `--quiet` | `-q` |
| `--yes` | `-y` |

## Root Hidden Long-Form Flag Aliases
Registered in `internal/cmd/root.go` via `flagAlias(...)`. These are hidden from help and have no short dash form.

| Canonical Flag | Hidden Alias |
|---|---|
| `--resolve-names` | `--rn` |
| `--dry-run` | `--dr` |
| `--help-json` | `--hj` |
| `--time-zone` | `--tz` |
| `--idempotency-key` | `--idem` |
| `--max-rate-limit-retries` | `--max-rl` |
| `--rate-limit-delay` | `--rld` |
| `--server-error-delay` | `--sed` |
| `--json` | `--j` |
| `--output` | `--out` |
| `--query` | `--qr` |
| `--query-file` | `--qf` |
| `--items-only` | `--io` |
| `--results-only` | `--ro` |

## Command Alias Inventory
| Command Path | Aliases |
|---|---|
| `cw account` | `acc`, `ac` |
| `cw account get` | `g` |
| `cw account update` | `up` |
| `cw agent-bots` | `bots`, `ab` |
| `cw agent-bots create` | `mk` |
| `cw agent-bots delete` | `rm` |
| `cw agent-bots delete-avatar` | `da` |
| `cw agent-bots get` | `g` |
| `cw agent-bots list` | `ls` |
| `cw agent-bots reset-token` | `rt` |
| `cw agent-bots update` | `up` |
| `cw agents` | `agent`, `a` |
| `cw agents bulk-create` | `bc` |
| `cw agents create` | `mk` |
| `cw agents delete` | `rm` |
| `cw agents get` | `g` |
| `cw agents list` | `ls` |
| `cw agents update` | `up` |
| `cw api` | `ap` |
| `cw assign` | `reassign`, `as` |
| `cw audit-logs` | `audit`, `al` |
| `cw audit-logs list` | `ls` |
| `cw auth` | `au` |
| `cw automation-rules` | `automation`, `rules`, `ar` |
| `cw automation-rules create` | `mk` |
| `cw automation-rules delete` | `rm` |
| `cw automation-rules get` | `g` |
| `cw automation-rules list` | `ls` |
| `cw automation-rules update` | `up` |
| `cw cache` | `ch` |
| `cw campaigns` | `campaign`, `camp`, `cm` |
| `cw campaigns create` | `mk` |
| `cw campaigns delete` | `rm` |
| `cw campaigns get` | `g` |
| `cw campaigns list` | `ls` |
| `cw campaigns update` | `up` |
| `cw canned-responses` | `cr`, `canned` |
| `cw canned-responses create` | `mk` |
| `cw canned-responses delete` | `rm`, `del` |
| `cw canned-responses get` | `g` |
| `cw canned-responses list` | `ls` |
| `cw canned-responses search` | `q` |
| `cw canned-responses update` | `up` |
| `cw client` | `cl` |
| `cw client contacts create` | `mk` |
| `cw client contacts get` | `g` |
| `cw client conversations create` | `mk` |
| `cw client conversations get` | `g` |
| `cw client conversations list` | `ls` |
| `cw client last-seen` | `lsn` |
| `cw client last-seen update` | `up` |
| `cw client messages create` | `mk` |
| `cw close` | `close-conversation`, `resolve-conversation`, `resolve`, `x` |
| `cw comment` | `cmt` |
| `cw config` | `cfg` |
| `cw config dashboard list` | `ls` |
| `cw config profiles delete` | `rm` |
| `cw config profiles list` | `ls` |
| `cw contacts` | `contact`, `customers`, `co` |
| `cw contacts bulk` | `bk` |
| `cw contacts bulk add-label` | `add` |
| `cw contacts bulk remove-label` | `rl` |
| `cw contacts contactable-inboxes` | `ci` |
| `cw contacts conversations` | `cv` |
| `cw contacts create` | `mk` |
| `cw contacts create-inbox` | `cri` |
| `cw contacts delete` | `rm` |
| `cw contacts filter` | `f` |
| `cw contacts get` | `g` |
| `cw contacts labels-add` | `la` |
| `cw contacts list` | `ls` |
| `cw contacts notes-add` | `na` |
| `cw contacts notes-delete` | `nd` |
| `cw contacts search` | `q` |
| `cw contacts update` | `up` |
| `cw conversations` | `conv`, `c` |
| `cw conversations bulk` | `bk` |
| `cw conversations bulk add-label` | `add` |
| `cw conversations bulk assign` | `asgn` |
| `cw conversations bulk batch-update` | `bu` |
| `cw conversations bulk resolve` | `res` |
| `cw conversations create` | `mk` |
| `cw conversations custom-attributes` | `ca` |
| `cw conversations filter` | `f` |
| `cw conversations follow` | `fw` |
| `cw conversations get` | `g` |
| `cw conversations labels-add` | `la` |
| `cw conversations labels-remove` | `lr` |
| `cw conversations list` | `ls` |
| `cw conversations mark-unread` | `mu` |
| `cw conversations search` | `q` |
| `cw conversations toggle-priority` | `tp` |
| `cw conversations toggle-status` | `ts` |
| `cw conversations transcript` | `tr` |
| `cw conversations triage` | `tri` |
| `cw conversations update` | `up` |
| `cw conversations watch` | `w` |
| `cw csat` | `satisfaction`, `cs` |
| `cw csat get` | `g` |
| `cw csat list` | `ls` |
| `cw ctx` | `context`, `ct` |
| `cw custom-attributes` | `attrs`, `ca` |
| `cw custom-attributes create` | `mk` |
| `cw custom-attributes delete` | `rm` |
| `cw custom-attributes get` | `g` |
| `cw custom-attributes list` | `ls` |
| `cw custom-attributes update` | `up` |
| `cw custom-filters` | `filters`, `cf` |
| `cw custom-filters create` | `mk` |
| `cw custom-filters delete` | `rm` |
| `cw custom-filters get` | `g` |
| `cw custom-filters list` | `ls` |
| `cw custom-filters update` | `up` |
| `cw dashboard` | `dash` |
| `cw handoff` | `escalate`, `transfer`, `ho` |
| `cw inbox-members` | `inbox_members`, `im` |
| `cw inbox-members list` | `ls` |
| `cw inbox-members update` | `up` |
| `cw inboxes` | `inbox`, `in` |
| `cw inboxes agent-bot` | `bot` |
| `cw inboxes create` | `mk` |
| `cw inboxes csat-template` | `cst` |
| `cw inboxes csat-template get` | `g` |
| `cw inboxes delete` | `rm` |
| `cw inboxes delete-avatar` | `da` |
| `cw inboxes get` | `g` |
| `cw inboxes list` | `ls` |
| `cw inboxes set-agent-bot` | `sab` |
| `cw inboxes sync-templates` | `sync` |
| `cw inboxes update` | `up` |
| `cw integrations` | `integration`, `int`, `ig` |
| `cw integrations hook-create` | `hc` |
| `cw integrations hook-delete` | `hd` |
| `cw integrations hook-update` | `hu` |
| `cw integrations notion delete` | `rm` |
| `cw integrations shopify delete` | `rm` |
| `cw labels` | `label`, `l` |
| `cw labels create` | `mk` |
| `cw labels delete` | `rm` |
| `cw labels get` | `g` |
| `cw labels list` | `ls` |
| `cw labels update` | `up` |
| `cw mentions` | `mn` |
| `cw mentions list` | `ls` |
| `cw messages` | `message`, `msg`, `m` |
| `cw messages batch-send` | `bs` |
| `cw messages create` | `mk` |
| `cw messages delete` | `rm` |
| `cw messages list` | `ls` |
| `cw messages update` | `up` |
| `cw note` | `internal-note`, `n` |
| `cw open` | `get`, `show`, `o` |
| `cw platform` | `pf` |
| `cw platform account-users create` | `mk` |
| `cw platform account-users delete` | `rm` |
| `cw platform account-users list` | `ls` |
| `cw platform accounts create` | `mk` |
| `cw platform accounts delete` | `rm` |
| `cw platform accounts get` | `g` |
| `cw platform accounts update` | `up` |
| `cw platform agent-bots` | `ab` |
| `cw platform agent-bots create` | `mk` |
| `cw platform agent-bots delete` | `rm` |
| `cw platform agent-bots get` | `g` |
| `cw platform agent-bots list` | `ls` |
| `cw platform agent-bots update` | `up` |
| `cw platform users create` | `mk` |
| `cw platform users delete` | `rm` |
| `cw platform users get` | `g` |
| `cw platform users update` | `up` |
| `cw portals` | `portal`, `po` |
| `cw portals articles create` | `mk` |
| `cw portals articles delete` | `rm` |
| `cw portals articles get` | `g` |
| `cw portals articles list` | `ls` |
| `cw portals articles search` | `q` |
| `cw portals articles update` | `up` |
| `cw portals categories create` | `mk` |
| `cw portals categories delete` | `rm` |
| `cw portals categories get` | `g` |
| `cw portals categories list` | `ls` |
| `cw portals categories update` | `up` |
| `cw portals create` | `mk` |
| `cw portals delete` | `rm` |
| `cw portals delete-logo` | `dl` |
| `cw portals get` | `g` |
| `cw portals list` | `ls` |
| `cw portals send-instructions` | `si` |
| `cw portals ssl-status` | `ssl` |
| `cw portals update` | `up` |
| `cw profile` | `pr` |
| `cw profile get` | `g` |
| `cw public` | `pub` |
| `cw public contacts create` | `mk` |
| `cw public contacts get` | `g` |
| `cw public contacts update` | `up` |
| `cw public conversations create` | `mk` |
| `cw public conversations get` | `g` |
| `cw public conversations list` | `ls` |
| `cw public inboxes get` | `g` |
| `cw public messages create` | `mk` |
| `cw public messages list` | `ls` |
| `cw public messages update` | `up` |
| `cw reopen` | `open-conversation`, `ro` |
| `cw reply` | `respond`, `r` |
| `cw reports` | `report`, `rpt`, `rp` |
| `cw reports agent-summary` | `agents-summary` |
| `cw reports events list` | `ls` |
| `cw schema` | `sc` |
| `cw schema list` | `ls` |
| `cw search` | `find`, `s` |
| `cw snooze` | `pause`, `defer`, `sn` |
| `cw status` | `st` |
| `cw survey` | `sv` |
| `cw survey get` | `g` |
| `cw teams` | `team`, `t` |
| `cw teams create` | `mk` |
| `cw teams delete` | `rm` |
| `cw teams get` | `g` |
| `cw teams list` | `ls` |
| `cw teams members-add` | `ma` |
| `cw teams members-remove` | `mr` |
| `cw teams update` | `up` |
| `cw version` | `v` |
| `cw webhooks` | `webhook`, `wh` |
| `cw webhooks create` | `mk` |
| `cw webhooks delete` | `rm`, `remove` |
| `cw webhooks get` | `g` |
| `cw webhooks list` | `ls` |
| `cw webhooks update` | `up` |

## Local Flag Shorthand Inventory
Local declarations only (root-inherited shorthand flags are listed once under `cw`).

| Command Path | Flag | Shorthand |
|---|---|---|
| `cw` | `--json` | `-j` |
| `cw` | `--output` | `-o` |
| `cw` | `--quiet` | `-q` |
| `cw` | `--yes` | `-y` |
| `cw agents create` | `--emit` | `-E` |
| `cw agents get` | `--emit` | `-E` |
| `cw agents update` | `--emit` | `-E` |
| `cw api` | `--raw-field` | `-F` |
| `cw api` | `--method` | `-X` |
| `cw api` | `--body` | `-d` |
| `cw api` | `--field` | `-f` |
| `cw api` | `--input` | `-i` |
| `cw campaigns create` | `--emit` | `-E` |
| `cw campaigns create` | `--labels` | `-L` |
| `cw campaigns create` | `--message` | `-m` |
| `cw campaigns get` | `--emit` | `-E` |
| `cw campaigns update` | `--emit` | `-E` |
| `cw campaigns update` | `--labels` | `-L` |
| `cw campaigns update` | `--message` | `-m` |
| `cw comment` | `--resolve` | `-R` |
| `cw comment` | `--content` | `-c` |
| `cw contacts create` | `--emit` | `-E` |
| `cw contacts create` | `--email` | `-e` |
| `cw contacts create` | `--name` | `-n` |
| `cw contacts get` | `--emit` | `-E` |
| `cw contacts list` | `--page` | `-p` |
| `cw contacts show` | `--emit` | `-E` |
| `cw contacts update` | `--emit` | `-E` |
| `cw contacts update` | `--email` | `-e` |
| `cw contacts update` | `--name` | `-n` |
| `cw conversations counts` | `--status` | `-s` |
| `cw conversations create` | `--emit` | `-E` |
| `cw conversations create` | `--message` | `-m` |
| `cw conversations create` | `--status` | `-s` |
| `cw conversations follow` | `--all` | `-A` |
| `cw conversations follow` | `--status` | `-s` |
| `cw conversations get` | `--emit` | `-E` |
| `cw conversations list` | `--labels` | `-L` |
| `cw conversations list` | `--since` | `-S` |
| `cw conversations list` | `--all` | `-a` |
| `cw conversations list` | `--page` | `-p` |
| `cw conversations list` | `--status` | `-s` |
| `cw conversations meta` | `--status` | `-s` |
| `cw conversations search` | `--page` | `-p` |
| `cw conversations toggle-status` | `--status` | `-s` |
| `cw conversations typing` | `--private` | `-P` |
| `cw conversations update` | `--emit` | `-E` |
| `cw conversations watch` | `--status` | `-s` |
| `cw inboxes create` | `--emit` | `-E` |
| `cw inboxes create` | `--name` | `-n` |
| `cw inboxes csat-template set` | `--message` | `-m` |
| `cw inboxes get` | `--emit` | `-E` |
| `cw inboxes list` | `--all` | `-a` |
| `cw inboxes list` | `--limit` | `-l` |
| `cw inboxes list` | `--page` | `-p` |
| `cw inboxes update` | `--emit` | `-E` |
| `cw inboxes update` | `--name` | `-n` |
| `cw mentions list` | `--since` | `-S` |
| `cw messages create` | `--private` | `-P` |
| `cw messages create` | `--content` | `-c` |
| `cw messages list` | `--limit` | `-l` |
| `cw messages update` | `--content` | `-c` |
| `cw note` | `--resolve` | `-R` |
| `cw note` | `--content` | `-c` |
| `cw open` | `--type` | `-T` |
| `cw platform accounts create` | `--name` | `-n` |
| `cw platform accounts update` | `--name` | `-n` |
| `cw platform agent-bots create` | `--name` | `-n` |
| `cw platform agent-bots update` | `--name` | `-n` |
| `cw platform users create` | `--name` | `-n` |
| `cw platform users update` | `--name` | `-n` |
| `cw ref` | `--emit` | `-E` |
| `cw ref` | `--type` | `-T` |
| `cw reply` | `--private` | `-P` |
| `cw reply` | `--resolve` | `-R` |
| `cw reply` | `--content` | `-c` |
| `cw reports events list` | `--since` | `-S` |
| `cw search` | `--emit` | `-E` |
| `cw search` | `--limit` | `-l` |
| `cw search` | `--type` | `-t` |
| `cw teams create` | `--emit` | `-E` |
| `cw teams get` | `--emit` | `-E` |
| `cw teams update` | `--emit` | `-E` |
| `cw webhooks create` | `--emit` | `-E` |
| `cw webhooks update` | `--emit` | `-E` |

## Local Shorthand Reuse
All reused shortcodes map to the same long flag name (no mixed-meaning shortcodes found).

| Shorthand | Reuse Count | Long Flag Name(s) |
|---|---:|---|
| `-E` | 23 | `emit` |
| `-n` | 10 | `name` |
| `-s` | 7 | `status` |
| `-c` | 5 | `content` |
| `-m` | 4 | `message` |
| `-p` | 4 | `page` |
| `-L` | 3 | `labels` |
| `-P` | 3 | `private` |
| `-R` | 3 | `resolve` |
| `-S` | 3 | `since` |
| `-l` | 3 | `limit` |
| `-T` | 2 | `type` |
| `-a` | 2 | `all` |
| `-e` | 2 | `email` |

## Notes
- No true collisions were detected in either command-token scope or effective flag shorthand scope.
- Alias semantics are now consistent across the tree (same alias token => same command name).
