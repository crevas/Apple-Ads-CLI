package appleads

import "strings"

// NormalizeCurrency follows Lily's Money convention; no exchange conversion.
func NormalizeCurrency(currency string) string {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if currency == "CNY" {
		return "RMB"
	}
	return currency
}
