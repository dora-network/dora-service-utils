package types

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// MarshalJSON Error on invalid input. Omits zero-valued assets. Also sorts assetIDs in output alphabetically.
func (b *Balances) MarshalJSON() ([]byte, error) {
	if err := b.Validate(true); err != nil {
		return []byte{}, err
	}
	prefix := "{"
	contents := ""
	suffix := "}"
	assetIDs := []string{}
	stringMap := map[string]string{}
	for assetID, amount := range b.Bals {
		if amount == 0 || assetID == "" {
			continue // skip zero amounts
		}
		// Add to unsorted output
		assetIDs = append(assetIDs, assetID)
		stringMap[assetID] = displayAmount(amount)
	}
	// Sort assets by ID
	sort.Slice(assetIDs, func(i, j int) bool { return assetIDs[i] < assetIDs[j] })
	// Create output
	for i, assetID := range assetIDs {
		// Uses json number format, not string. Amount will not be quoted. Example: "USDY":12.13
		contents += fmt.Sprintf("\"%s\":%s", assetID, stringMap[assetID])
		if i+1 < len(assetIDs) {
			contents += "," // comma after all elements but the last
		}
	}
	return []byte(prefix + contents + suffix), nil
}

// displayAmount displays an amount as a string.
func displayAmount(amount int64) string {
	return fmt.Sprintf("%d", amount)
}

// UnmarshalJSON into Balances. Overwrites any values originally in b, even on error.
func (b *Balances) UnmarshalJSON(data []byte) error {
	b.Bals = map[string]int64{}

	// Expected format (whitespace added for clarity):
	// {
	//	 "USDY": 1234,
	//   "Bond-A": -5678,
	//   "some_thing_001": 9
	// }

	// Formatting should match JSON of a float map. Error if it doesn't.
	floatMap := map[string]float64{}
	err := json.Unmarshal(data, &floatMap)
	if err != nil {
		return err
	}

	// Keep only the following characters:
	// - Alphanumeric
	// - :,_-
	// This also trims whitespace, doublequotes, and curly braces in the expected JSON
	// so commas and colons are the only things left separating the data
	re := regexp.MustCompile("[^A-Za-z0-9:,_-]")
	s := re.ReplaceAllLiteralString(string(data), "")

	// Expected format (newlines added for clarity):
	// USDY:1234,
	// Bond-A:-5678,
	// some_thing_001:9

	if s != "" {
		lines := strings.Split(s, ",")
		for _, line := range lines {
			split := strings.Split(line, ":")
			if len(split) != 2 {
				return fmt.Errorf("line \"%s\" is not split by single colon", line)
			}
			// Find and validate asset ID
			assetID := split[0]
			if err = ValidAssetID(assetID); err != nil {
				return err
			}
			// Reject duplicate balances
			if _, ok := b.Bals[assetID]; ok {
				return fmt.Errorf("duplicate balance during unmarshal: %s", assetID)
			}
			// Process numerical amount
			amountString := split[1]
			amount, err := strconv.ParseInt(amountString, 10, 64) // positive or negative integer
			if err != nil {
				return err
			}
			// Add to balances
			b.Bals[assetID] = amount
		}
	}

	// Validate according to our own rules
	return b.Validate(true)
}
