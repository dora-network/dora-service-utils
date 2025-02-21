package types

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/dora-network/dora-service-utils/errors"
)

const (
	CouponPrefix   = "Coupon_"
	SnapshotPrefix = "Snapshot_"
)

// ValidAssetID checks that an asset ID contains only alphanumeric characters and underscores,
// as well as at most one hyphen somewhere in the middle, and is non-empty.
func ValidAssetID(id string) error {
	if strings.HasPrefix(id, CouponPrefix) || strings.HasPrefix(id, SnapshotPrefix) {
		// InterestSources entries do not start with coupon asset, they end with it.
		// Individual asset IDs must not start with this prefix either, to prevent confusion.
		// Same applies to total supply snapshots.
		return errors.Data("invalid asset ID: %s", id)
	}
	re := regexp.MustCompile("[^A-Za-z0-9_-]")
	trimmed := re.ReplaceAllLiteralString(id, "")
	if id == "" ||
		id != trimmed || // this checks whether the regexp removed any characters outside the accepted set
		strings.HasPrefix(id, "-") ||
		strings.HasSuffix(id, "-") ||
		strings.Count(id, "-") > 1 {
		return errors.Data("invalid asset ID: %s", id)
	}
	split := strings.Split(id, "-")
	if len(split) == 2 {
		// ID contained a hyphen. It is either a pool share or an InterestSources entry.
		s := split[1]
		if strings.HasPrefix(s, CouponPrefix) {
			// InterestSources entry rules
			period := strings.TrimPrefix(s, CouponPrefix)
			// Period must be a valid int64 (we don't care what it is, just that it is valid)
			if _, err := strconv.ParseInt(period, 10, 64); err != nil {
				return errors.Data("invalid asset ID: %s", id)
			}
		} else if strings.HasPrefix(s, SnapshotPrefix) {
			// TotalSupplySnapshot entry rules
			period := strings.TrimPrefix(s, SnapshotPrefix)
			// Period must be a valid int64 (we don't care what it is, just that it is valid)
			if _, err := strconv.ParseInt(period, 10, 64); err != nil {
				return errors.Data("invalid asset ID: %s", id)
			}
		} else {
			// Pool share rules
			if split[0] == split[1] {
				// No asset can be paired with itself
				return errors.Data("invalid asset ID: %s", id)
			}
		}
	}
	return nil
}
