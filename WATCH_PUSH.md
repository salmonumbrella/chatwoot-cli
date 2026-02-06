# Push Watch For Agents (Webhook -> SSE)

Goal: let agents get notified immediately when a customer replies, without polling and without WebSockets.

Architecture:
- Chatwoot sends `message_created` webhooks (push) to a receiver.
- The receiver (`chatwoot watch serve`) streams events to agents over SSE.
- Agents run `chatwoot conversations follow <id|url>` to follow a conversation in real time.

Robustness features:
- The receiver buffers recent events per conversation in Redis (optional) and supports replay on reconnect via SSE `Last-Event-ID`.
- When Redis is enabled, multiple `chatwoot watch serve` instances can fan out live events via Redis Pub/Sub.
- The CLI follower automatically reconnects and (by default) falls back to polling if SSE is unavailable.

## Server Setup (Wanver: Caddy + docker compose)

This is a one-time setup on the server where Chatwoot is hosted.

### 1) Deploy the watch receiver container

Add a service to `~/code/wanver/wanver-shop-service-manager/docker-compose.yaml` (names/network should match your stack):

```yaml
  chatwoot-watch:
    container_name: chatwoot_watch_container
    build:
      context: /path/to/chatwoot-cli   # clone this repo on the server
      dockerfile: Dockerfile.watch
    restart: unless-stopped
    networks:
      - internal-network
    environment:
      # IMPORTANT: this is the backend URL used by the receiver to authorize agent tokens.
      # This can be the public URL, but internal is preferred.
      - CHATWOOT_BASE_URL=https://chatwoot.wanver.shop
      - CHATWOOT_ACCOUNT_ID=1

      # Optional but recommended: Redis for durable replay/dedupe across receiver restarts.
      # Your wanver stack already has `redis_container`.
      - CHATWOOT_WATCH_REDIS_URL=redis://:${REDIS_PASSWORD}@redis_container:6379/0

      # Allow private/internal base URLs if you point CHATWOOT_BASE_URL at a container name.
      # - CHATWOOT_ALLOW_PRIVATE=1
      # - CHATWOOT_BASE_URL=http://chatwoot_rails_container:3000

      # Shared token to validate inbound webhook POSTs.
      - CHATWOOT_WATCH_HOOK_TOKEN=REPLACE_WITH_LONG_RANDOM
    command:
      [
        "watch",
        "serve",
        "--bind", "0.0.0.0",
        "--port", "8789",
        "--hook-path", "/hooks/chatwoot",
        "--watch-path", "/watch",
        "--hook-token", "${CHATWOOT_WATCH_HOOK_TOKEN}",
        "--redis-url", "${CHATWOOT_WATCH_REDIS_URL}"
      ]
```

Notes:
- Using the public `CHATWOOT_BASE_URL` is simplest.
- If you switch to internal `CHATWOOT_BASE_URL` (recommended), set `CHATWOOT_ALLOW_PRIVATE=1`.

### 2) Route paths in Caddy

In `~/code/wanver/wanver-shop-service-manager/Caddyfile`, extend the `chatwoot.wanver.shop` site block:

```caddyfile
chatwoot.wanver.shop {
  handle /hooks/chatwoot* {
    reverse_proxy chatwoot_watch_container:8789
  }

  handle /watch* {
    reverse_proxy chatwoot_watch_container:8789
  }

  reverse_proxy chatwoot_rails_container:3000
}
```

### 3) Create the Chatwoot webhook

Create a webhook pointing to the receiver endpoint:

```bash
chatwoot watch setup \
  --url "https://chatwoot.wanver.shop/hooks/chatwoot" \
  --token "${CHATWOOT_WATCH_HOOK_TOKEN}" \
  --subscriptions message_created
```

You can also create this webhook in the Chatwoot UI. The important part is:
- URL includes the shared token as a query param: `?token=...`
- Subscriptions include `message_created`

Check status:

```bash
chatwoot watch status --url "https://chatwoot.wanver.shop/hooks/chatwoot"
```

## Agent Usage

Follow a conversation:

```bash
chatwoot conversations follow 123
```

Or with a URL:

```bash
chatwoot conversations follow "https://chatwoot.wanver.shop/app/accounts/1/conversations/123"
```

By default:
- prints the last 20 messages for context (`--tail 20`)
- then prints new incoming (customer) messages only (`--incoming-only`)
- if SSE disconnects, it falls back to polling (`--poll-fallback`)

Useful flags:
- `--tail 0` (don’t print history)
- `--incoming-only=false` (show outgoing/activity too)
- `--poll-fallback=false` (fail if SSE is down)
- `--poll-interval 5` (seconds)

## Security Notes

- Webhook receiver auth uses a shared token in the webhook URL query string (because Chatwoot webhooks cannot add custom headers).
- SSE stream auth uses the agent’s normal Chatwoot `api_access_token` header; the receiver validates access by calling Chatwoot’s conversation API with that token.

