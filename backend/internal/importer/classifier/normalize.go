package classifier

import (
	"strings"

	"golang.org/x/text/unicode/norm"
)

func NormalizeText(value string) string {
	normalized := norm.NFKC.String(value)
	normalized = strings.ToLower(normalized)
	return strings.Join(strings.Fields(normalized), " ")
}
