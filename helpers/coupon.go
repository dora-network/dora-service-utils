package helpers

// Copy of Coupon from graphtypes
type Coupon struct {
	// Date of the payment. Must be RFC1123 format. Example: Mon, 02 Jan 2006 15:04:05 MST.
	Date string `json:"date" graphql:"date"`
	// Date of the coupon period's start. Leave empty for an instant payment.
	Start string `json:"start" graphql:"start"`
	// Dollars to pay out per unit of bond. For example, a coupon payment of 0.03 represents 3% if 1.0 bonds = $1.
	Yield float64 `json:"yield" graphql:"yield"`
	// Whether this payment is not the asset's coupon but rather its final maturation.
	// Freezes asset after this date, and ensures payment is not marked as "Interest" in transactions.
	IsMaturity bool `json:"isMaturity" graphql:"is_maturity"`
}
