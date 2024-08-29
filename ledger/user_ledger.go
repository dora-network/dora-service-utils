package ledger

import (
	"context"
	"fmt"
	"github.com/cenkalti/backoff/v4"
	"github.com/dora-network/dora-service-utils/errors"
	"github.com/dora-network/dora-service-utils/redis"
	redisv9 "github.com/redis/go-redis/v9"
	"sort"
	"time"
)

type UserLedger struct {
	userID   string
	balances Balances
}

type Balances []*Balance

func NewUserLedger(userID string, x ...Balance) UserLedger {
	ul := UserLedger{
		userID:   userID,
		balances: Balances{},
	}
	for _, v := range x {
		if !v.IsZero() && v.UserID == ul.userID {
			ul.balances = append(ul.balances, &v)
		}
	}
	ul.balances.Sort()
	return ul
}

// UserID gets the ID of the user owning a UserLedger.
func (ul UserLedger) UserID() string {
	return ul.userID
}

// AssetIDs gets the IDs of assets present in a UserLedger.
func (ul UserLedger) AssetIDs() (ids []string) {
	for _, b := range ul.balances {
		ids = append(ids, b.AssetID)
	}
	return ids
}

// MustAssetIDs requires that the asset IDs in UserLedger match the input exactly (independent of order)
func (ul UserLedger) MustAssetIDs(ids ...string) error {
	if len(ul.AssetIDs()) != len(ids) {
		return errors.Data("MustAssetIDs: length of ul != %d", len(ids))
	}
	for _, id := range ids {
		if !ul.Has(id) {
			return errors.Data("MustAssetIDs: asset %s was not present in ul", id)
		}
	}
	return nil
}

func NewUserLedgerFromMap(userID string, bMap map[string]*Balance) UserLedger {
	ledger := UserLedger{
		userID:   userID,
		balances: Balances{},
	}
	for _, b := range bMap {
		ledger.balances = append(
			ledger.balances, b,
		)
	}
	ledger.balances.Sort()
	return ledger
}

// Sort balances by assetID
func (bals Balances) Sort() {
	sort.SliceStable(
		bals, func(i, j int) bool {
			return bals[i].AssetID < bals[j].AssetID
		},
	)
}

// Sanitize returns balances without zero amounts.
func (ul UserLedger) Sanitize() UserLedger {
	sanitized := UserLedger{}
	for _, b := range ul.balances {
		if !b.IsZero() {
			sanitized.balances = append(sanitized.balances, b)
		}
	}
	sanitized.balances.Sort()
	return sanitized
}

// Slice converts balances back to []*Balance
func (ul UserLedger) Slice() (s []*Balance) {
	s = append(s, ul.balances...)
	return s
}

// MapBals converts []*Balance to map[string]*Balance, including zero amounts
func (ul UserLedger) MapBals() map[string]*Balance {
	balMap := map[string]*Balance{}
	for _, b := range ul.balances {
		balMap[b.AssetID] = b
	}
	return balMap
}

// Select retrieves a single Balance from UserLedger. Balance is zero if asset is not in UserLedger.
func (ul UserLedger) Select(assetID string) *Balance {
	m := ul.MapBals()
	bal, ok := m[assetID]
	if !ok {
		bal = ZeroBalance(ul.userID, assetID)
	}
	return bal
}

// Has returns true if balances contains a Balance with a given assetID
func (ul UserLedger) Has(asset string) bool {
	m := ul.MapBals()
	_, ok := m[asset]
	return ok
}

func (ul UserLedger) String() string {
	s := ""
	ul.balances.Sort()
	for i, b := range ul.balances {
		if i > 0 {
			s = s + ", "
		}
		s = s + b.String()
	}
	return s
}

// Equal returns true if UserLedger are equal. Treats missing amounts and zero amounts as equal.
func (ul UserLedger) Equal(x UserLedger) bool {
	bMap := ul.MapBals() // may include zero values
	xMap := x.MapBals()  // may include zero values

	// Since maps maybe have different keys, this logic checks equality by indexing through both maps.
	// The shared elements end up being checked twice, but this isn't a problem.
	for id, xBal := range xMap {
		bBal, ok := bMap[id]
		if !ok {
			bBal = ZeroBalance(xBal.UserID, xBal.AssetID)
		}
		if !xBal.Equal(bBal) {
			// returns false if any b in x is not equal to bMap[b.AssetID]
			return false
		}
	}
	for id, bBal := range bMap {
		xBal, ok := xMap[id]
		if !ok {
			xBal = ZeroBalance(bBal.UserID, bBal.AssetID)
		}
		if !xBal.Equal(bBal) {
			// returns false if any b in x is not equal to bMap[b.AssetID]
			return false
		}
	}
	return true
}

// EnoughBalance returns true if Balance.Balance contains at least a required amount of specified assets
func (ul UserLedger) EnoughBalance(req ...Amount) bool {
	bMap := ul.MapBals()
	for _, r := range req {
		bal, ok := bMap[r.AssetID]
		if !ok {
			bal = ZeroBalance(ul.userID, r.AssetID)
		}
		if r.GT(bal.Balance) {
			// returns false if any balance r in req is greater than bMap[r.AssetID] Balance.Balance
			return false
		}
	}
	return true
}

// EnoughSupplied returns true if Balance.Supplied contains at least a required amount of specified assets
func (ul UserLedger) EnoughSupplied(req ...Amount) bool {
	bMap := ul.MapBals()
	for _, r := range req {
		bal, ok := bMap[r.AssetID]
		if !ok {
			bal = ZeroBalance(ul.userID, r.AssetID)
		}
		if r.GT(bal.Supplied) {
			// returns false if any balance r in req is greater than bMap[r.AssetID] Balance.Supplied
			return false
		}
	}
	return true
}

// EnoughBorrowed returns true if Balance.Borrowed contains at least a required amount of specified assets
func (ul UserLedger) EnoughBorrowed(req ...Amount) bool {
	bMap := ul.MapBals()
	for _, r := range req {
		bal, ok := bMap[r.AssetID]
		if !ok {
			bal = ZeroBalance(ul.userID, r.AssetID)
		}
		if r.GT(bal.Borrowed) {
			// returns false if any balance r in req is greater than bMap[r.AssetID] Balance.Supplied
			return false
		}
	}
	return true
}

// Add some Amount to UserLedger and returns the result.
func (ul UserLedger) Add(adds ...Amount) (UserLedger, error) {
	bMap := ul.MapBals()
	for _, a := range adds {
		b, ok := bMap[a.AssetID]
		if !ok {
			b = ZeroBalance(ul.userID, a.AssetID)
		}
		if err := b.Add(a); err != nil {
			return UserLedger{}, err
		}
		bMap[a.AssetID] = b.Copy()
	}

	return NewUserLedgerFromMap(ul.userID, bMap), nil
}

// Sub some Amount from UserLedger and returns the result.
func (ul UserLedger) Sub(subs ...Amount) (UserLedger, error) {
	if !ul.EnoughBalance(subs...) {
		return UserLedger{}, errors.Wrap(
			errors.InvalidInputError,
			errors.ErrInsufficientBalance,
			fmt.Sprintf("for Subs: %#v", subs),
		)
	}
	bMap := ul.MapBals()
	for _, s := range subs {
		b := bMap[s.AssetID]
		if err := b.Sub(s); err != nil {
			return UserLedger{}, err
		}
		bMap[s.AssetID] = b.Copy()
	}

	return NewUserLedgerFromMap(ul.userID, bMap), nil
}

// Lock some Amount from the UserLedger and returns the result.
func (ul UserLedger) Lock(locks ...Amount) (UserLedger, error) {
	if !ul.EnoughBalance(locks...) {
		return UserLedger{}, errors.Wrap(
			errors.InvalidInputError,
			errors.ErrInsufficientBalance,
			fmt.Sprintf("for Locks: %#v", locks),
		)
	}
	bMap := ul.MapBals()
	for _, l := range locks {
		b := bMap[l.AssetID]
		if err := b.Lock(l); err != nil {
			return UserLedger{}, err
		}
		bMap[l.AssetID] = b.Copy()
	}

	return NewUserLedgerFromMap(ul.userID, bMap), nil
}

// Unlock some Amount from the UserLedger and returns the result.
func (ul UserLedger) Unlock(unlocks ...Amount) (UserLedger, error) {
	bMap := ul.MapBals()
	for _, u := range unlocks {
		b, ok := bMap[u.AssetID]
		if !ok {
			return UserLedger{}, errors.Wrap(errors.InvalidInputError, errors.ErrInsufficientBalance, u.AssetID)
		}
		if err := b.Unlock(u); err != nil {
			return UserLedger{}, err
		}
		bMap[u.AssetID] = b.Copy()
	}

	return NewUserLedgerFromMap(ul.userID, bMap), nil
}

// Supply some Amount from the UserLedger balance to supplied, and returns the result.
func (ul UserLedger) Supply(supplies ...Amount) (UserLedger, error) {
	if !ul.EnoughBalance(supplies...) {
		return UserLedger{}, errors.Wrap(
			errors.InvalidInputError,
			errors.ErrInsufficientBalance,
			fmt.Sprintf("for Supply: %#v", supplies),
		)
	}
	bMap := ul.MapBals()
	for _, s := range supplies {
		b, ok := bMap[s.AssetID]
		if !ok {
			return UserLedger{}, errors.Wrap(errors.InvalidInputError, errors.ErrInsufficientBalance, s.AssetID)
		}
		if err := b.Supply(s); err != nil {
			return UserLedger{}, err
		}
		bMap[s.AssetID] = b.Copy()
	}

	return NewUserLedgerFromMap(ul.userID, bMap), nil
}

// Withdraw some Amount from the UserLedger supplied to balance, and returns the result.
func (ul UserLedger) Withdraw(withdrawals ...Amount) (UserLedger, error) {
	if !ul.EnoughSupplied(withdrawals...) {
		return UserLedger{}, errors.Wrap(
			errors.InvalidInputError,
			errors.ErrInsufficientBalance,
			fmt.Sprintf("for Supply: %#v", withdrawals),
		)
	}
	bMap := ul.MapBals()
	for _, w := range withdrawals {
		b, ok := bMap[w.AssetID]
		if !ok {
			return UserLedger{}, errors.Wrap(errors.InvalidInputError, errors.ErrInsufficientBalance, w.AssetID)
		}
		if err := b.Withdraw(w); err != nil {
			return UserLedger{}, err
		}
		bMap[w.AssetID] = b.Copy()
	}

	return NewUserLedgerFromMap(ul.userID, bMap), nil
}

// Borrow some Amount to the UserLedger borrowed, and returns the result.
func (ul UserLedger) Borrow(borrows ...Amount) (UserLedger, error) {
	bMap := ul.MapBals()
	for _, borrow := range borrows {
		b, ok := bMap[borrow.AssetID]
		if !ok {
			return UserLedger{}, errors.Wrap(errors.InvalidInputError, errors.ErrInsufficientBalance, borrow.AssetID)
		}
		if err := b.Borrow(borrow, false); err != nil {
			return UserLedger{}, err
		}
		bMap[borrow.AssetID] = b.Copy()
	}

	return NewUserLedgerFromMap(ul.userID, bMap), nil
}

// Repay some Amount from the UserLedger borrowed, and returns the result.
func (ul UserLedger) Repay(repays ...Amount) (UserLedger, error) {
	if !ul.EnoughBorrowed(repays...) {
		return UserLedger{}, errors.Wrap(
			errors.InvalidInputError,
			errors.ErrInsufficientBalance,
			fmt.Sprintf("for Supply: %#v", repays),
		)
	}
	bMap := ul.MapBals()
	for _, r := range repays {
		b, ok := bMap[r.AssetID]
		if !ok {
			return UserLedger{}, errors.Wrap(errors.InvalidInputError, errors.ErrInsufficientBalance, r.AssetID)
		}
		if err := b.Repay(r); err != nil {
			return UserLedger{}, err
		}
		bMap[r.AssetID] = b.Copy()
	}

	return NewUserLedgerFromMap(ul.userID, bMap), nil
}

func GetUserLedger(
	ctx context.Context,
	rdb redis.Client,
	timeout time.Duration,
	userIDs ...string,
) ([]UserLedger, error) {
	watch := make([]string, len(userIDs))
	for i, userID := range userIDs {
		watch[i] = UserBalanceKey(userID)
	}
	return getUserLedger(ctx, rdb, timeout, watch)
}

func getUserLedger(ctx context.Context, rdb redis.Client, timeout time.Duration, keys []string) (
	[]UserLedger,
	error,
) {
	var ledgers []UserLedger

	f := func(tx *redisv9.Tx) error {
		// This is just a simple read from Redis, but we're going to read it in a transaction to ensure
		// that if some other process is changing the data while we are attempting to read it, we're not
		// reading it with stale data.

		// we use the TxPipelined method to execute multiple commands in a single transaction
		// and collect the results, if any of the keys we are watching have been modified
		// since we started the transaction, the transaction will fail and we will retry
		cmd, err := tx.TxPipelined(
			ctx, func(pipe redisv9.Pipeliner) error {
				for _, key := range keys {
					pipe.HGetAll(ctx, key)
				}
				return nil
			},
		)

		for _, c := range cmd {
			res, err := c.(*redisv9.SliceCmd).Result()
			if err != nil {
				return err
			}

			var balances []Balance
			for _, v := range res {
				b := new(Balance)
				if err := b.UnmarshalBinary([]byte(v.(string))); err != nil {
					return err
				}
				balances = append(balances, *b)
			}
			if len(balances) > 0 {
				ledgers = append(ledgers, NewUserLedger(balances[0].UserID, balances...))
			} else {
				ledgers = append(ledgers, UserLedger{})
			}

		}

		return err
	}

	if err := redis.TryTransaction(
		ctx,
		rdb,
		f,
		backoff.NewExponentialBackOff(backoff.WithMaxElapsedTime(timeout)),
		keys...,
	); err != nil {
		return nil, err
	}

	return ledgers, nil
}
