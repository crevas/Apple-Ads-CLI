# Apple Ads CLI by Lily

Apple Ads CLI by Lily is an open-source, AI-friendly command line tool for
Apple Ads campaign operations, reporting, and the Campaign Management API v5 to
Platform API v1 migration.

The binary command is:

```sh
lily
```

This repository contains only the open-source CLI. Lily's hosted dashboard,
credential vault, revenue warehouse, billing, and commercial analytics services
live outside this repository.

## Why This Exists

Most Apple Ads automation tools expose raw endpoint-shaped commands. Lily takes
a business-first approach:

- plan a campaign package before writing to Apple Ads
- create campaign, ad group, keywords, negative keywords, CPA goal, and creative
  attachment in one command
- return JSON by default so Codex, Claude Code, CI, and humans can review plans
- keep the current v5 API working while preparing for Apple Ads Platform API v1
- optionally enrich reports through Lily Ads Revenue Analytics

## Install

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

Release binaries will be attached to GitHub Releases:

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
export LILY_ADS_PROVIDER=campaignv5
export LILY_ADS_CLIENT_ID=...
export LILY_ADS_TEAM_ID=...
export LILY_ADS_KEY_ID=...
export LILY_ADS_ORG_ID=...
export LILY_ADS_PRIVATE_KEY_PATH=/path/to/AuthKey.p8
```

For the Platform API provider:

```sh
export LILY_ADS_PROVIDER=platform
export LILY_ADS_AD_ACCOUNT_ID=...
```

Optional:

```sh
export LILY_ADS_CURRENCY=USD
export LILY_ADS_V5_BASE_URL=https://api.searchads.apple.com/api/v5
export LILY_ADS_PLATFORM_BASE_URL=https://api.ads.apple.com/v1
export LILY_CLOUD_BASE_URL=https://www.chatlily.ai
```

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
- show the JSON plan to the user
- re-run with `--yes` only after explicit user confirmation
- use `--correlation-id` to connect CLI output to an agent trace

## Lily Ads Revenue Analytics

Core Apple Ads operations are free and open source. Revenue enrichment is
exposed through Lily Ads Revenue Analytics:

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
