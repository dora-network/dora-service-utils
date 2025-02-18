package types

import (
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"
)

func TestBalanceJSON(t *testing.T) {
	require := require.New(t)

	bals := &Balances{}
	bals0 := NewBalances("bondA", 0)
	bals1 := NewBalances("bondA", -100)
	bals2 := NewBalances("pool-share", 100).AddAmount("bond_B_001", 42)

	b, _ := json.Marshal(bals0)
	require.Equal("{}", string(b))
	require.NoError(json.Unmarshal(b, &bals))
	require.Equal(
		&Balances{
			Bals: map[string]int64{},
		}, bals,
	)

	b, _ = json.Marshal(bals1)
	require.Equal("{\"bondA\":-100}", string(b))
	require.NoError(json.Unmarshal(b, &bals))
	require.Equal(bals1, bals)

	b, _ = json.Marshal(bals2)
	require.Equal("{\"bond_B_001\":42,\"pool-share\":100}", string(b))
	require.NoError(json.Unmarshal(b, &bals))
	require.Equal(bals2, bals)
}
