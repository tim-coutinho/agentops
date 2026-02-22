# Watch Mode

When `--watch` flag is provided:

## Step W1: Enter Polling Loop

```bash
POLL_INTERVAL=30  # seconds

while true; do
    clear
    echo "=== Agent Mail Inbox (Watch Mode) ==="
    echo "Press Ctrl+C to exit"
    echo ""

    # Execute Steps 3-7 from main workflow

    # Show last update time
    echo ""
    echo "Last updated: $(date)"
    echo "Next refresh in ${POLL_INTERVAL}s"

    sleep $POLL_INTERVAL
done
```

## Step W2: Alert on New Messages

When new messages arrive since last poll:

```bash
# Compare message counts
if [ "$NEW_COUNT" -gt "$PREV_COUNT" ]; then
    echo "*** NEW MESSAGES ($((NEW_COUNT - PREV_COUNT))) ***"
    # Optionally use terminal bell
    echo -e "\a"
fi

# Highlight urgent HELP_REQUESTs
if [ "$HELP_WAITING_MINS" -gt 5 ]; then
    echo "*** URGENT: HELP_REQUEST waiting ${HELP_WAITING_MINS}m ***"
fi
```

## Step W3: Show Message Summaries

For each new message, display summary:

```markdown
---
**NEW:** From demigod-gt-125 | Thread: gt-125 | 10s ago
Subject: PROGRESS
> Step 4 in progress. Files touched: src/auth.py, tests/test_auth.py
---
```
