package importer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type expectedPreview struct {
	SourceType   string        `json:"source_type"`
	ExpectedRows []expectedRow `json:"expected_rows"`
}

type expectedRow struct {
	RowNumber                int          `json:"row_number"`
	OccurredAt               string       `json:"occurred_at"`
	Title                    string       `json:"title"`
	Merchant                 string       `json:"merchant"`
	AmountCents              int64        `json:"amount_cents"`
	Direction                string       `json:"direction"`
	TargetTransactionType    string       `json:"target_transaction_type"`
	DuplicateStatus          string       `json:"duplicate_status"`
	RowStatus                string       `json:"row_status"`
	SuspiciousReasonContains string       `json:"suspicious_reason_contains"`
	Error                    *expectedErr `json:"error"`
}

type expectedErr struct {
	Code            string `json:"code"`
	MessageContains string `json:"message_contains"`
}

func TestParseCSVFixtures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		sourceType string
		csvFile    string
		expected   string
	}{
		{name: "wechat", sourceType: SourceTypeWechat, csvFile: "wechat-basic.csv", expected: "wechat-basic.preview.json"},
		{name: "alipay", sourceType: SourceTypeAlipay, csvFile: "alipay-basic.csv", expected: "alipay-basic.preview.json"},
		{name: "generic", sourceType: SourceTypeGeneric, csvFile: "generic-basic.csv", expected: "generic-basic.preview.json"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fixtureRoot := filepath.Join("..", "..", "..", "docs", "fixtures", "imports")
			csvFile, err := os.Open(filepath.Join(fixtureRoot, tt.csvFile))
			if err != nil {
				t.Fatalf("open fixture csv: %v", err)
			}
			defer csvFile.Close()

			got, err := ParseCSV(tt.sourceType, csvFile)
			if err != nil {
				t.Fatalf("ParseCSV returned error: %v", err)
			}

			want := readExpectedPreview(t, filepath.Join(fixtureRoot, "expected", tt.expected))
			if got.SourceType != want.SourceType {
				t.Fatalf("source type mismatch: got %q want %q", got.SourceType, want.SourceType)
			}
			if len(got.Rows) != len(want.ExpectedRows) {
				t.Fatalf("row count mismatch: got %d want %d", len(got.Rows), len(want.ExpectedRows))
			}

			for i := range want.ExpectedRows {
				assertPreviewRow(t, got.Rows[i], want.ExpectedRows[i])
			}
		})
	}
}

func readExpectedPreview(t *testing.T, path string) expectedPreview {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read expected preview: %v", err)
	}

	var expected expectedPreview
	if err := json.Unmarshal(data, &expected); err != nil {
		t.Fatalf("unmarshal expected preview: %v", err)
	}
	return expected
}

func assertPreviewRow(t *testing.T, got PreviewRow, want expectedRow) {
	t.Helper()

	if got.RowNumber != want.RowNumber {
		t.Fatalf("row_number mismatch: got %d want %d", got.RowNumber, want.RowNumber)
	}
	if want.OccurredAt != "" && got.OccurredAt != want.OccurredAt {
		t.Fatalf("row %d occurred_at mismatch: got %q want %q", want.RowNumber, got.OccurredAt, want.OccurredAt)
	}
	if got.Title != want.Title {
		t.Fatalf("row %d title mismatch: got %q want %q", want.RowNumber, got.Title, want.Title)
	}
	if got.Merchant != want.Merchant {
		t.Fatalf("row %d merchant mismatch: got %q want %q", want.RowNumber, got.Merchant, want.Merchant)
	}
	if got.AmountCents != want.AmountCents {
		t.Fatalf("row %d amount_cents mismatch: got %d want %d", want.RowNumber, got.AmountCents, want.AmountCents)
	}
	if got.Direction != want.Direction {
		t.Fatalf("row %d direction mismatch: got %q want %q", want.RowNumber, got.Direction, want.Direction)
	}
	if got.TargetTransactionType != want.TargetTransactionType {
		t.Fatalf("row %d target_transaction_type mismatch: got %q want %q", want.RowNumber, got.TargetTransactionType, want.TargetTransactionType)
	}
	if got.DuplicateStatus != want.DuplicateStatus {
		t.Fatalf("row %d duplicate_status mismatch: got %q want %q", want.RowNumber, got.DuplicateStatus, want.DuplicateStatus)
	}
	if got.RowStatus != want.RowStatus {
		t.Fatalf("row %d row_status mismatch: got %q want %q", want.RowNumber, got.RowStatus, want.RowStatus)
	}
	if want.SuspiciousReasonContains != "" && !strings.Contains(got.SuspiciousReason, want.SuspiciousReasonContains) {
		t.Fatalf("row %d suspicious reason mismatch: got %q want contains %q", want.RowNumber, got.SuspiciousReason, want.SuspiciousReasonContains)
	}
	if want.Error == nil {
		if got.Error != nil {
			t.Fatalf("row %d unexpected error: %+v", want.RowNumber, got.Error)
		}
		return
	}
	if got.Error == nil {
		t.Fatalf("row %d expected error %+v, got nil", want.RowNumber, want.Error)
	}
	if got.Error.Code != want.Error.Code {
		t.Fatalf("row %d error code mismatch: got %q want %q", want.RowNumber, got.Error.Code, want.Error.Code)
	}
	if !strings.Contains(got.Error.Message, want.Error.MessageContains) {
		t.Fatalf("row %d error message mismatch: got %q want contains %q", want.RowNumber, got.Error.Message, want.Error.MessageContains)
	}
}
