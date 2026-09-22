# agentx-protocol

Unified transcript format shared by [agentx](https://github.com/ArcheMind/agentx)
and downstream consumers (e.g. the chill Someone Agent Harness).

- **Transcript spec**: [docs/unified-transcript-format.md](docs/unified-transcript-format.md)
- **Hook lifecycle spec**: [docs/unified-hook-lifecycle.md](docs/unified-hook-lifecycle.md)
- **Hook configuration spec**: [docs/unified-hook-configuration.md](docs/unified-hook-configuration.md)
- **Normative schema**: [schema/unified-transcript.schema.json](schema/unified-transcript.schema.json)
- **Go**: `github.com/ArcheMind/agentx-protocol/transcript`
- **TypeScript**: npm package `agentx-protocol` (install from git, pinned to a commit)
- **Fixtures**: [fixtures/](fixtures/) — shared conformance corpus; both
  implementations must accept every `valid/` session and reject every
  `invalid/` session (`invalid/` files wrap the session with a `reason`)

## Rules

Consumers import an implementation from this repository; they never define
their own transcript message shapes. Format changes land here first — spec +
schema + both implementations + fixtures in one commit — then consumers
upgrade their pinned version.

## Develop

```
make test   # go test + tsc + node --test
```
