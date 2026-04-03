package validation

import "regexp"

var (
	panRegex   = regexp.MustCompile(`^[A-Z]{5}[0-9]{4}[A-Z]$`)
	phoneRegex = regexp.MustCompile(`^[0-9]{10}$`)
)

func IsValidPAN(pan string) bool {
	return panRegex.MatchString(pan)
}

func IsValidPhone10(phone string) bool {
	return phoneRegex.MatchString(phone)
}
