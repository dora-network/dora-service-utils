package types

import "github.com/goccy/go-json"

// Balances contains zero or more (AssetID string, Amount int64) key-value pairs.
type Balances struct {
	Bals map[string]int64 `json:"bals" redis:"bals" spanner:"bals"`
}

func (b *Balances) MarshalBinary() ([]byte, error) {
	return json.Marshal(b)
}

func (b *Balances) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, b)
}

// EmptyBalances returns an empty Balances.
func EmptyBalances() *Balances {
	return &Balances{
		Bals: make(map[string]int64),
	}
}

// NewBalances constructs Balances containing a given amount of a single asset.
// If asset ID is empty, returns an empty Balances instead.
func NewBalances(assetID string, amount int64) *Balances {
	return EmptyBalances().AddAmount(assetID, amount)
}
