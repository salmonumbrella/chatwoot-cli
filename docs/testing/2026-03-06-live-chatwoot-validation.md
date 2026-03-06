# Live Chatwoot Validation Runbook

Date: 2026-03-06  
Operator: Codex  
Workspace: `/Users/sadimir/code/experiments/chatwoot-cli`  
Artifacts: `/var/folders/_8/zxb6zrf15csd95ss0q_475180000gn/T//chatwoot-cli-live-validation`

## Environment Preflight

- Status: pass
- Command:
  - `bash scripts/live-validation/preflight.sh`
- Observed profile:
  - Profile: `default`
  - Base URL: `https://chatwoot.wanver.shop`
  - Account ID: `1`

## Authenticated Profile Snapshot

- Status: pass
- Notes:
  - `cw st -o json --compact-json` confirmed the live account was authenticated before every mutating phase.
  - The authenticated snapshot is saved as `artifacts/preflight.json`.

## Selected Live Fixtures

- Text-only conversation:
  - ID: `44269`
  - Why selected: recent real conversation with bounded message history and no attachments in the inspected slice.
- Private-note conversation:
  - ID: `24430`
  - Why selected: real conversation where `cw ct --tail 5 --public-only` returned `public_only=true` and excluded private notes.
- Image-attachment conversation:
  - ID: `24004`
  - Why selected: real conversation with image attachments and only `20` total messages, which made `cw ctx --exclude-attachments` practical to validate live.
- Document-attachment conversation:
  - ID: `20590`
  - Why selected: real conversation with multiple PDF attachments hosted on external object storage and good extraction quality via `pdftotext`.

## Scratch Resources

- Status: pass
- Scratch label: `cli-live-validation`
- Scratch contact ID: `223042`
- Scratch conversation ID: `44270`
- Scratch inbox ID: `50`
- Baseline assignee: agent `15` (`21`)
- Notes:
  - Scratch resources live in a `Channel::Api` inbox so validation does not touch a real customer thread.
  - The scratch conversation was normalized before validation to `open`, assigned to agent `15`, and labeled `cli-live-validation`.

## Read-Only Validation Results

- Status: pass
- Commands validated:
  - `cw --help-json`
  - `cw ctx --help-json`
  - `cw assign --help-json`
  - `cw c ls --help-json`
  - `cw sc ls -o json --compact-json`
  - `cw auth skill`
  - `cw ct 44269 -o json --compact-json`
  - `cw ct 24430 --tail 5 --public-only -o agent`
  - `cw ct 24004 --tail 5 --exclude-attachments -o json --compact-json`
- Assertions that passed:
  - `ctx --help-json` exposed positional args plus `tail`, `public-only`, `exclude-attachments`, `embed-images`, and `light`.
  - `assign --help-json` exposed `mutates=true` and `supports_dry_run=true`.
  - `c ls --help-json` exposed `field_schema=conversation` and field presets.
  - `auth skill` regenerated the workspace skill at `/Users/sadimir/.claude/skills/chatwoot-workspace/SKILL.md`.
  - `ctx --public-only` set `meta.public_only=true` and excluded private notes.
  - `ctx --exclude-attachments` set `meta.exclude_attachments=true` and removed attachment metadata from returned messages.
- Live note:
  - Fixture discovery and read validation had to avoid very large image threads because `cw ctx` still walks full history before tailing. The harness was updated to choose a bounded real image fixture instead of the largest recent image conversation.

## Mutation Validation Results

- Status: pass
- Dry-run commands:
  - `cw assign 44270 --agent 15 --dry-run -o json --compact-json`
  - `cw close 44270 --dry-run -o json --compact-json`
  - `cw reopen 44270 --dry-run -o json --compact-json`
- Dry-run result:
  - All previews returned `dry_run=true`.
  - Before/after scratch conversation snapshots were identical.
- Real reversible mutations on scratch conversation `44270`:
  - Assign agent `15 -> 5 -> 15`
  - Close `open -> resolved`
  - Reopen `resolved -> open`
  - Set custom attribute `cli_validation=active`
  - Add temporary label `cli-live-validation-mutation`
- Restoration result:
  - Final scratch state restored to `open`, assignee `15`, label `cli-live-validation`, and empty conversation custom attributes.
- Live note:
  - On this account, the conversation labels endpoint behaved like label replacement rather than additive merge.
  - Label mutation also caused the assignee to drift to agent `4` (`Vladimir`) during the live run, so the harness explicitly reapplies the baseline assignee before final equality checks.

## Raw API Validation Results

- Status: pass
- Scratch contact used: `223042`
- Live-safe commands validated:
  - object-body patch:
    - `cw ap /contacts/223042 -X PATCH -d '{"name":"CLI Live Validation","custom_attributes":{"owner":"codex","cli_validation":"raw-api-object"}}' --include -o json --compact-json`
  - object-body merge with `--field` and `--raw-field`:
    - `cw ap /contacts/223042 -X PATCH -d '{"name":"CLI Live Validation"}' -f name=CLI\ Live\ Validation -F 'custom_attributes={"owner":"codex","cli_validation":"raw-api-merged","mode":"merged"}' --include -o json --compact-json`
  - oversize rejection before request execution:
    - oversized `-i` file failed locally with `JSON payload exceeds maximum size of 1048576 bytes`
  - invalid merge rejection before request execution:
    - mixed non-object `-d '["bad"]' -f name=...` failed locally with `cannot use --field or --raw-field with a non-object JSON body`
- Boundary decision:
  - top-level array, scalar, and explicit `null` request bodies remain covered by automated tests only; Chatwoot does not expose a safe production endpoint here that justifies forcing those shapes onto live data.
- Live note:
  - Contact `custom_attributes` patching on this Chatwoot account is merge-semantic, not replace-semantic. Absent keys are not cleared by a later PATCH.

## Attachment / Document Characterization

- Status: pass
- Document fixture conversation ID: `20590`
- Sample attachments characterized:
  - Sample 1: PDF, `991117` bytes, host `wanver-chatwoot.sfo3.digitaloceanspaces.com`, `pdftotext_ok`
  - Sample 2: PDF, `635280` bytes, host `wanver-chatwoot.sfo3.digitaloceanspaces.com`, `pdftotext_ok`
  - Sample 3: PDF, `444573` bytes, host `wanver-chatwoot.sfo3.digitaloceanspaces.com`, `pdftotext_ok`
- Extraction quality:
  - All three sampled PDFs extracted cleanly with `pdftotext`.
  - The extracted text was usable immediately for future document-analysis workflows.
- Recommendation:
  - Support `pdf` first.
  - Stream downloads to disk.
  - Return extracted text plus metadata, not base64 document blobs.
  - Keep per-file and per-command byte limits even if image embedding remains separately capped.

## Attachment Extraction Validation

- Status: pass
- Commands validated:
  - `cw c attachments 20590 --light --compact-json`
  - `cw c attachments extract 20590 --limit 0 --light --compact-json`
  - `cw c attachments extract 20590 --index 3 --light --compact-json`
  - `cw c attachments extract 20590 --index 10 --light --compact-json`
- Assertions that passed:
  - `attachments --light` returned compact attachment metadata with indexes, sizes, and names, without signed attachment URLs.
  - `attachments extract --limit 0 --light` extracted all `7` supported document attachments in the live fixture conversation.
  - PDF extraction succeeded with extractor `pdftotext`.
  - XLSX extraction succeeded with extractor `xlsx-xml`.
  - The first XLSX extraction returned usable tabular text beginning with `[sheet1.xml]`.
- Live note:
  - The live Chatwoot attachments endpoint returned `id: 0` for these attachments, so extraction is keyed by attachment list index rather than attachment ID.

## `--li` / Token Audit

- Status: pass
- Audit outcome:
  - Darwin reviewed high-signal candidates and identified three best targets for this slice:
    - `messages create/update/retry`
    - `contacts search`
    - `search --best` / `search --select` when `--light` is set
- Implemented:
  - `messages create`, `messages update`, and `messages retry` now support `--light` / `--li` compact mutation payloads.
  - `contacts search` now supports `--light` / `--li`.
  - `search --best --light` and `search --select --light` now return compact light payloads instead of bypassing light mode.
  - `conversations attachments` now supports `--light` / `--li`.
  - `conversations attachments extract` returns compact extraction payloads for document analysis.
- Live smoke results:
  - `cw messages create 44270 --content 'CLI live validation compact message' --light -o agent --compact-json`
    - Returned `{"id":44270,"mid":192324}`
  - `cw messages update 44270 192324 --content 'CLI live validation compact message updated' --light -o agent --compact-json`
    - Returned `{"id":44270,"mid":192324}`
  - `cw contacts search --query cli-live-validation --light -o json --compact-json`
    - Returned compact contact identity only
  - `cw search cli-live-validation --type contacts --best --light -o json --compact-json`
    - Returned compact best-match payload with short-key item data
  - `cw c attachments 20590 --light --compact-json`
    - Returned compact attachment inventory with indexes, types, sizes, and derived filenames only
  - `cw c attachments extract 20590 --index 10 --light --compact-json`
    - Returned compact XLSX extraction payload with extractor `xlsx-xml`

## Company Skill Flow Tuning

- Status: pass
- Source skill reviewed:
  - `/Users/sadimir/.claude/skills/_wanver-agent-skills__clis__chatwoot-cli`
- Highest-friction flows observed in the live skill:
  - contact-to-conversation lookup needed `jq` to isolate the inbox-specific thread
  - message triage needed `jq` to keep only incoming/public messages and tail the last few
  - handoff previews and compact outputs were missing or too optimistic for agent use
- Implemented for those flows:
  - `cw co cv CONTACT --ib 28 --st open --l 1 --li`
    - native inbox/status/limit filtering on contact conversations
    - live result on contact `219788` returned a single compact conversation payload (`195` bytes)
  - `cw m ls 20590 --sla --in --tail 3 --li`
    - native incoming-only and tail filtering for triage reads
    - live result returned the last `3` incoming customer messages in `406` bytes
  - `cw contacts search --query '曼斐諾' --light`
    - compact contact lookup without full contact payloads
    - live result returned a `61` byte compact payload
  - `cw ho 44270 --agent 'Lily Hu' --team 'Billing' --priority urgent --reason 'CLI agent UX dry-run' --dry-run -o json`
    - preview now preserves agent/team names without forcing live resolution first
  - `cw ho 44270 --team 2 --priority high --reason 'CLI agent UX state-check' --li`
    - live compact mutation output now reflects the fetched post-mutation conversation state instead of echoing requested team/priority values
- Follow-up note from the live scratch thread:
  - On this Chatwoot account, team-only assignment on scratch conversation `44270` did not persist, while priority did.
  - The CLI now reports the actual resulting state, so the compact payload omitted `tm` and kept `pri:"h"` after the live handoff.

## Rollback Notes

- Scratch conversation final state:
  - `open`
  - assignee agent `15`
  - labels `["cli-live-validation"]`
  - conversation custom attributes empty
- Scratch contact final state:
  - name `CLI Live Validation`
  - email `cli-live-validation@example.com`
  - custom attributes currently `{"cli_validation": true, "mode": "merged", "owner": "codex"}`
- Residual scratch-only artifact:
  - The contact still retains `mode=merged` because live Chatwoot contact custom-attribute PATCHes merged keys instead of replacing them.

## Open Issues

- `cw ctx` on very large threads is still expensive for discovery workflows because it traverses full history before tailing.
- Conversation label mutation on the live account behaved like label replacement and also disturbed assignee state during the run.
- Contact custom-attribute PATCH semantics on the live account are merge-only, which matters for any future “reset to empty” workflow.

## Final Status

- Overall: pass
- Validated successfully:
  - richer `--help-json` command contracts
  - `ctx` metadata controls (`--tail`, `--public-only`, `--exclude-attachments`)
  - `auth skill` regeneration
  - dry-run guarantees for `assign`, `close`, and `reopen`
  - real reversible scratch mutations
  - live-safe raw API body/merge/oversize validation
  - real document download and extraction characterization
  - explicit live document extraction for `pdf` and `xlsx` attachments
  - new `--light` / `--li` compact outputs for `messages create/update/retry`, `contacts search`, `search --best` / `search --select`, and `conversations attachments`
- Validated only on scratch resources:
  - assign/close/reopen mutations
  - raw API mutation flows
  - compact `messages create/update` live smoke
- Behaviors still covered only by automated tests:
  - raw API top-level array/scalar/`null` body acceptance
  - `messages retry --light` on a genuinely failed live message
  - interactive `search --select --light` against the live account
