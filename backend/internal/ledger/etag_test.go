package ledger

import "testing"

func TestTask503ALedgerETagRoundTripAndStrictIfMatchParsing(t *testing.T) {
	etag := FormatLedgerETag("ledger-a", 7)
	if etag != `"ledger:ledger-a:v7"` {
		t.Fatalf("unexpected etag %q", etag)
	}

	version, err := ParseLedgerIfMatch(etag, "ledger-a")
	if err != nil || version != 7 {
		t.Fatalf("parse etag: version=%d err=%v", version, err)
	}

	for _, value := range []string{
		"",
		"ledger:ledger-a:v7",
		`W/"ledger:ledger-a:v7"`,
		`"ledger:ledger-b:v7"`,
		`"ledger:ledger-a:v0"`,
		`"ledger:ledger-a:vx"`,
		`"ledger:ledger-a:v7", "ledger:ledger-a:v8"`,
	} {
		if _, err := ParseLedgerIfMatch(value, "ledger-a"); err == nil {
			t.Fatalf("expected invalid If-Match %q", value)
		}
	}
}
