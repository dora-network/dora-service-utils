package orderbook

import "fmt"

func ID(baseID, quoteID string) string {
	return fmt.Sprintf("%s-%s", baseID, quoteID)
}

type Side string

const (
	Buy  Side = "buy"
	Sell Side = "sell"
)
