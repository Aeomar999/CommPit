# MCP Setup

Point an AI coding agent at your sandbox over MCP so it can read
messages, wait for OTPs and reset projects without touching the UI.

## Add the server

The MCP endpoint lives at `/mcp` on the main port (streamable HTTP):

```bash
claude mcp add --transport http mocksms http://127.0.0.1:4010/mcp
```

No `X-Mocksms` header is needed (Host allow-list protection still
applies). The server must be running — MCP calls the core service
directly, but through the HTTP transport, so there is no offline mode.

## Tools

| Tool | What it does | Key arguments |
|---|---|---|
| `list_messages` | List messages, newest first | `project`, `channel`, `to`, `from`, `status`, `direction`, `limit` (default 20, max 100) |
| `wait_for_message` | Block until a message arrives, then return it | `to`, `channel`, `since` (RFC3339), `timeout_seconds` (default 10, max 60) |
| `get_latest_otp` | Newest OTP code for a recipient | `to` (required) |
| `get_latest_email` | Newest email for a recipient | `to` (required) |
| `simulate_inbound` | Simulate an inbound SMS from a handset | `from`, `to`, `body` (all required) |
| `list_requests` | Provider-adapter request logs | `project`, `limit` |
| `reset_project` | Delete every message in a project | `project` |

Every tool takes an optional `project` (ULID). Empty means the default
project — the one keyless API calls resolve to. Results come back as
JSON text plus structured output.

## Typical session

```text
> Send a test SMS to +15005550006, then tell me the OTP.
Agent: [simulate inbound or waits] get_latest_otp(to=+15005550006)
<  {"code":"482916","source":"extracted","message_id":"msg_..."}
```

## Troubleshooting

- **Connection refused:** start the server first (`mocksms serve`).
- **Empty results:** check the project — traffic sent with unlinked
  credentials lands in auto-provisioned projects, not default. Link
  credentials on the Settings page or pass the ULID explicitly.
- **wait timeouts:** the default `since` is call time, so already-arrived
  messages never match — pass an explicit past `since`, or send after
  the wait starts.
