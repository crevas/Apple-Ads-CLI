---
name: apple-ads-cli-by-lily
description: Use Apple Ads CLI by Lily to plan, review, report, and optimize Apple Ads or Apple Search Ads campaigns with AI-friendly JSON workflows. Use when the user asks Claude Code to manage Apple Ads campaigns, create campaign packages, review keywords, set CPA goals, add negative keywords, choose creatives or product pages, inspect Platform API readiness, run campaign reports, or enrich decisions with Lily Ads Revenue Analytics revenue/ROAS.
---

# Apple Ads CLI by Lily

## Overview

Use `lily` as a business-first Apple Ads CLI. Keep Apple Ads operations tied to local Apple credentials, use Lily login only for optional revenue enrichment, and present write operations as user-facing approval choices rather than raw shell instructions.

## Core Rules

- Start with `lily ads doctor` before calling Apple Ads APIs.
- Use local `APPLE_ADS_*` credentials for Apple Ads campaign, ad group, keyword, creative, and report operations.
- Treat `lily login` as optional. It only enables Lily Ads Revenue Analytics revenue, paid-user, profit, and ROAS enrichment.
- Never say Apple Ads operations require Lily login.
- Never ask the user to upload `.p8` keys to Lily. Private keys stay local.
- If revenue returns `login_required`, `dashboard_required`, or `account_mismatch`, keep the Apple Ads result and skip ROAS.
- Do not calculate ROAS when the Apple Ads account and Lily revenue account do not match.
- Default to dry-run JSON plans. Execute writes only after explicit approval.
- When the user asks to create a campaign with incomplete details, first analyze available app/report context and produce a recommended draft. Do not reply with a checklist of missing budget, bid, CPA, or keyword parameters unless app/account identity is ambiguous or a value is truly unsafe to infer.

## User-Facing Communication

Speak to the user in campaign/business language, not terminal language.

For plan reviews, show:

- campaign name, app, country or region, budget, status, placement
- ad group bid and CPA goal
- keyword counts by match type
- negative keyword counts by level
- creative or product page choice
- specific risks or items needing confirmation

Use Claude Code's native user confirmation or permission flow when available. If no native choice UI is available, ask the user to choose one of:

- Confirm and create
- Modify plan
- Cancel

Do not show commands such as `bash ...`, `run.sh`, or copied shell blocks to non-technical users unless they ask for implementation details. Keep CLI commands as agent-internal execution details.

## Workflows

### Diagnose Setup

Run:

```bash
lily ads doctor
lily auth status
```

Explain the result as:

- Apple Ads local credentials: ready or missing
- Lily Ads Revenue Analytics: connected or optional
- Platform API readiness: ready, reserved, or blocked

### Plan Campaign Package

Use `lily ads plan recommend` for business-level draft recommendations when the user gives intent such as "create a UK campaign" but not every setting. Include inferred CPA goal, negative keywords, and creative/product page choices when you can justify them from context; otherwise show the CLI `assumptions` and ask the user to approve or modify.

Use `lily ads plan create` only after the user accepts the recommendation or provides a fully specified package. Include CPA goal, negative keywords, and creative/product page choices when provided.

Default behavior is JSON dry-run. Review the returned `assumptions`, `review`, `planned`, and `confirmation` fields.

Before execution:

- summarize the business plan
- call out budget, status, CPA, keyword, negative keyword, and creative risks
- ask for confirmation through Claude Code's native confirmation flow when possible
- execute only after the user approves

### Reports And Revenue

Use `lily ads reports campaigns` for Apple Ads reports. Revenue enrichment may be included when Lily login and account binding are valid.

Interpret sources explicitly:

- `appleAds.source`: local Apple Ads API
- `revenue.source`: Lily Ads Revenue Analytics
- `roas`: only valid when the same Apple Ads account is bound on both sides

If revenue is unavailable, say the ad data was retrieved but paid-user, profit, and ROAS were skipped.

### Platform API

Use `lily ads platform readiness` before Platform API work. Platform API v1 commands may be reserved until Apple exposes endpoint contracts. Do not promise execution for reserved endpoints; say Lily is ready to support them when available.
