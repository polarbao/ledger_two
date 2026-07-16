package ledger

import (
	"fmt"
	"regexp"
	"strconv"
)

func FormatLedgerETag(ledgerID string, version int64) string {
	return fmt.Sprintf(`"ledger:%s:v%d"`, ledgerID, version)
}

func ParseLedgerIfMatch(value, ledgerID string) (int64, error) {
	pattern := regexp.MustCompile(`^"ledger:` + regexp.QuoteMeta(ledgerID) + `:v([1-9][0-9]*)"$`)
	matches := pattern.FindStringSubmatch(value)
	if len(matches) != 2 {
		return 0, fmt.Errorf("invalid ledger If-Match")
	}

	version, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil || version < 1 {
		return 0, fmt.Errorf("invalid ledger If-Match version")
	}
	return version, nil
}
