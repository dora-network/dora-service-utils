package math

import "math/big"

// ApproxExponential is the taylor series expansion of e^x centered around x=0, truncated
// to the cubic term. It can be used with great accuracy to determine e^x when x is very small.
// Note that e^x = 1 + x/1! + x^2/2! + x^3 / 3! + ...
func ApproxExponential(x *big.Float) *big.Float {
	sum := AddF(
		big.NewFloat(1),                      // 1
		x,                                    // + x / 1!
		DivF(MulF(x, x), big.NewFloat(2)),    // + x^2 / 2!
		DivF(MulF(x, x, x), big.NewFloat(6)), // + x^3 / 3!
	)
	return sum // approximated e^x
}
