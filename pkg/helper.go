package pkg

import (
	"regexp"
	"strings"
)

var spaces = regexp.MustCompile("[[:space:][:punct:]]+")
var illegalChars = regexp.MustCompile("[^[:alnum:]\\p{L}-]")

func sanitizeFilename(file string) string {
	file = spaces.ReplaceAllString(file, "-")
	file = strings.Trim(file , "-")
	return illegalChars.ReplaceAllString(file, "")
}
