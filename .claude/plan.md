TITLE: Fix slow live stream feedback to UI
TYPE: fix

## Root Cause Analysis

After thorough investigation, the slowness is **primarily our application**, not Claude. I found 3 bugs/bottlenecks in the streaming pipeline, ranked by severity:

### Bug 1 (CRITICAL): WritePump silently drops batched messages

**File:** `internal/ws/client.go:99-103`

```go
w.Write(message)
// Drain any queued messages into the same write frame.
n := len(c.send)
for i := 0; i < n; i++ {
    w.Write(<-c.send)
}
```

When messages queue up in the channel (which happens constantly during streaming), the drain loop writes multiple JSON objects into **the same WebSocket frame** without any delimiter. The frontend receives one `event.data` containing concatenated JSON like:

```
{"type":"output","content":"hello"}{"type":"output","content":"world"}
```

`JSON.parse()` fails on this → the `catch` block silently discards it → **messages are lost**. During fast streaming, most messages arrive in batches, so the majority of output is silently dropped. This is the primary cause of perceived slowness.

### Bug 2 (HIGH): Parser skips streaming delta events

**File:** `internal/claude/parser.go:51-61`

The parser only extracts `Content` from `"assistant"` and `"result"` events. The Claude CLI's `stream-json` format also emits intermediate event types (like `content_block_start`, `content_block_delta`) that carry incremental text. These fall into the `default` case which leaves `Content` empty. The frontend then filters them out (`msg.content` is falsy). The user only sees output when a complete assistant message lands — not token-by-token.

### Issue 3 (MEDIUM): Synchronous DB write blocks streaming pipeline

**File:** `internal/workflow/engine.go:171`

```go
_ = e.queries.CreateTaskLog(taskID, stage, ev.Content)
```

Every event triggers a synchronous SQLite INSERT **before** broadcasting to WebSocket. SQLite writes with WAL can take 1-10ms each. During fast streaming (dozens of events/second), this creates a bottleneck that delays broadcasts.

### Issue 4 (LOW): Frontend performance - array copy per message

**File:** `frontend/src/hooks/useWebSocket.ts:27`

```typescript
setMessages((prev) => [...prev, msg]);
```

Creates a full array copy on every message. For long-running tasks with thousands of messages, this becomes O(n) per message. Combined with un-memoized `outputLines` filtering on every render.

---

## Implementation Plan

### Step 1: Fix WritePump message framing (Bug 1)

**File:** `internal/ws/client.go`

Change the drain loop to send each queued message as its **own WebSocket frame** instead of concatenating into one:

```go
case message, ok := <-c.send:
    c.conn.SetWriteDeadline(time.Now().Add(writeWait))
    if !ok {
        c.conn.WriteMessage(websocket.CloseMessage, []byte{})
        return
    }

    if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
        return
    }

    // Drain any queued messages — each as its own frame.
    n := len(c.send)
    for i := 0; i < n; i++ {
        if err := c.conn.WriteMessage(websocket.TextMessage, <-c.send); err != nil {
            return
        }
    }
```

This preserves the drain-the-queue optimization (reduces goroutine wake-ups) while ensuring each message is a valid, independent WebSocket frame that `JSON.parse()` can handle.

### Step 2: Handle streaming delta events in parser (Bug 2)

**File:** `internal/claude/parser.go`

Add handling for `content_block_delta` events to extract incremental text:

```go
case "content_block_delta":
    ev.Content = extractDeltaText(msg.Message)
```

Add a new struct + extractor for the delta format:
```go
type deltaPayload struct {
    Delta struct {
        Type string `json:"type"`
        Text string `json:"text"`
    } `json:"delta"`
}

func extractDeltaText(raw json.RawMessage) string {
    var dp deltaPayload
    if err := json.Unmarshal(raw, &dp); err != nil {
        return ""
    }
    return dp.Delta.Text
}
```

### Step 3: Make DB log writes non-blocking (Issue 3)

**File:** `internal/workflow/engine.go`

Move the `CreateTaskLog` call to a goroutine so it doesn't block the broadcast:

```go
onEvent := func(ev claude.Event) {
    // Store log asynchronously — don't block the broadcast.
    go func() {
        _ = e.queries.CreateTaskLog(taskID, stage, ev.Content)
    }()

    // Broadcast to WebSocket clients
    msg, _ := json.Marshal(map[string]interface{}{
        "type":    "output",
        "stage":   stage,
        "content": ev.Content,
        "raw":     json.RawMessage(ev.Raw),
    })
    e.hub.Broadcast(taskID, msg)
}
```

### Step 4: Optimize frontend message handling (Issue 4)

**File:** `frontend/src/pages/TaskDetail.tsx`

Memoize `outputLines` so it doesn't recompute on unrelated renders:

```typescript
const outputLines = useMemo(() => {
    const lines: string[] = [];
    for (const msg of messages) {
        if (msg.type === 'output' && msg.content) {
            lines.push(msg.content);
        }
    }
    return lines;
}, [messages]);
```

---

## Tests to Write

### `internal/ws/client_test.go` (new)
- **Test batched messages are sent as separate frames**: Send multiple messages rapidly through a client's send channel, read from a mock WebSocket connection, and verify each message arrives as valid independent JSON.

### `internal/claude/parser_test.go` (new or extend)
- **Test content_block_delta parsing**: Verify that `ParseEvent` extracts text from `content_block_delta` events.
- **Test assistant event parsing**: Verify existing behavior still works.
- **Test unknown event types**: Verify unknown types return empty Content (existing behavior).

### `internal/workflow/engine_test.go` (extend if exists)
- **Test onEvent broadcasts without blocking on DB**: Verify that the broadcast happens even if DB is slow (mock a slow DB).

---

## Files to Modify

| File | Change |
|------|--------|
| `internal/ws/client.go` | Fix WritePump to send separate frames per message |
| `internal/claude/parser.go` | Handle `content_block_delta` events |
| `internal/workflow/engine.go` | Make `CreateTaskLog` async |
| `frontend/src/pages/TaskDetail.tsx` | Memoize `outputLines` |

## Files to Create

| File | Purpose |
|------|---------|
| `internal/ws/client_test.go` | Test frame-per-message behavior |
| `internal/claude/parser_test.go` | Test delta event parsing |

---

## Risk Assessment

- **Step 1** (WritePump fix): Low risk. Each message already contains complete JSON. Sending as separate frames is the correct WebSocket pattern. The only tradeoff is slightly more frame overhead, which is negligible.
- **Step 2** (Parser delta events): Low risk. Additive change — existing event types keep working. If the CLI format doesn't use `content_block_delta`, the new code simply never triggers.
- **Step 3** (Async DB writes): Low risk. Log writes are fire-and-forget (return value already discarded). Only concern is potential SQLite write contention under extreme load, but SQLite handles this via WAL.
- **Step 4** (Frontend optimization): Low risk. Behavioral change is minimal — same data, just computed more efficiently.

## Implementation Order

1. Fix WritePump framing (highest impact, fixes silent message loss)
2. Add delta event parsing (enables real-time token streaming)
3. Async DB writes (unblocks the streaming pipeline)
4. Frontend optimization (polish)
5. Tests
