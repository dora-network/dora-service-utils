package orderbook

import (
	"fmt"
	"strings"
)

func ID(baseID, quoteID string) string {
	return fmt.Sprintf("%s-%s", baseID, quoteID)
}

type Side string

const (
	Buy  Side = "buy"
	Sell Side = "sell"
)

func GetBaseFromOrderBookID(orderBookID string) string {
	assets := strings.Split(orderBookID, "-")
	if len(assets) != 2 {
		return ""
	}
	return assets[0]
}

func GetQuoteFromOrderBookID(orderBookID string) string {
	assets := strings.Split(orderBookID, "-")
	if len(assets) != 2 {
		return ""
	}
	return assets[1]
}
