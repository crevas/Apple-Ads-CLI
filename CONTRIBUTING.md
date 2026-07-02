# Contributing

Thanks for helping improve Apple Ads CLI by Lily.

## Development

```sh
go test ./...
go build -o bin/lily ./cmd/lily
```

Write commands should stay dry-run by default. If a command can mutate Apple Ads
state, it must require `--yes` or `--execute`.

## Design Principles

- Keep commands business-first, not endpoint-shaped.
- Keep JSON output stable for AI agents and CI.
- Keep provider-specific API shapes behind provider packages.
- Do not commit credentials, `.p8` keys, reports, snapshots, or logs.

## Pull Requests

Please include:

- a short description of the user-facing workflow
- provider impact: `campaignv5`, `platform`, or both
- tests for parsing, planning, or provider payload changes
- a note when a command changes the AI agent contract
