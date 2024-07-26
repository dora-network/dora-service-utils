package orderbook

import "fmt"

func ID(baseID, quoteID string) string {
	return fmt.Sprintf("%s-%s", baseID, quoteID)
}
