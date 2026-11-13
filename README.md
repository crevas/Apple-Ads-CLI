# Apple Ads CLI by Lily

> **Lily 0.3.0 defaults to Apple Ads Platform API v1 (`platform`).**
> (2026-11-13) Explicit `campaignv5` configuration remains supported. Set
> `APPLE_ADS_AD_ACCOUNT_ID` before live requests. See the
> [v1 migration and compatibility audit](docs/v1-default-migration.md).
> Upgrading from 0.2.3: explicit v5 settings are preserved; unconfigured installs
> now use v1.

Apple Ads CLI by Lily is an open-source, AI-friendly, business-first command
line tool for planning Apple Ads campaign packages, reviewing AI-agent generated
changes, and supporting Apple Ads Platform API 1.0 (`v1`) alongside Campaign
Management API v5.

Apple Ads is an Apple brand. Apple Ads CLI by Lily is an independent,
unofficial tool and is not affiliated with, endorsed by, or sponsored by Apple.

## What Platform API 1.0 Unlocks for Advertisers

The important change is not `adAccountId` or a new response envelope. Classic
v5 already covers campaign management, reporting, and per-keyword suggested
bids. Platform API 1.0 exposes a broader set of Apple's opportunity signals as
dedicated query workflows. An operator can discover demand, estimate upside,
and decide where bids or budget have room to grow before spending more.

| Platform API 1.0 intelligence | Better campaign decision | Potential performance impact |
| --- | --- | --- |
| Keyword, phrase, and category suggestions | Expand from an app or seed terms into related App Store demand, phrases, and categories. | Find relevant reach without building every opportunity set manually. |
| Search-term popularity insights | Query weekly or monthly search-term rank and popularity by country and App Store genre. | Spot rising demand and localization opportunities earlier. |
| Target CPA suggestions plus target CPA and daily budget recommendations | Set a campaign starting point, then identify live campaigns where a different acquisition target or budget may unlock incremental installs. | Reduce avoidable underfunding and make scaling decisions with estimated outcomes. |
| Expanded impression-share insights | Compare first-slot and all-slot visibility by search term and market. | Separate genuine auction headroom from keywords with limited demand. |
| Change history | Trace campaign edits and field-level changes by time, entity, and event type. | Diagnose performance shifts faster and avoid repeating harmful changes. |
| Apple Maps brand and location workflows | Work with business brands, location groups, locations, eligibility, and dedicated brand reports. | Bring Maps campaign planning and measurement into the same automation stack. |
| Campaign groups, creatives, and bulk operations | Scale ad-account organization, new creative models, and correlated bulk changes. | Launch and reconcile larger programs with less manual object stitching. |

The strongest workflow combines these Apple opportunity signals with actual
post-install revenue. Apple can estimate installs, spend, and CPA; RevenueCat or
AppsFlyer can show whether the acquired users produce enough revenue and ROAS to
justify scaling.

Lily CLI 0.2.3 turns the supported opportunity signals into read-only commands while
retaining Platform campaign workflows, authenticated live reporting, and the
classic v5 provider. Querying an opportunity never applies or dismisses it.

Important contract distinction: Platform API 1.0 does not document a keyword
recommendations endpoint. Keyword, phrase, and category discovery belongs to
Suggestions. Recommendations covers target CPA and daily budget.

Read the full guide: [What Platform API 1.0 unlocks for campaign performance](https://www.chatlily.ai/guides/apple-ads-platform-api-1-0-new-features)

## Why This Exists

Most Apple Ads automation tools expose raw endpoint-shaped commands. Lily takes
a business-first approach:

- plan a campaign package before writing to Apple Ads
- create campaign, ad group, keywords, negative keywords, CPA goal, and creative
  attachment in one command
- return JSON by default so Codex, Claude Code, CI, and humans can review plans
  through business summaries and confirmation choices
- use Apple Ads Platform API 1.0 for campaign workflows and live reporting
- keep Campaign Management API v5 available during account migrations
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

Confirm that the installed CLI includes Platform API 1.0 support:

```sh
lily ads platform readiness
```

Run a live Platform API 1.0 campaign report after configuring local Apple Ads
credentials and an `APPLE_ADS_AD_ACCOUNT_ID`:

```sh
lily --provider platform ads reports campaigns \
  --app-id 999999999 \
  --from 2026-08-01 \
  --to 2026-08-14
```

Find keyword opportunities and compare them with demand signals before changing
a campaign:

```sh
lily --provider platform ads suggestions keywords \
  --app-id 999999999 --countries US --terms "flight booking"

lily --provider platform ads suggestions phrases \
  --query-type search --phrases "flight booking,cheap flights"

lily --provider platform ads recommendations target-cpa \
  --app-id 999999999 --state AVAILABLE

lily --provider platform ads insights search-term-popularity \
  --country US --genre Travel
```

For search-term popularity, `--genre Productivity` and `--genre Utilities` both
query Apple's combined `PRODUCTIVITY_UTILITIES` category. Canonical values and
full App Store paths such as `>Mobile Software Applications>Productivity` are
also accepted.

Ask Lily to recommend a review-only campaign draft when an agent or operator
has the business intent but not every bid, budget, or keyword ready yet:

```sh
lily ads plan recommend \
  --app-id 999999999 \
  --country UK
```

The recommendation response includes `assumptions`, `review`, `planned`, and
`confirmation` fields so AI agents can explain the suggested budget, bid, CPA
goal, keywords, and risks before asking the user to approve or modify.

Preview a fully specified business plan without writing to Apple Ads:

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
Writes require explicit budget, bid, and keywords.

The same business command can explicitly select Platform API v1:

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
export APPLE_ADS_CLIENT_ID=...
export APPLE_ADS_TEAM_ID=...
export APPLE_ADS_KEY_ID=...
export APPLE_ADS_AD_ACCOUNT_ID=...
export APPLE_ADS_PRIVATE_KEY_PATH=/path/to/AuthKey.p8
```

Platform API v1 is the default when no provider is configured. An explicit
provider in the config file or environment is preserved. Override it per command
with `--provider platform` (or `--provider v1`). The ad account ID must be verified;
the CLI never assumes it is the legacy organization ID.

For legacy v5 access:

```sh
export APPLE_ADS_PROVIDER=campaignv5
export APPLE_ADS_ORG_ID=...
```

Provider precedence: command flag → `APPLE_ADS_PROVIDER` → `LILY_ADS_PROVIDER`
→ config file → `platform`.

When creating a v1 plan with `--cpa-goal`, the CLI verifies the saved ad group
CPA Goal after creation. If v1 omits `cpaCap`, an explicitly configured
`APPLE_ADS_ORG_ID` permits a read-only v5 compatibility check. Missing values,
unavailable reads, wrong identities, or mismatched amounts/currencies return a
nonzero exit code with created IDs and `cpaGoalVerification` in JSON. Inspect
those resources before retrying; creation is not atomic. `--cpa-goal` keeps manual
CPT bidding and does not enable campaign Target CPA.

CNY Money inputs are normalized to RMB without changing amounts or names. Bulk
item failures also return a nonzero exit code and preserve the item results.

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

## Apple Ads Platform API 1.0 Support

The CLI hides API shape changes behind providers:

```txt
campaignv5 -> https://api.searchads.apple.com/api/v5
platform   -> https://api.ads.apple.com/v1
```

Lily 0.2.0 introduced Apple Ads Platform API 1.0 campaign workflows and
authenticated live campaign reporting. Lily 0.2.2 aligns requests with Apple's
official open-source Node SDK contract and completes the documented read-only
opportunity and insight queries. Lily 0.3.0 defaults to `platform`;
explicit `campaignv5` settings remain available for legacy compatibility.
See the [migration guide](docs/v1-default-migration.md) for existing installations.

Apple publishes official Node, Python, Java, and Swift clients, but not a Go
client. The CLI therefore keeps its small Go HTTP provider and treats
`@apple/apple-ads-platform` 1.117.0 as the generated contract source for v1
models, endpoint paths, enums, and regression tests. See the
[SDK contract review](docs/apple-sdk-contract.md) for the upgrade assessment.
Apple's announced retirement date for Campaign Management API v5 is
January 26, 2027.

Known compatibility differences handled by the providers:

- v5 context header uses `orgId`; Platform uses `adAccountId`
- v5 successful responses use `data`; Platform uses `result`
- v5 find/get-all patterns are replaced by Platform `/query` patterns
- v5 campaign payloads use `adamId`; Platform campaigns use promoted objects
- v5 keyword status uses `ACTIVE`; Platform keyword status uses `ENABLED`
- Platform bulk keyword creation uses `/keywords/bulk-create`

Read-only Platform opportunity commands:

```sh
lily --provider platform ads suggestions keywords --app-id 999999999 --countries US
lily --provider platform ads suggestions phrases --app-id 999999999
lily --provider platform ads suggestions phrases --query-type search --phrases "task manager,to do list"
lily --provider platform ads suggestions categories --app-id 999999999
lily --provider platform ads suggestions categories --query-type search --categories Productivity,Business
lily --provider platform ads suggestions target-cpa --app-id 999999999
lily --provider platform ads recommendations target-cpa --app-id 999999999
lily --provider platform ads recommendations daily-budget --app-id 999999999
lily --provider platform ads insights search-term-popularity --country US --genre Travel
lily --provider platform ads insights impression-share --app-id 999999999 --countries US
lily --provider platform ads change-history query --entity-types Campaign,Keyword
```

These commands only read Apple data. `lily ads recommendations apply` remains
reserved because applying or dismissing a recommendation changes campaign state.

## Commands

```sh
lily login --token <token>
lily logout
lily auth status
lily ads doctor
lily ads platform readiness
lily ads plan recommend [flags]
lily ads plan create [flags]
lily ads reports campaigns [flags]
lily ads revenue summary [flags]
lily --provider platform ads suggestions keywords [flags]
lily --provider platform ads suggestions phrases [flags]
lily --provider platform ads suggestions categories [flags]
lily --provider platform ads suggestions target-cpa [flags]
lily --provider platform ads recommendations target-cpa [flags]
lily --provider platform ads recommendations daily-budget [flags]
lily --provider platform ads insights search-term-popularity [flags]
lily --provider platform ads insights impression-share [flags]
lily --provider platform ads change-history query [flags]
lily --provider platform ads change-history detail --id <detailId>
```

Planned next commands:

```sh
lily ads reports keywords
lily ads reports search-terms
lily ads query campaigns
lily ads migrate plan
```

## License

MIT.

This project is an independent, unofficial tool and is not affiliated with,
endorsed by, or sponsored by Apple Inc. Apple, App Store, Apple Ads, and related
marks are trademarks of Apple Inc.
