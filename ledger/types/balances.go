package types

import "github.com/goccy/go-json"

// Balances contains zero or more (AssetID string, Amount int64) key-value pairs.
type Balances struct {
	Bals map[string]int64 `json:"bals" redis:"bals"`
}

func (b *Balances) MarshalBinary() ([]byte, error) {
	return json.Marshal(b)
}

func (b *Balances) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, b)
}

// Empty returns an empty Balances.
func Empty() *Balances {
	return &Balances{
		Bals: make(map[string]int64),
	}
}

// New constructs Balances containing a given amount of a single asset.
// If asset ID is empty, returns an empty Balances instead.
func New(assetID string, amount int64) *Balances {
	b := &Balances{}
	return b.AddAmount(assetID, amount)
}
