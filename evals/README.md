# Agent Mode Evals

Evaluation suite for chatwoot-cli agent mode features.

## Prerequisites

- Authenticated chatwoot CLI (`chatwoot auth login`)
- Access to a Chatwoot instance with test data

## Running Evals

```bash
# Run all evals
./evals/run-all.sh

# Run specific scenario
./evals/scenarios/01-since-flag.sh

# Run with verbose output
VERBOSE=1 ./evals/run-all.sh
```

## Results

Results are written to `evals/results/` with timestamps.

## Adding New Evals

1. Create a new script in `evals/scenarios/`
2. Source `../lib/helpers.sh`
3. Use `run_eval`, `assert_json_field`, `assert_contains` helpers
4. Return 0 for pass, 1 for fail
