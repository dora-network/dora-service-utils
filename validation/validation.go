package validation

import (
	"strings"

	"github.com/dora-network/dora-service-utils/errors"
	"github.com/govalues/decimal"
)

func ValidateDecimalIsInt(value interface{}) error {
	_, err := decimalValueIsInt(value)
	return err
}

func decimalValueIsInt(value interface{}) (decimal.Decimal, error) {
	v, ok := value.(decimal.Decimal)
	if !ok {
		return v, errors.ErrInvalidInput
	}
	if !v.IsInt() {
		return v, errors.ErrValueMustBeExpressedAsInteger
	}
	return v, nil
}

func ValidateDecimalIsPositiveInt(value interface{}) error {
	v, err := decimalValueIsInt(value)
	if err != nil {
		return err
	}
	if v.IsNeg() || v.IsZero() {
		return errors.ErrAmountMustBePositive
	}
	return nil
}

func ValidatePositiveDecimal(value interface{}) error {
	v, ok := value.(decimal.Decimal)
	if !ok {
		return errors.ErrInvalidInput
	}
	if v.IsNeg() || v.IsZero() {
		return errors.ErrAmountMustBePositive
	}
	return nil
}

func HasHyphen(value interface{}) error {
	v, ok := value.(string)
	if !ok {
		return errors.ErrInvalidInput
	}
	if strings.Contains(v, "-") {
		return errors.ErrAssetIDMustNotContainHyphen
	}
	return nil
}
