package math

import (
	"math/big"
)

/*
	This file is designed to simplify math/big syntax.

	Before:
		a.Add(a,b)
		c = big.NewInt(0).Add(a,b)
		d = a.Add(b).Add(c)

	After:
		a = math.Add(a,b)
		c = math.Add(a,b)
		d = math.Add(a,b,c)

	Should be much more intuitive without the z-receiver-overwrite=redundant-return pattern.
	Performance consequences negligible for our purposes.
*/

// todo: basic tests

// Float from big.Int
func Float(i *big.Int) *big.Float {
	return big.NewFloat(0).SetInt(i)
}

// Int from big.Float, rounded toward zero
func Int(f *big.Float) *big.Int {
	i := big.NewInt(0)
	f.Int(i)
	return i
}

// Add any amount of big.Ints together
func Add(ints ...*big.Int) *big.Int {
	sum := big.NewInt(0)
	for _, n := range ints {
		sum.Add(sum, n)
	}
	return sum
}

// Sub any amount of big.Ints from an initial value
func Sub(i *big.Int, ints ...*big.Int) *big.Int {
	diff := big.NewInt(0)
	diff.Add(diff, i)
	for _, n := range ints {
		diff.Sub(diff, n)
	}
	return diff
}

// Mul any amount of big.Ints together
func Mul(ints ...*big.Int) *big.Int {
	product := big.NewInt(1)
	for _, n := range ints {
		product.Mul(product, n)
	}
	return product
}

// MulIF multiplies a big.int by a big.Float and converts the result back to big.int.
func MulIF(i *big.Int, f *big.Float) *big.Int {
	iFloat := big.NewFloat(0).SetInt(i)
	outFloat := MulF(iFloat, f)
	output, _ := outFloat.Int(big.NewInt(0))
	return output
}

// MulIF64 multiplies a big.int by a float64 and converts the result back to big.int.
func MulIF64(i *big.Int, f64 float64) *big.Int {
	f := big.NewFloat(f64)
	return MulIF(i, f)
}

// MulI64 multiplies a big.int by an int64
func MulI64(i *big.Int, i64 int64) *big.Int {
	return Mul(i, big.NewInt(i64))
}

// Div divides an initial big.Int by any amount of big.Ints, and returns a big.Float (not another Int)
// note that Div(a,b,c,d) = a / (b*c*d)
func Div(i *big.Int, ints ...*big.Int) *big.Float {
	quo := big.NewFloat(0)
	quo.Add(quo, Float(i))
	for _, n := range ints {
		quo.Quo(quo, Float(n))
	}
	return quo
}

// AddF any amount of big.Floats together
func AddF(floats ...*big.Float) *big.Float {
	sum := big.NewFloat(0)
	for _, n := range floats {
		sum.Add(sum, n)
	}
	return sum
}

// SubF any amount of big.Floats from an initial value
func SubF(f *big.Float, floats ...*big.Float) *big.Float {
	diff := big.NewFloat(0)
	diff.Add(diff, f)
	for _, n := range floats {
		diff.Sub(diff, n)
	}
	return diff
}

// MulF any amount of big.Floats together
func MulF(floats ...*big.Float) *big.Float {
	product := big.NewFloat(1)
	for _, n := range floats {
		product.Mul(product, n)
	}
	return product
}

// DivF divides an initial big.Float by any amount of big.Floats
// note that DivF(a,b,c,d) = a / (b*c*d)
func DivF(f *big.Float, floats ...*big.Float) *big.Float {
	quo := big.NewFloat(0)
	quo.Add(quo, f)
	for _, n := range floats {
		quo.Quo(quo, n)
	}
	return quo
}

// Sqrt of a big.Float. If float is negative, makes it positive first.
func Sqrt(f *big.Float) *big.Float {
	root := big.NewFloat(1)
	if f.Sign() == -1 {
		f = MulF(f, big.NewFloat(-1))
	}
	return root.Sqrt(f)
}

// MinF64 gets the minimum of two float64
func MinF64(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// MaxF64 gets the maximum of two float64
func MaxF64(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// DivI divides two big.Ints a/b and returns the result as an int, rounded down by default.
// Should only be used for positive inputs.
func DivI(a, b *big.Int, roundUp bool) *big.Int {
	quo := big.NewInt(1).Div(a, b) // a/b rounded down
	mod := big.NewInt(0).Mod(a, b) // remainder of a/b
	if roundUp && mod.Sign() == 1 {
		// If there was a remainder and we want to round up, add 1
		quo = Add(quo, big.NewInt(1))
	}
	return quo
}
