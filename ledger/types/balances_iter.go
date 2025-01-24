package types

// Iterate over each (assetID,amount) in balances
// Stops iteration and returns on first error
func (b *Balances) Iterate(f func(id string, amt int64) error) error {
	for k, v := range b.Bals {
		err := f(k, v)
		if err != nil {
			return err
		}
	}
	return nil
}

// IterateIDs over each (assetID,amount) in balances which matches a slice of asset IDs
// Stops iteration and returns on first error
func (b *Balances) IterateIDs(ids []string, f func(id string, amt int64) error) error {
	for _, id := range ids {
		err := f(id, b.Bals[id])
		if err != nil {
			return err
		}
	}
	return nil
}
