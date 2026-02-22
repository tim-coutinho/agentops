# Transport Reference

## MCP Tool Reference

| Tool | Purpose |
|------|---------|
| `fetch_inbox` | Get messages for agent |
| `search_messages` | Find messages by query |
| `summarize_thread` | Summarize a thread |
| `acknowledge_message` | Mark message as read |

## HTTP Fallback

If MCP tools unavailable but HTTP server running:

```bash
# Fetch inbox via HTTP
curl -s "http://localhost:8765/mcp/" \
  -H "Content-Type: application/json" \
  -d '{
    "method": "fetch_inbox",
    "params": {
      "project_key": "<project-key>",
      "agent_name": "<agent-name>"
    }
  }'

# Search messages
curl -s "http://localhost:8765/mcp/" \
  -H "Content-Type: application/json" \
  -d '{
    "method": "search_messages",
    "params": {
      "project_key": "<project-key>",
      "query": "HELP_REQUEST"
    }
  }'
```

## Without Agent Mail

If Agent Mail is not available:

```markdown
Agent Mail not available.

To enable:
1. Start MCP Agent Mail server:
   Start your Agent Mail MCP server (implementation-specific). See `docs/agent-mail.md`.

2. Add to ~/.claude/mcp_servers.json:
   {
     "mcp-agent-mail": {
       "type": "http",
       "url": "http://127.0.0.1:8765/mcp/"
     }
   }

3. Restart Claude Code session
```
