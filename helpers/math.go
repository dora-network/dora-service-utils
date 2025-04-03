package helpers

// Divides n by 10^x then optionally rounds up
func applyDecimalsThenRound(n int64, x int, roundUp bool) int64 {
	m := exp10(x)
	result := n / m
	if result*m < n && roundUp {
		return result + 1 // rounded up
	}
	return result // n / 10^x was an exact integer, or roundUp was false
}

// Returns 10^x for positive x; 1 otherwise
func exp10(x int) int64 {
	result := int64(1)
	for i := 0; i < x; i++ {
		result *= 10
	}
	return result
}
