package math

import "math/big"

// AddFS adds a float to a string containing a float, and returns the result as a string. Error if invalid inputs.
func AddFS(a string, b *big.Float) (string, error) {
	x, err := ValidBigFloat(a)
	if err != nil {
		return "", err
	}
	return AddF(x, b).String(), nil
}
