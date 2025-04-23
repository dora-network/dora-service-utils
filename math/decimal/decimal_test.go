package decimal_test

import (
	mdecimal "github.com/dora-network/dora-service-utils/math/decimal"
	"github.com/govalues/decimal"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDecimalComparisons(t *testing.T) {
	a := decimal.MustParse("1.00")
	b := decimal.MustParse("2.00")
	c := decimal.MustParse("1.00")

	t.Run(
		"EQ", func(t *testing.T) {
			assert.True(t, mdecimal.EQ(a, c))
			assert.False(t, mdecimal.EQ(a, b))
		},
	)

	t.Run(
		"GT", func(t *testing.T) {
			assert.True(t, mdecimal.GT(b, a))
			assert.False(t, mdecimal.GT(a, b))
		},
	)

	t.Run(
		"LT", func(t *testing.T) {
			assert.True(t, mdecimal.LT(a, b))
			assert.False(t, mdecimal.LT(b, a))
		},
	)

	t.Run(
		"GTE", func(t *testing.T) {
			assert.True(t, mdecimal.GTE(b, a))
			assert.True(t, mdecimal.GTE(a, c))
			assert.False(t, mdecimal.GTE(a, b))
		},
	)

	t.Run(
		"LTE", func(t *testing.T) {
			assert.True(t, mdecimal.LTE(a, b))
			assert.True(t, mdecimal.LTE(a, c))
			assert.False(t, mdecimal.LTE(b, a))
		},
	)
}
