package math_test

import (
	"math/big"
	"testing"

	"github.com/dora-network/bond-api-golang/math"

	"github.com/stretchr/testify/require"
)

func TestValidBigInt(t *testing.T) {
	tcs := []struct {
		title  string
		bigInt string
		expErr bool
	}{
		{
			"valid: normal number",
			"12300231",
			false,
		},
		{
			"valid: big number",
			"5867456423415648712346549872315489741231564897756423146878",
			false,
		},
		{
			"invalid: number with .",
			"12300231.3223",
			true,
		},
		{
			"invalid: number with ,",
			"12300231,3223",
			true,
		},
		{
			"invalid: number with letters",
			"12300231sdas",
			true,
		},
	}

	for _, tc := range tcs {
		t.Run(
			tc.title, func(t *testing.T) {
				_, err := math.ValidBigInt(tc.bigInt)
				if tc.expErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
			},
		)
	}
}

func TestExchangeRate(t *testing.T) {
	tcs := []struct {
		name        string
		amountIn    string
		amountOut   string
		exponentIn  int
		exponentOut int
		expResult   *big.Float
		expErr      error
	}{
		{
			" swap 152_000000 USDC for 1_000 BondA",
			"152000000",
			"1000",
			6,
			3,
			big.NewFloat(0.006578947368421052),
			nil,
		},
		{
			" swap 1_000000 BondA for 233_0000 USDC",
			"1000000",
			"2330000",
			6,
			4,
			big.NewFloat(233.0),
			nil,
		},
		{
			" swap 1_564723 BondA for 521_1231 USDC",
			"1564723",
			"5211231",
			6,
			4,
			big.NewFloat(333.044954282642998),
			nil,
		},
	}

	for _, tc := range tcs {
		t.Run(
			tc.name, func(t *testing.T) {
				result, actErr := math.ExchangeRate(tc.amountIn, tc.amountOut, tc.exponentIn, tc.exponentOut)
				if tc.expErr != nil {
					require.EqualError(t, actErr, tc.expErr.Error())
					return
				}
				require.NoError(t, actErr)
				require.Equal(t, tc.expResult.String(), result.String())
			},
		)
	}
}

func TestValueInUSD(t *testing.T) {
	tcs := []struct {
		name      string
		amount    *big.Int
		price     *big.Float
		expResult *big.Int
		expErr    error
	}{
		{
			"10000000 * 390.478",
			big.NewInt(10000000),
			big.NewFloat(390.478),
			big.NewInt(3904780000),
			nil,
		},
		{
			"1 * 1.17625",
			big.NewInt(1),
			big.NewFloat(1.17625),
			big.NewInt(1),
			nil,
		},
		{
			"13 * 1.17625",
			big.NewInt(13),
			big.NewFloat(1.17625),
			big.NewInt(15),
			nil,
		},
		{
			"1 * 0.99",
			big.NewInt(1),
			big.NewFloat(0.99),
			big.NewInt(0),
			nil,
		},
		{
			"1 * 1.99",
			big.NewInt(1),
			big.NewFloat(1.99),
			big.NewInt(1),
			nil,
		},
	}

	for _, tc := range tcs {
		t.Run(
			tc.name, func(t *testing.T) {
				result, actErr := math.ValueInUSD(tc.amount, tc.price)
				if tc.expErr != nil {
					require.EqualError(t, actErr, tc.expErr.Error())
					return
				}
				require.NoError(t, actErr)
				require.Equal(t, tc.expResult.String(), result.String())
			},
		)
	}
}

func TestInverseExchangeRate(t *testing.T) {
	tcs := []struct {
		name                string
		exchangeRate        *big.Float
		inverseExchangeRate *big.Float
	}{
		{
			"10ETH/BTC",
			big.NewFloat(10),
			big.NewFloat(0.1),
		},
		{
			"10.5ETH/BTC",
			big.NewFloat(10.5),
			big.NewFloat(0.09523809524),
		},
	}

	for _, tc := range tcs {
		t.Run(
			tc.name, func(t *testing.T) {
				actInverseExchangeRate := math.InverseExchangeRate(tc.exchangeRate)
				require.Equal(t, tc.inverseExchangeRate.String(), actInverseExchangeRate.String())
			},
		)
	}
}

func TestBigInt_Misc(t *testing.T) {
	one := new(big.Int).SetInt64(1)
	minusOne := new(big.Int).SetInt64(-1)
	zero := new(big.Int).SetInt64(0)

	// IsNegative
	require.False(t, math.IsNegative(one))
	require.False(t, math.IsNegative(zero))
	require.True(t, math.IsNegative(minusOne))

	// IsPositive
	require.True(t, math.IsPositive(one))
	require.False(t, math.IsPositive(zero))
	require.False(t, math.IsPositive(minusOne))

	// AllPositive
	require.True(t, math.AllPositive(one))
	require.False(t, math.AllPositive(zero))
	require.False(t, math.AllPositive(minusOne))
	require.True(t, math.AllPositive(one, one))
	require.False(t, math.AllPositive(one, zero))
	require.False(t, math.AllPositive(zero, minusOne))

	// IsZero
	require.False(t, math.IsZero(one))
	require.True(t, math.IsZero(zero))
	require.False(t, math.IsZero(minusOne))

	// LT
	require.False(t, math.LT(one, one))
	require.True(t, math.LT(minusOne, zero))
	require.False(t, math.LT(one, zero))

	// EQ
	require.True(t, math.EQ(one, one))
	require.False(t, math.EQ(minusOne, zero))
	require.False(t, math.EQ(one, zero))

	// GT
	require.False(t, math.GT(one, one))
	require.False(t, math.GT(minusOne, zero))
	require.True(t, math.GT(one, zero))

	// LTE
	require.True(t, math.LTE(one, one))
	require.True(t, math.LTE(minusOne, zero))
	require.False(t, math.LTE(one, zero))

	// GTE
	require.True(t, math.GTE(one, one))
	require.False(t, math.GTE(minusOne, zero))
	require.True(t, math.GTE(one, zero))
}

func TestValidBigFloats(t *testing.T) {
	validFloats := []string{
		"1.9",
		"-3.5",
		"154.0918230912830198232",
		"0.0",
	}

	invalidFloats := []string{
		"1.9",
		"-3.5",
		"invalid",
		"0.0",
	}

	_, err := math.ValidBigFloats(validFloats...)
	require.NoError(t, err)

	_, err = math.ValidBigFloats(invalidFloats...)
	require.Error(t, err)
}

func TestValidBigIntPercentage(t *testing.T) {
	tcs := []struct {
		title  string
		bigInt string
		expErr bool
	}{
		{
			"invalid bigInt",
			"invalidBigInt",
			true,
		},
		{
			"invalid: more than 100%",
			"10000000",
			true,
		},
		{
			"invalid: less than 0%",
			"-1",
			true,
		},
		{
			"valid",
			"1000000",
			false,
		},
	}

	for _, tc := range tcs {
		t.Run(
			tc.title, func(t *testing.T) {
				_, err := math.ValidBigIntPercentage(tc.bigInt)
				if tc.expErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
			},
		)
	}
}

func TestBigFloat_Misc(t *testing.T) {
	bF := new(big.Float).SetFloat64(12.66)
	f64, err := math.BigFloatToFloat64(bF)
	require.NoError(t, err)
	require.Equal(t, 12.66, f64)

	require.Equal(
		t,
		math.MulFloat64ByBigFloat(f64, new(big.Float).SetFloat64(33.918230)).String(),
		new(big.Float).SetFloat64(429.4047918).String(),
	)

	require.True(t, math.LTFloat(bF, new(big.Float).SetFloat64(13.00)))
}

func TestExchangeRate_ValidForBigInt(t *testing.T) {
	type args struct {
		amountIn    string
		amountOut   string
		exponentIn  int
		exponentOut int
	}
	tests := []struct {
		name string
		args args
		want *big.Float
	}{
		{
			name: "works for big.Int",
			args: args{
				amountIn:    big.NewInt(100).String(),
				amountOut:   big.NewInt(1000).String(),
				exponentIn:  6,
				exponentOut: 6,
			},
			want: big.NewFloat(10),
		},
		{
			name: "works for big.Int",
			args: args{
				amountIn:    big.NewInt(1000).String(),
				amountOut:   big.NewInt(100).String(),
				exponentIn:  6,
				exponentOut: 6,
			},
			want: big.NewFloat(.1),
		},
		{
			name: "works for big.Int",
			args: args{
				amountIn:    big.NewInt(204458972725).String(),
				amountOut:   big.NewInt(9223372036854775807).String(),
				exponentIn:  6,
				exponentOut: 6,
			},
			want: big.NewFloat(45111114.05),
		},
		{
			name: "works for big.Int",
			args: args{
				amountIn:    big.NewInt(9223372036854775807).String(),
				amountOut:   big.NewInt(204458972725).String(),
				exponentIn:  6,
				exponentOut: 6,
			},
			want: big.NewFloat(2.216748624e-08),
		},
		{
			name: "works for big.Int",
			args: args{
				amountIn:    "30893089680998375935793389036",
				amountOut:   "1570957057752780267839076398763967",
				exponentIn:  6,
				exponentOut: 6,
			},
			want: big.NewFloat(50851.40638),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := math.ExchangeRate(tt.args.amountIn, tt.args.amountOut, tt.args.exponentIn, tt.args.exponentOut)
			require.NoError(t, err)
			require.Equal(t, got.String(), tt.want.String(), "Want: %s got: %s", tt.want.String(), got.String())
		})
	}
}
