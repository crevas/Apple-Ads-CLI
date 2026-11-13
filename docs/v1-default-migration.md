# Platform API v1 default and Web compatibility audit

Reviewed November 13, 2026 for Lily 0.3.0. Upgrade from 0.2.3 to receive the
v1 default and compatibility fixes described below.

## Existing installations

With no provider configured, the CLI now selects `platform` (Apple Ads v1).
Provider precedence is command flag, `APPLE_ADS_PROVIDER`, `LILY_ADS_PROVIDER`,
config file, then default. An existing explicit `campaignv5` setting remains
effective. To switch it, update that setting or pass `--provider platform`.

Configure `APPLE_ADS_AD_ACCOUNT_ID` with the verified v1 account ID. The CLI does
not infer it from `APPLE_ADS_ORG_ID`, since those identifiers can differ. Missing
v1 scope fails before Apple requests and explains the migration requirement.
`lily ads doctor` checks local configuration; `lily ads platform readiness`
describes build capabilities and is not an authenticated connectivity test.

```sh
export APPLE_ADS_AD_ACCOUNT_ID=123456789
lily ads doctor
lily ads reports campaigns --app-id 987654321 \
  --from 2026-11-01 --to 2026-11-02 --no-revenue

# Explicit legacy access requires APPLE_ADS_ORG_ID:
lily --provider campaignv5 ads reports campaigns --app-id 987654321 \
  --from 2026-11-01 --to 2026-11-02 --no-revenue
```

## Comparison with Web runtime fixes

| Web issue / compatibility requirement | CLI finding and outcome |
| --- | --- |
| Default v1 with explicit v5 override | Changed default, empty-provider normalization, provider construction, help and diagnostics; preserved file/environment/flag overrides. |
| Campaign requires `MANUAL_CPT` / `TAP` | Already correct in CLI; retained and covered through the full HTTP creation flow. CPA Goal remains an ad group setting. |
| Supply source and placement mapping | Added explicit `APPSTORE` source and translated `APPSTORE_PRODUCT_PAGES_BROWSE` to v1 `APPSTORE_PRODUCT_PAGES`; search results, search tab and today tab stay distinct. Legacy v5 accepts the reverse alias. |
| CNY / RMB Money compatibility | Added to campaign budgets, ad group bids, CPA Goal and keyword bids for both providers and dry-run reviews. Amounts, names and keyword text are preserved; no currency conversion. Normalization does not mutate the caller's plan. |
| v1 omits persisted CPA Goal on read | Added post-creation v1 GET verification, optional v5 GET for an omitted field, identity validation and exact amount/currency comparison. Missing, malformed or mismatched values fail verification. |
| Campaign-level negative queries must exclude ad group negatives | CLI currently only creates negatives and already scopes these separately. No negative query command exists to port the Web query fix into; both creation scopes are tested. |
| Country/placement filters on campaign queries | `ads query campaigns` is only planned, not implemented. CLI does not send those unsupported query filters. |
| Campaign report filters use promoted object ID | Already correct: CLI sends `promotedObjectId`, uses app campaign reports, and preserves explicit body offset/page size. Verified through HTTP tests. |
| Missing URL pagination must not become zero | CLI builds native v1 request bodies; no Web URL-to-body adapter exists. Nonzero offset/page-size preservation is tested. |
| Stale chat failures / proposal blockers | Web chat execution cache and proposal state do not exist in this CLI. No equivalent patch applies. |

The audit also found a CLI-specific creation failure: resource ID extraction
attempted to type-assert a named Go response map to an unnamed map, always
missing IDs. Fixed for v1 `result`, v5 `data`, and top-level IDs, including full
numeric IDs. HTTP integration tests now create a complete campaign package.

Platform bulk responses can contain item failures despite HTTP success. The CLI
now checks every correlated result, preserves the response and created IDs, and
returns a nonzero exit code for failed or incomplete batches.

## CPA Goal and recovery behavior

`--cpa-goal` sends `cpaCap.value` with manual CPT bidding. After creating the
package, the CLI reads the ad group and compares actual saved values. Only when
v1 omits `cpaCap` may it issue one v5 GET using an explicitly configured
`APPLE_ADS_ORG_ID`. An explicit null, mismatched identity, failed v1 GET or
malformed value does not trigger fallback. No writes are retried on v5.

`cpaGoalVerification` reports `verified`, `unverified`, or `mismatch`, the read
source (`v1` or `v5-compatibility`), and valid actual Money when present. CNY/RMB
are equivalent for comparison; other currencies and different amounts fail.
The requested value is never substituted for a missing response value.

If the compatibility endpoint is unavailable or no legacy org is configured,
verification can fail after resources were created. The CLI emits JSON with
`error`, `executed` resource IDs and recovery guidance, and exits nonzero. Inspect
the existing resources before retrying. This is not an atomic campaign package
operation, and it does not automatically delete or recreate partial results.
Plans without `--cpa-goal` make no CPA compatibility reads. Explicit v5 plans
retain their existing behavior and do not claim the new v1 CPA verification.

## Validation and limits

Validation uses local HTTP fixtures with the real CLI command parser, OAuth
token flow, headers, serialization and response decoder, plus provider and
configuration tests. It covers default v1, explicit v5, CNY Money, request
strategies, negative scopes, paging, successful and mismatched CPA readback,
missing/error responses, large IDs and partial bulk failures.

Passed: 84 tests including subtests under `go test -race ./...`, `go vet ./...`,
`go build -o bin/lily ./cmd/lily`, formatting and `git diff --check`. The compiled
binary also passed isolated readiness, default-v1 dry-run and explicit-v5
dry-run checks. No local CLI credentials were available
for an authenticated Apple smoke test during this review. No live campaign
writes were performed during pre-release validation.

## Contract references

- [Apple CampaignCreate](https://developer.apple.com/documentation/apple-ads-platform-api/campaigncreate)
- [Apple's generated SDK contract](https://github.com/apple/apple-ads-platform-api-node)
- [Pinned SDK contract review](apple-sdk-contract.md)

The published 1.117.0 SDK archive was verified against npm's SHA-512 integrity.
Its declarations were checked for `AdGroup.cpaCap`,
`CPAGoal.value`, bulk response result arrays and per-item success/correlation
fields; the seven relevant declarations are identical to the locally installed
Web SDK 1.109.0. The Web runtime fixes supplement these generated contracts where a
successful write may omit deprecated fields on readback.
