# Apple Ads CLI by Lily

Apple Ads CLI by Lily is an open-source, AI-friendly, business-first command
line tool for planning Apple Ads campaign packages, reviewing AI-agent generated
changes, and supporting Campaign Management API v5 and Platform API v1.

Apple Ads is an Apple brand. Apple Ads CLI by Lily is an independent,
unofficial tool and is not affiliated with, endorsed by, or sponsored by Apple.

## Why This Exists

Most Apple Ads automation tools expose raw endpoint-shaped commands. Lily takes
a business-first approach:

- plan a campaign package before writing to Apple Ads
- create campaign, ad group, keywords, negative keywords, CPA goal, and creative
  attachment in one command
- return JSON by default so Codex, Claude Code, CI, and humans can review plans
  through business summaries and confirmation choices
- keep the current v5 API working while preparing for Apple Ads Platform API v1
- optionally add keyword-level revenue analytics through Lily with RevenueCat
  or AppsFlyer

## Install

Install with Homebrew:

```sh
brew install crevas/tap/lilyads
```

Or use the hosted install script:

```sh
curl -fsSL https://www.chatlily.ai/apple-ads-cli/install | bash
```

Install from source with Go:

```sh
go install github.com/crevas/Apple-Ads-CLI/cmd/lily@latest
```

Or build locally:

```sh
git clone https://github.com/crevas/Apple-Ads-CLI.git
cd Apple-Ads-CLI
go build -o bin/lily ./cmd/lily
```

Release binaries are also available through the repository install script:

```sh
curl -fsSL https://raw.githubusercontent.com/crevas/Apple-Ads-CLI/main/install.sh | bash
```

## Quick Start

Run a local readiness check:

```sh
lily ads platform readiness
```

Preview a business plan without writing to Apple Ads:

```sh
lily ads plan create \
  --name "AwayFinder UK Category" \
  --app-id 999999999 \
  --country UK \
  --daily-budget 300 \
  --currency USD \
  --adgroup "AwayFinder UK Keywords" \
  --bid 2.00 \
  --cpa-goal 12.00 \
  --exact-keywords "flight booking,cheap flights" \
  --broad-keywords "travel app,holiday planner" \
  --negative-exact "jobs,wallpaper" \
  --campaign-negative-broad "free games" \
  --creative product-page:pp_123456789
```

Dry-run is the default. Add `--yes` or `--execute` to perform writes.
Dry-run responses include `review` and `confirmation` objects so AI agents can
show a business summary and ask the user to confirm, modify, or cancel.

Preview the same business command against the next-generation Platform
provider:

```sh
lily --provider platform ads plan create \
  --name "AwayFinder UK Category" \
  --app-id 999999999 \
  --country GB \
  --daily-budget 300 \
  --bid 2.00 \
  --cpa-goal 12.00 \
  --exact-keywords "flight booking,cheap flights"
```

## Configuration

Environment variables:

```sh
export APPLE_ADS_PROVIDER=campaignv5
export APPLE_ADS_CLIENT_ID=...
export APPLE_ADS_TEAM_ID=...
export APPLE_ADS_KEY_ID=...
export APPLE_ADS_ORG_ID=...
export APPLE_ADS_PRIVATE_KEY_PATH=/path/to/AuthKey.p8
```

For the Platform API provider:

```sh
export APPLE_ADS_PROVIDER=platform
export APPLE_ADS_AD_ACCOUNT_ID=...
```

Optional:

```sh
export APPLE_ADS_CURRENCY=USD
export APPLE_ADS_V5_BASE_URL=https://api.searchads.apple.com/api/v5
export APPLE_ADS_PLATFORM_BASE_URL=https://api.ads.apple.com/v1
export LILY_CLOUD_BASE_URL=https://www.chatlily.ai
```

Older `LILY_ADS_*` aliases are still supported for compatibility, but new local
Apple Ads credentials should use the `APPLE_ADS_*` prefix.

You can also create:

```txt
~/.config/lily/apple-ads.json
```

with matching JSON keys from `internal/config.Config`.

## AI Agent Contract

The default output is JSON. Write commands are dry-run by default.

The main business-first command is:

```sh
lily ads plan create
```

It plans or executes:

1. Create campaign.
2. Create ad group.
3. Bulk-create keywords.
4. Bulk-create campaign and ad group negative keywords.
5. Optionally create/select creative assets and attach an ad.

Agents should:

- call without `--yes` first
- summarize the `review` object in business terms
- use the `confirmation` object to ask the user to confirm, modify, or cancel
- use native confirmation UI when the host application provides it
- avoid showing shell commands or file paths to non-technical users unless asked
- re-run with `--yes` only after explicit user confirmation
- use `--correlation-id` to connect CLI output to an agent trace

## Agent Skills

This repository includes AI agent skills for Codex and Claude Code:

```txt
agent-skills/codex/apple-ads-cli-by-lily
agent-skills/claude/apple-ads-cli-by-lily
```

Install the Codex skill:

```sh
mkdir -p ~/.codex/skills
cp -R agent-skills/codex/apple-ads-cli-by-lily ~/.codex/skills/
```

Install the Claude Code skill:

```sh
mkdir -p ~/.claude/skills
cp -R agent-skills/claude/apple-ads-cli-by-lily ~/.claude/skills/
```

Both skills teach agents the safe Lily workflow: local Apple Ads credentials for
Apple operations, optional Lily Ads Revenue Analytics for revenue/ROAS, dry-run
plans first, and user-facing confirmation choices before writes.

## Lily Ads Revenue Analytics

Core Apple Ads operations are free and open source. Apple Ads API credentials
are configured locally through environment variables or
`~/.config/lily/apple-ads.json`; private keys stay on your machine. Lily login is
optional and only enables revenue enrichment through Lily Ads Revenue Analytics:

```sh
lily login --token <token>
lily ads revenue summary --app-id 999999999 --from 2026-06-01 --to 2026-06-30
lily ads reports campaigns --app-id 999999999 --from 2026-06-01 --to 2026-06-30
```

When Lily login or commercial activation is missing, the CLI still returns the
Apple Ads result where possible and appends a structured revenue notice. This
helps agents explain that paid-user status and ROAS cannot be calculated until
Lily Ads Revenue Analytics is activated.

## Platform API Readiness

The CLI hides API shape changes behind providers:

```txt
campaignv5 -> https://api.searchads.apple.com/api/v5
platform   -> https://api.ads.apple.com/v1
```

Known compatibility differences handled by the providers:

- v5 context header uses `orgId`; Platform uses `adAccountId`
- v5 successful responses use `data`; Platform uses `result`
- v5 find/get-all patterns are replaced by Platform `/query` patterns
- v5 campaign payloads use `adamId`; Platform campaigns use promoted objects
- v5 keyword status uses `ACTIVE`; Platform keyword status uses `ENABLED`
- Platform bulk keyword creation uses `/keywords/bulk-create`

Reserved Platform recommendation commands:

```sh
lily ads suggestions cpa --app-id 999999999
lily ads recommendations apply --type target-cpa --app-id 999999999
```

They currently return `status: reserved` and are designed to be enabled as soon
as Apple exposes target CPA recommendation/suggestion endpoint contracts.

## Commands

```sh
lily login --token <token>
lily logout
lily auth status
lily ads doctor
lily ads platform readiness
lily ads plan create [flags]
lily ads reports campaigns [flags]
lily ads revenue summary [flags]
lily ads suggestions cpa --app-id <adamId>
lily ads recommendations apply --type target-cpa
```

Planned next commands:

```sh
lily ads reports keywords
lily ads reports search-terms
lily ads query campaigns
lily ads migrate plan
lily ads change-history query
```

## License

MIT.

This project is an independent, unofficial tool and is not affiliated with,
endorsed by, or sponsored by Apple Inc. Apple, App Store, Apple Ads, and related
marks are trademarks of Apple Inc.
