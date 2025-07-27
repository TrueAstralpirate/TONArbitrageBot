package utils

import (
	"strconv"
	"strings"
)

func ParseReserve(s string, decimals int) (float64, error) {
	if len(s) < decimals {
		s = strings.Repeat("0", decimals-len(s)) + s
	}
	if len(s) == decimals {
		return strconv.ParseFloat("0."+s, 64)
	}
	return strconv.ParseFloat(s[:len(s)-decimals]+"."+s[len(s)-decimals:], 64)
}
