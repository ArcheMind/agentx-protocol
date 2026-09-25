# Unified Transcript Format

The unified transcript format normalizes agent conversation histories from
multiple providers into a single structured shape. This repository is the
protocol's home: the specification (this document), the machine-readable
JSON Schema, and the reference implementations.

- Normative schema: [`schema/unified-transcript.schema.json`](../schema/unified-transcript.schema.json)
- Go implementation: [`transcript/`](../transcript) (`github.com/ArcheMind/agentx-protocol/transcript`)
- TypeScript implementation: [`src/`](../src) (npm package `agentx-protocol`)
- Shared conformance fixtures: [`fixtures/`](../fixtures)

Consumers (agentx, chill harness, …) import an implementation from this
repository; they do not define their own transcript message shapes. Format
changes land here first — spec + schema + both implementations + fixtures in
one commit — then consumers upgrade their pinned version.

## Session envelope

```json
{
  "id": "string",
  "provider": "string (e.g. claude | codex | gemini | opencode | pi)",
  "isSubagent": false,
  "workspace": "/absolute/path",
  "title": "first user intent (≤96 chars)",
  "startedAt": "RFC3339Nano UTC",
  "updatedAt": "RFC3339Nano UTC",
  "messageCount": 0,
  "source": "native file path",
  "messages": [ "...Message[]" ]
}
```

## Message

Each message is one event in the conversation. Tool results are independent
messages with `role: "tool"`, never disguised as `role: "user"`.

```json
{
  "id": "string",
  "parentId": "string | null",
  "role": "user | assistant | tool",
  "content": [ "...ContentBlock[]" ],
  "timestamp": "RFC3339Nano UTC",
  "model": "string (optional, assistant only)",
  "stopReason": "string (optional, assistant only)",
  "toolUseId": "string (required for role=tool, references a tool_use block id)",
  "toolName": "string (required for role=tool)",
  "isError": "boolean (optional, tool only; true when the tool execution failed)"
}
```

### Roles

| Role | Meaning |
|---|---|
| `user` | Human input. Only `text` blocks. |
| `assistant` | Model output. May contain text, thinking, and tool_use blocks. |
| `tool` | Tool execution result. Only `text` blocks. Links back to a tool_use block via `toolUseId`; `isError: true` marks a failed execution. |

### Causal chain

`parentId` links each message to its causal predecessor, forming a linear
chain within a turn. The first message of a session has `parentId: null`.

## ContentBlock

Content is an ordered array of typed blocks. The `type` field discriminates.

### text

```json
{ "type": "text", "text": "string" }
```

### thinking

Model reasoning. Nullable — absent when the provider does not expose it
(Codex, Gemini, OpenCode). When present, may be plaintext (Claude) or
opaque/encrypted (Pi). View derivation preserves thinking blocks as-is;
consumers that cannot use them simply ignore the block.

```json
{
  "type": "thinking",
  "text": "string (empty if encrypted)",
  "signature": "string (optional, provider-specific integrity proof)"
}
```

### tool_use

Tool invocation request. Always inside an `assistant` message.

```json
{
  "type": "tool_use",
  "id": "string (unique, referenced by tool message's toolUseId)",
  "name": "string (tool name)",
  "input": {}
}
```

## Provider normalization rules

### Claude Code

Source: `~/.claude/projects/*/*.jsonl`

| Native | Unified |
|---|---|
| `type:"assistant"` event, `content[].type:"thinking"` | `thinking` block |
| `type:"assistant"` event, `content[].type:"text"` | `text` block |
| `type:"assistant"` event, `content[].type:"tool_use"` | `tool_use` block |
| `type:"user"` event with `tool_result` content | Split to `role:"tool"` message; `tool_use_id` → `toolUseId`; `is_error` → `isError` |
| `type:"user"` event without `tool_result` | `role:"user"` message |
| `message.model` | `model` field |
| `stop_reason` | `stopReason` field |
| `parentUuid` | `parentId` |

### Codex CLI

Source: `~/.codex/sessions/*/*/*/*/*.jsonl`

| Native | Unified |
|---|---|
| `response_item` with `payload.type:"message"`, `role:"user"` | `role:"user"` message; `input_text` blocks → `text` blocks |
| `response_item` with `payload.type:"message"`, `role:"assistant"` | `role:"assistant"` message; `output_text` → `text` blocks |
| `response_item` with `payload.type:"function_call"` | `tool_use` block appended to preceding assistant message; `arguments` (JSON string) → `input` (parsed object) |
| `response_item` with `payload.type:"function_call_output"` | `role:"tool"` message; `call_id` → `toolUseId`; `name` from matched function_call; `output` → `text` block |
| `ordinal` | Ordering tiebreak; not mapped to `parentId` (derive from sequence) |
| `session_meta.source.subagent` | `isSubagent` |

### Gemini CLI

Source: `~/.gemini/tmp/*/chats/session-*.json`

| Native | Unified |
|---|---|
| `type:"user"` | `role:"user"` message |
| `type:"gemini"`, `content` text | `role:"assistant"` message, `text` block |
| `type:"gemini"`, `toolCalls[]` | Unpack: each `toolCall` → `tool_use` block in assistant message; each `toolCall.result[].functionResponse` → separate `role:"tool"` message; `toolCall.id` links both via `toolUseId` |
| `type:"info"` | Dropped (system notifications) |
| `kind:"subagent"` | `isSubagent` |

### Pi

Source: `~/.pi/agent/sessions/*/*.jsonl`

| Native | Unified |
|---|---|
| `message.role:"assistant"`, `content[].type:"thinking"` | `thinking` block; `thinkingSignature` → `signature` |
| `message.role:"assistant"`, `content[].type:"toolCall"` | `tool_use` block; `toolCallId` → `id`; `toolName` → `name` |
| `message.role:"assistant"`, `content[].type:"text"` | `text` block |
| `message.role:"toolResult"` | `role:"tool"` message; `toolCallId` → `toolUseId`; `toolName` → `toolName`; `isError` → `isError` |
| `message.role:"user"` | `role:"user"` message |
| `parentId` | `parentId` (direct mapping) |

### OpenCode

Source: `~/.local/share/opencode/project/*/storage/session/`

| Native | Unified |
|---|---|
| `message.role:"user"` | `role:"user"` message |
| `message.role:"assistant"` | `role:"assistant"` message |
| `part.type:"text"` | `text` block |
| `part.type:"step-start"` | `tool_use` block (id = part id, name = "step", input = {}) |
| `part.type:"step-finish"` | `role:"tool"` message; content = `text` block with token summary |
| `part.type:"patch"` | `tool_use` block (name = "patch", input = {hash, files}) |
| `parentID` (non-empty) | `isSubagent` |

## View derivation (offload)

A view is a lossy projection of the full transcript for bounded-context
replay. The full transcript is the golden standard and is never mutated.

### Rules

1. **Thinking offload**: Remove `thinking` blocks from the view. The full
   transcript retains them; the view builder writes them to offload files
   (same layout as tool results). Consumers that need reasoning access it
   via the offload path, not the inline view.
2. **Tool result routing**: Partition `role:"tool"` messages by `toolName`:
   - Inline list (configurable, e.g. `["cave"]`): Keep in view; content
     summarized to first/last 2 lines + pointer to full output.
   - Everything else: Remove from view entirely; write full content to
     offload file.
3. **Turn folding**: A turn boundary is a `role:"user"` message. Within each
   turn, keep only the closing assistant message (last assistant message with
   non-empty text). Non-closing assistant messages without inline tool_use
   blocks are removed from the view (full content written to offload file).
4. **Size cap**: Total view text capped at a configurable limit (default
   120,000 characters). Exceeded → keep head + tail, insert truncation
   marker.

### Offload file layout

```
history/<msgIndex>/<callSeq>.out.txt   — tool result full output
history/<msgIndex>.md                  — full assistant message text
```

`msgIndex` = 0-based position in the transcript (append-only, stable across
turns). `callSeq` = 0-based tool_use index within that message.

## Validation

The normative constraints live in
[`schema/unified-transcript.schema.json`](../schema/unified-transcript.schema.json).
Both implementations expose a validator enforcing the schema's structural
rules plus message-id uniqueness within a session. Cross-message referential
checks (`parentId` / `toolUseId` resolution) are intentionally not enforced:
native sessions may be truncated mid-chain.
