# Security

Do not open public issues with credentials, tokens, private keys, Apple Ads
account identifiers, or customer data.

Report sensitive vulnerabilities privately to the repository maintainers.

The CLI reads Apple Ads private keys from your local machine and does not commit
or upload them. Keep `.p8`, `.pem`, `.key`, `.env`, logs, reports, and snapshots
out of git. The repository `.gitignore` blocks common local credential and
output paths.
