package types

import "github.com/goccy/go-json"

type Price struct {
	AssetID string  `json:"asset_id"`
	Price   float64 `json:"price"`
}

func (p *Price) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}

func (p *Price) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, p)
}
