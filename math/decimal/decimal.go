package decimal

import (
	"github.com/govalues/decimal"
)

func EQ(a, b decimal.Decimal) bool {
	return a.Cmp(b) == 0
}

func GT(a, b decimal.Decimal) bool {
	return a.Cmp(b) > 0
}

func LT(a, b decimal.Decimal) bool {
	return a.Cmp(b) < 0
}

func GTE(a, b decimal.Decimal) bool {
	return a.Cmp(b) >= 0
}

func LTE(a, b decimal.Decimal) bool {
	return a.Cmp(b) <= 0
}
