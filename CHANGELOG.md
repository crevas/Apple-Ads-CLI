# Changelog

## 0.1.5

- Adds business-facing `review` and `confirmation` objects to campaign plan
  dry-runs so AI agents can show confirm, modify, and cancel choices instead of
  raw shell instructions.
- Adds a Codex skill for safe Apple Ads CLI by Lily workflows.
- Updates agent guidance to keep CLI commands internal unless the user asks for
  technical details.

## 0.1.4

- Adds Apple Ads account binding context to Lily Ads Revenue Analytics requests
  so revenue enrichment cannot be mixed with a different local Apple Ads account.
- Treats Lily Ads Revenue Analytics account mismatches as a structured
  `account_mismatch` status.
- Ensures `lily login` and `lily logout` do not persist Apple Ads credential
  values that were only provided through environment variables.

## 0.1.3

- Fixes Apple Ads campaign report requests by enabling row totals whenever
  grand totals are requested.
- Keeps the reserved Platform API report payload aligned with the same row and
  grand total behavior.

## 0.1.2

- Uses `APPLE_ADS_*` as the preferred local Apple Ads credential environment
  variable prefix.
- Keeps `LILY_ADS_*` aliases for compatibility while making doctor/help output
  avoid Lily-branded names for Apple credentials.

## 0.1.1

- Clarifies that Lily login is optional and only enables Lily Ads Revenue
  Analytics enrichment.
- Updates `lily auth status` and `lily ads doctor` so AI agents distinguish
  local Apple Ads API credentials from optional Lily revenue login.
- Keeps Apple Ads private key setup local and explicit in structured next steps.

## 0.1.0

- Initial open-source Apple Ads CLI by Lily.
- Adds `lily ads plan create` with dry-run JSON plans by default.
- Supports Campaign Management API v5 as the default provider.
- Adds a Platform API provider behind `--provider platform`.
- Adds Lily Ads Revenue Analytics login/revenue command integration.
- Reserves target CPA suggestion and recommendation command surfaces.
