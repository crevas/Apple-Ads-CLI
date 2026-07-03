# Changelog

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
