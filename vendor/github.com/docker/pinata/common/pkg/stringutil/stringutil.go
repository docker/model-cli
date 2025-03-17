package stringutil

import "strings"

// Contains returns true if string contains at least one value of an array
func Contains(s string, a []string) bool {
	for _, e := range a {
		if strings.Contains(s, e) {
			return true
		}
	}
	return false
}

// HasPrefix returns true if string contains at least one prefix of an array
func HasPrefix(s string, a []string) bool {
	for _, e := range a {
		if strings.HasPrefix(s, e) {
			return true
		}
	}
	return false
}

// HasPrefixIgnoringCase returns true if string contains at least one prefix of an array ignoring the case
func HasPrefixIgnoringCase(s string, a []string) bool {
	sLen := len(s)
	for _, e := range a {
		eLen := len(e)
		if sLen >= eLen && strings.EqualFold(s[:len(e)], e) {
			return true
		}
	}
	return false
}
