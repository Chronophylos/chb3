package util

import "math"

// Divmod returns quotient and remainder of a and b.
func Divmod(a, b float64) (float64, int) {
	return math.Floor(a / b), int(math.Mod(a, b))
}
