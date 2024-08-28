package math

import (
	"errors"
	"math/bits"
)

var (
	ErrOverflowAdd = errors.New("integer overflow in addition")
	ErrOverflowMul = errors.New("integer overflow in multiplication")
	ErrOverflowSub = errors.New("integer overflow in subtraction")
	ErrDivByZero   = errors.New("divide by zero")
)

// CheckedAddU64 adds two uint64's together, returning an error in the event of an overflow.
func CheckedAddU64(a, b uint64) (uint64, error) {
	sum, carryOut := bits.Add64(a, b, 0)
	if carryOut == 1 {
		return 0, ErrOverflowAdd
	}
	return sum, nil
}

// CheckedMulU64 multiplies two uint64's together, returning an error in the event
// of an overflow.
func CheckedMulU64(a, b uint64) (uint64, error) {
	hi, lo := bits.Mul64(a, b)
	if hi > 0 {
		return 0, ErrOverflowMul
	}
	return lo, nil
}

// CheckedSubU64 computes `a - b` for two uint64's, returning an error in the event
// that is smaller than b
func CheckedSubU64(a, b uint64) (uint64, error) {
	result, borrow := bits.Sub64(a, b, 0)
	if borrow == 1 {
		return 0, ErrOverflowSub
	}
	return result, nil
}

// CheckedDivU64 computes `a / b` for two uint64's, returning an error in the event
// that b is 0
func CheckedDivU64(a, b uint64) (uint64, error) {
	if b == 0 {
		return 0, ErrDivByZero
	}
	result, _ := bits.Div64(0, a, b)
	return result, nil
}
