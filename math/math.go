package math

import (
	"fmt"
	"math"
	"math/big"
	"strconv"

	"github.com/dora-network/dora-service-utils/errors"
)

var (
	// BigInt100Percent defines the 100% to have precisions.
	BigInt100Percent = big.NewInt(1000000)

	// Exponents represents all the values to multiply by when normalizing two assets exponents.
	Exponents = map[int]*big.Float{
		-18: big.NewFloat(0.000000000000000001),
		-17: big.NewFloat(0.00000000000000001),
		-16: big.NewFloat(0.0000000000000001),
		-15: big.NewFloat(0.000000000000001),
		-14: big.NewFloat(0.00000000000001),
		-13: big.NewFloat(0.0000000000001),
		-12: big.NewFloat(0.000000000001),
		-11: big.NewFloat(0.00000000001),
		-10: big.NewFloat(0.0000000001),
		-9:  big.NewFloat(0.000000001),
		-8:  big.NewFloat(0.00000001),
		-7:  big.NewFloat(0.0000001),
		-6:  big.NewFloat(0.000001),
		-5:  big.NewFloat(0.00001),
		-4:  big.NewFloat(0.0001),
		-3:  big.NewFloat(0.001),
		-2:  big.NewFloat(0.01),
		-1:  big.NewFloat(0.1),
		0:   big.NewFloat(1.0),
		1:   big.NewFloat(10.0),
		2:   big.NewFloat(100.0),
		3:   big.NewFloat(1000.0),
		4:   big.NewFloat(10000.0),
		5:   big.NewFloat(100000.0),
		6:   big.NewFloat(1000000.0),
		7:   big.NewFloat(10000000.0),
		8:   big.NewFloat(100000000.0),
		9:   big.NewFloat(1000000000.0),
		10:  big.NewFloat(10000000000.0),
		11:  big.NewFloat(100000000000.0),
		12:  big.NewFloat(1000000000000.0),
		13:  big.NewFloat(10000000000000.0),
		14:  big.NewFloat(100000000000000.0),
		15:  big.NewFloat(1000000000000000.0),
		16:  big.NewFloat(10000000000000000.0),
		17:  big.NewFloat(100000000000000000.0),
		18:  big.NewFloat(1000000000000000000.0),
	}
)

const (
	Base10                = 10
	DefaultFloatPrecision = 8
)

func IsNegative(n *big.Int) bool {
	return n != nil && n.Sign() == -1
}

func IsPositive(n *big.Int) bool {
	return n != nil && n.Sign() > 0
}

func AllPositive(ints ...*big.Int) bool {
	for _, n := range ints {
		if n == nil || n.Sign() <= 0 {
			return false
		}
	}
	return true
}

func IsZero(n *big.Int) bool {
	return n != nil && n.Sign() == 0
}

func LT(x, y *big.Int) bool {
	return x != nil && y != nil && x.Cmp(y) == -1
}

func EQ(x, y *big.Int) bool {
	return x != nil && y != nil && x.Cmp(y) == 0
}

func NotEQ(x, y *big.Int) bool {
	return !EQ(x, y)
}

func GT(x, y *big.Int) bool {
	return x != nil && y != nil && x.Cmp(y) == 1
}

func LTE(x, y *big.Int) bool {
	return EQ(x, y) || LT(x, y)
}

func GTE(x, y *big.Int) bool {
	return EQ(x, y) || GT(x, y)
}

func IsFloatPositive(n *big.Float) bool {
	return n != nil && n.Sign() > 0
}

func IsFloatZero(n *big.Float) bool {
	return n != nil && n.Sign() == 0
}

func ZeroBigInt() *big.Int {
	return big.NewInt(0)
}

// ValidBigInts validates if the values are all valid big int.
func ValidBigInts(values ...string) (bigValues []*big.Int, err error) {
	bigValues = make([]*big.Int, len(values))
	for i, v := range values {
		bigValue, err := ValidBigInt(v)
		if err != nil {
			return nil, err
		}

		bigValues[i] = bigValue
	}
	return bigValues, nil
}

// ValidNotNegativeBigInt validates if the value is a valid and not negative big int.
// Valid values: [0-∞].
func ValidNotNegativeBigInt(value string) (v *big.Int, err error) {
	v, err = ValidBigInt(value)
	if err != nil {
		return nil, err
	}
	if IsNegative(v) {
		return nil, errors.Data("%s is negative", value)
	}
	return v, nil
}

// ValidPositiveBigInt validates if the value is a valid and positive big int.
// Valid values: [1-∞].
func ValidPositiveBigInt(value string) (v *big.Int, err error) {
	v, err = ValidBigInt(value)
	if err != nil {
		return nil, err
	}
	if !IsPositive(v) {
		return nil, errors.Data("%s is not positive", value)
	}
	return v, nil
}

// ValidPositiveBigInts validates if the values are a valid and positive big ints.
// Valid values: [1-∞].
func ValidPositiveBigInts(values ...string) (bigValues []*big.Int, err error) {
	bigValues = make([]*big.Int, len(values))
	for i, v := range values {
		bigValue, err := ValidPositiveBigInt(v)
		if err != nil {
			return nil, err
		}

		bigValues[i] = bigValue
	}
	return bigValues, nil
}

// ValidPositiveFloat64 validates if the value is a valid and positive float64.
// Valid values: (0-∞).
func ValidPositiveFloat64(value float64) error {
	if math.IsNaN(value) {
		return errors.Data("float64 was NaN")
	}
	if math.IsInf(value, 0) {
		return errors.Data("float64 was infinite")
	}
	if value <= 0 {
		return errors.Data("float64 %f was not positive", value)
	}
	return nil
}

// ValidPositiveFloat64s validates if the values are valid and positive float64s.
// Valid values: (0-∞).
func ValidPositiveFloat64s(values ...float64) error {
	for _, v := range values {
		if err := ValidPositiveFloat64(v); err != nil {
			return err
		}
	}
	return nil
}

// ValidBigInt validates if the value is a valid big int.
func ValidBigInt(value string) (v *big.Int, err error) {
	v, ok := new(big.Int).SetString(value, Base10)
	if !ok {
		return nil, errors.Data("%s is not a valid big.Int", value)
	}
	return v, nil
}

// ValidBigFloats validates if the values are all valid big float.
func ValidBigFloats(values ...string) (bigValues []*big.Float, err error) {
	bigValues = make([]*big.Float, len(values))
	for i, v := range values {
		bigValue, err := ValidBigFloat(v)
		if err != nil {
			return nil, err
		}

		bigValues[i] = bigValue
	}
	return bigValues, nil
}

// ValidBigFloat validates if the value is a valid big float.
func ValidBigFloat(value string) (v *big.Float, err error) {
	v, ok := new(big.Float).SetString(value)
	if !ok {
		return nil, errors.Data("%s is not a valid big.Float", value)
	}
	return v, nil
}

// ValidPositiveBigFloat validates if the value is a valid and positive big float.
// Valid values: (0-∞].
func ValidPositiveBigFloat(value string) (v *big.Float, err error) {
	v, err = ValidBigFloat(value)
	if err != nil {
		return nil, err
	}
	if !IsFloatPositive(v) {
		return nil, errors.Data("%s is not positive", value)
	}
	return v, nil
}

// ValidBigIntPercentage checks if the big int is valid and is between 0 ~ 1_000000.
func ValidBigIntPercentage(value string) (v *big.Int, err error) {
	v, err = ValidBigInt(value)
	if err != nil {
		return nil, err
	}

	if GT(v, BigInt100Percent) {
		return nil, errors.ErrBigIntNotValidPercentage
	}

	if IsNegative(v) {
		return nil, errors.ErrBigIntNotValidPercentage
	}

	return v, nil
}

func BigFloatToFloat64(f *big.Float) (float64, error) {
	value, err := strconv.ParseFloat(f.String(), 64)
	if err != nil {
		return 0, err
	}

	return value, err
}

func MulFloat64ByBigFloat(a float64, b *big.Float) *big.Float {
	return new(big.Float).Mul(new(big.Float).SetFloat64(a), b)
}

// ExchangeRate calculates it based on 2 asset amounts and exponents.
// Once we have swap info with amountIn and amountOut, we want to calculate the exchangeRate that was used for the
// swap, as well as, normalize the exponent if it's different between the assets.
// amountIn * exchangeRate = amountOut
// exchangeRate = amountOut / amountIn
//
// Example: a swap between USDC with 6 decimals and BondA with 4 decimals and price 152 USDC per BondA.
// From the tx we will receive amountIn: 152_000000 and amountOut: 1_0000.
// We need to calculate the exchange rate as well as normalize the exponent, since if we just do 1000 / 152000000,
// we will get 0,000006578947368 when the right response is 1/152 = 0,006578947368421.
// So we calculate the decimals diff (fromExponent - toExponent) -> 6 - 4 = 2.
// This means we need to multiply 0,000006578947368 by 10^2 -> 100.
// Besides, we created a map with all possible differences to not calculating 10^n every time we need to get the
// exchange rate.
func ExchangeRate(amountIn, amountOut string, exponentIn, exponentOut int) (*big.Float, error) {
	inAmtFloat, err := ValidBigFloat(amountIn)
	if err != nil {
		return nil, err
	}

	outAmtFloat, err := ValidBigFloat(amountOut)
	if err != nil {
		return nil, err
	}

	rate := new(big.Float).Quo(outAmtFloat, inAmtFloat)
	exponentFactor, err := ExponentFactor(exponentIn, exponentOut)
	if err != nil {
		return nil, err
	}

	return new(big.Float).Mul(rate, exponentFactor), nil
}

// InverseExchangeRate receives the exchange rate and inverts it from the perspective of the other quoted asset.
// Ex.: If exchange rate was 10 as 10ETH/BTC, the inverse was going to be 0.1 as 0.1BTC/ETH.
func InverseExchangeRate(exchangeRate *big.Float) *big.Float {
	return new(big.Float).Quo(new(big.Float).SetFloat64(1.0), exchangeRate)
}

// ExponentFactor calculates the factor to multiply by which, the assets with different exponents. Returns:
func ExponentFactor(fromExponent, toExponent int) (*big.Float, error) {
	exponentDiff := fromExponent - toExponent
	exponent, ok := Exponents[exponentDiff]
	if !ok {
		return nil, fmt.Errorf("multiplier not found for exponentDiff %d", exponentDiff)
	}
	return exponent, nil
}

// ValueInUSD given a specific amount, price.
func ValueInUSD(amount *big.Int, assetPrice *big.Float) (*big.Int, error) {
	value, _ := new(big.Float).Mul(new(big.Float).SetInt(amount), assetPrice).Int(nil)
	return value, nil
}

// LTFloat returns true if x < y, false otherwise.
func LTFloat(x, y *big.Float) bool {
	return x.Cmp(y) == -1
}

// Min returns the smaller of a or b
func Min(a, b *big.Int) *big.Int {
	if LT(a, b) {
		return a
	}
	return b
}
