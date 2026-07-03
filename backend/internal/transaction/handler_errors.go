package transaction

import (
	"net/http"

	appErrors "ledger_two/internal/errors"
	"ledger_two/internal/http/response"
)

func writeUnauthorized(w http.ResponseWriter) {
	response.WriteError(w, appErrors.NewAppError(http.StatusUnauthorized, appErrors.ErrCodeUnauthorized, "请先登录系统"))
}

func writeBadRequest(w http.ResponseWriter, message string) {
	response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeBadRequest, message))
}

func writeValidationError(w http.ResponseWriter, message string) {
	response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, message))
}

func writeInternalError(w http.ResponseWriter, message string) {
	response.WriteError(w, appErrors.NewAppError(http.StatusInternalServerError, appErrors.ErrCodeInternalError, message))
}
