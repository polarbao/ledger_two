package safety

import (
	"context"
	stderrors "errors"
	"testing"

	appErrors "ledger_two/internal/errors"
)

func TestExportJSONRejectsMissingExplicitLedgerContext(t *testing.T) {
	svc := &Service{}
	_, err := svc.ExportJSON(context.Background(), "user-a")
	var appErr *appErrors.AppError
	if !stderrors.As(err, &appErr) || appErr.Code != appErrors.ErrCodeLedgerRequired {
		t.Fatalf("expected LEDGER_REQUIRED, got %v", err)
	}
}
