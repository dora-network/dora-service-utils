package errors

import (
	"errors"
	"fmt"
)

// ErrorType represents the type of error.
type ErrorType string

const (
	// NotFoundError indicates a not found error.
	NotFoundError ErrorType = "NotFound"
	// InvalidInputError indicates an invalid input error.
	InvalidInputError ErrorType = "InvalidInput"
	// InternalError indicates an internal error.
	InternalError ErrorType = "Internal"
	// ErrAccessDenied indicates an access denied error.
	ErrAccessDenied ErrorType = "AccessDenied"
	// InvalidDataErr indicates a data validation error.
	InvalidDataErr ErrorType = "DataInvalid"
	// RateLimitErr indicates a rate limit error.
	RateLimitErr ErrorType = "RateLimit"
	// DeprecatedErr indicates the feature is deprecated.
	DeprecatedErr ErrorType = "Deprecated"
)

var (
	ErrNotImplemented      = New(InternalError, "not implemented")
	ErrUserAccessDenied    = New(ErrAccessDenied, "user access denied")
	ErrApiKeyNotFound      = New(NotFoundError, "failed to get api key")
	ErrAssetNotFound       = New(NotFoundError, "failed to fetch asset")
	ErrBaseUIDNotInformed  = New(NotFoundError, "baseUID isn't informed")
	ErrQuoteUIDNotInformed = New(NotFoundError, "quoteUID isn't informed")
	ErrInvalidTickSize     = New(NotFoundError, "tickSize must be greater than zero")
	ErrOrderBookExists     = New(NotFoundError, "orderBook already exists")
	ErrInvalidAssets       = New(NotFoundError, "invalid assets for orderBook")
	// ErrNotEnoughAmountIn error for when the swap amount in is not enough.
	ErrNotEnoughAmountIn = New(InvalidInputError, "amount in is not enough")
	// ErrNotEnoughAmountOut error for when the swap amount out is not enough.
	ErrNotEnoughAmountOut = New(InvalidInputError, "amount out is not enough")
	// ErrPoolShouldHave2Assets error for when the pool doesn't have 2 assets.
	ErrPoolShouldHave2Assets = New(InvalidInputError, "pool should have 2 assets")
	// ErrAddLiquidityHasMaxOf2Assets error for when the user tries to add liq product pool with more than 2 assets.
	ErrAddLiquidityHasMaxOf2Assets = New(InvalidInputError, "product pool has max of 2 assets")
	// ErrTimeFrom error when timeFrom should be less than timeTo.
	ErrTimeFrom = New(InvalidInputError, "timeFrom should be less than timeTo")
	// ErrAssetNotFoundInPool error for when the asset is not found in the pool assets.
	ErrAssetNotFoundInPool = New(NotFoundError, "asset not found in pool")
	// ErrNotEnoughLP error for when the pool does not give out enough LP shares.
	ErrNotEnoughLP = New(InvalidInputError, "new LP shares created are lower than min LP share needed")
	// ErrAssetBondMissingFields error for when the asset is a bond type but didn't fill all the fields.
	ErrAssetBondMissingFields   = New(InvalidInputError, "asset is a bond, but it is missing fields")
	ErrBigIntNotValidPercentage = New(
		InternalError,
		"value is not a valid BigInt percentage 0 ~ 1000000",
	)
	ErrInvalidTradeType                   = New(InvalidInputError, "invalid trade type")
	ErrPoolUIDOrOrderBookIDMustBeInformed = New(InvalidInputError, "poolUID and/or orderBookID must be informed")
	ErrAssetIDMustBeInformed              = New(InvalidInputError, "assetID must be informed")
	ErrAmountMustBePositive               = New(InvalidInputError, "the amount should be positive")
	ErrFromMustBeSmallerThanTo            = New(InvalidInputError, "from must be smaller than to")
	ErrResolutionHigherThanToMinusFrom    = New(InvalidInputError, "resolution higher than to minus from")
	ErrNoValidSwapPath                    = New(InternalError, "no swap path found")
	ErrSwapInputInNil                     = New(InternalError, "AssetIn and MinAmtOut cannot be nil")

	ErrVolumeNotFound    = New(NotFoundError, "volume not found")
	ErrIsInNotFound      = New(NotFoundError, "bond isin not found")
	ErrInvalidInput      = New(InvalidInputError, "invalid input")
	ErrBorrowLimit       = New(InvalidInputError, "user would be above their borrow limit")
	ErrInvalidPoolType   = New(InvalidInputError, "invalid pool type")
	ErrInvalidLimitPrice = New(InvalidInputError, "invalid limit price")

	ErrLiquidationIneligible = New(InvalidInputError, "user not eligible for liquidation")
	ErrLiquidationLenZero    = New(InvalidInputError, "liquidation-eligible user has zero collateral (or zero borrows)")

	ErrPriceMustBePositive                 = New(InvalidInputError, "price must be positive")
	ErrPriceMustBeGTEThanOrderBookTickSize = New(
		InvalidInputError,
		"price must be equal or greater than orderBook tickSize",
	)
	ErrInsufficientBalance = New(InvalidInputError, "insufficient balance")

	ErrReadNotAllowed  = New(InternalError, "txctx has reads disabled")
	ErrWriteNotAllowed = New(InternalError, "txctx has writes disabled")

	ErrAssetIDMustNotContainHyphen   = New(InvalidInputError, "assetID must not contain hyphen")
	ErrValueMustBeExpressedAsInteger = Data("value must be expressed as an integer")

	ErrCannotChangeOrderType = Data("order type cannot be changed")

	ErrInvalidInAssetSell       = Data("Sell orders must have assetIn = Base")
	ErrInvalidOutAssetSell      = Data("Sell order must have assetOut = Quote")
	ErrOrderBookIDAssetMismatch = Data("orderbook ID did not match assets")
	ErrInvalidInAssetBuy        = Data("Buy orders must have assetIn = Quote")
	ErrInvalidOutAssetBuy       = Data("Buy orders must have assetOut = Base")

	ErrOrderMissingUserID             = Data("order request missing user id")
	ErrOrderRequestValidationFailed   = Data("order request validation failed")
	ErrOrderAmendCannotChangeAssets   = Data("order amend cannot change assets")
	ErrOrderNotFound                  = Data("order not found")
	ErrOrderAmendCannotChangeLeverage = Data("order amend cannot change leverage")
	ErrOrderContainsInvalidOrderType  = Data("order contains invalid order type")

	ErrPoolAssetsMismatch = Data("pools asset mismatch")

	ErrInvalidOrderType = Data("invalid order type")
)

// TypedError represents an error with a specific type.
type TypedError struct {
	Type ErrorType
	Err  error
}

// Is returns true if the err is a *TypesError and its Type is the one specified
func Is(err error, typ ErrorType) bool {
	e, ok := err.(*TypedError)
	if ok {
		return e.Type == typ
	}
	return false
}

// Error implements the error interface for TypedError.
func (e *TypedError) Error() string {
	return e.Err.Error()
}

// Unwrap returns the underlying error.
func (e *TypedError) Unwrap() error {
	return e.Err
}

// New creates a new TypedError with the given error type and message.
func New(errorType ErrorType, message string) *TypedError {
	return &TypedError{Type: errorType, Err: errors.New(message)}
}

// Newf creates a new TypedError with the given error type and message.
func Newf(errorType ErrorType, message string, a ...any) *TypedError {
	return &TypedError{Type: errorType, Err: fmt.Errorf(message, a...)}
}

// NewInternal creates a new internal error with the given message.
func NewInternal(message string) *TypedError {
	return &TypedError{Type: InternalError, Err: errors.New(message)}
}

// NewRateLimit creates a new rate limit error with the given message.
func NewRateLimit(message string) *TypedError {
	return &TypedError{Type: RateLimitErr, Err: errors.New("Rate Limit: " + message)}
}

// NewAssetNotFound creates a new not found error with the given asset ID.
func NewAssetNotFound(assetID string) *TypedError {
	return &TypedError{Type: NotFoundError, Err: fmt.Errorf("asset %s not found", assetID)}
}

// NewNotEnoughAmountOut creates an error with insufficient swap amount out.
// Requires two Balance.String() inputs.
func NewNotEnoughAmountOut(wantBalance, gotBalance string) *TypedError {
	return &TypedError{
		Type: InvalidInputError,
		Err:  fmt.Errorf("amount out is not enough. want: %s, got: %s", wantBalance, gotBalance),
	}
}

// Wrap creates a new TypedError by wrapping an existing error with an additional message.
func Wrap(errorType ErrorType, err error, message string) *TypedError {
	return &TypedError{Type: errorType, Err: fmt.Errorf("%s: %w", message, err)}
}

// Data creates a new invalid data error
func Data(message string, a ...any) *TypedError {
	return &TypedError{Type: InvalidDataErr, Err: fmt.Errorf(message, a...)}
}
