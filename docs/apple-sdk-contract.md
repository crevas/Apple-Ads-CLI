# Apple Ads SDK contract review

Reviewed on November 3, 2026: `@apple/apple-ads-platform` **1.109.0 → 1.117.0**.
Version 1.117.0 was the npm `latest` release at review time.

This Go CLI uses the generated Apple SDK as a reference for its Platform API
contract; it does not install or execute the Node SDK. The reviewed version is
stored in `internal/providers/platform/apple_sdk_contract_version.txt`, embedded
in the binary for `lily ads platform readiness`, and checked against npm by the
scheduled `Apple Ads SDK Contract` workflow.

## Sources and method

- [Apple's official Node SDK repository](https://github.com/apple/apple-ads-platform-api-node)
- [Published 1.109.0 package](https://registry.npmjs.org/@apple/apple-ads-platform/-/apple-ads-platform-1.109.0.tgz)
- [Published 1.117.0 package](https://registry.npmjs.org/@apple/apple-ads-platform/-/apple-ads-platform-1.117.0.tgz)

Both package archives were verified against npm's published SHA-512 integrity
values. The review compared all generated TypeScript model declarations and
API declarations, plus JavaScript routes and model enums, then checked the
differences against the Go Platform provider and response handling.

## Changes and CLI impact

| SDK change | Impact on existing CLI commands |
| --- | --- |
| Adds `/keywords/bulk-delete`, `/negative-keywords/bulk-delete`, and seven related models. | The CLI does not expose these operations. Existing routes remain present. |
| Removes `storefront` from `AppsReportingRequest.groupBy`. | Campaign report requests do not send `groupBy`. |
| Changes reporting keyword status from `ACTIVE` to `ENABLED`. | Platform keyword creation already defaults to `ENABLED`; responses use generic JSON maps. |
| Removes expected performance fields from target CPA recommendations and history. | Recommendation responses are passed through as JSON; the CLI does not require these fields. |
| Removes expected impression/tap fields and forecast ranges from daily budget recommendations/history; adds optional `adAccountId`. | Recommendation responses are passed through without a fixed response schema. |
| Adds optional `ErrorDetail.info`. | Error bodies are retained as JSON maps; existing message extraction is unchanged. |
| Removes `LocationGroup.query` and updates rule documentation. | The CLI does not expose location-group operations. |
| Clarifies hourly metric date formatting. | No request change; response dates are not parsed into a fixed format. |

The existing creation, suggestion, recommendation-query, insight, and
change-history request models remain compatible. No new request fields are
required for existing CLI commands.

## Validation scope

Validation covers the existing Go test suite, formatting, `go vet`, compilation,
the local Platform readiness output, and the scheduled workflow's npm version
comparison. No authenticated Apple API calls or campaign mutations are needed
for this contract update.

Preserve the previously verified live-API workarounds: campaign reports request
only `GRAND_TOTAL`, and search-term popularity omits `sorting`. The relevant SDK
models have not changed, so this review does not establish that the live
endpoint limitations have been removed.
