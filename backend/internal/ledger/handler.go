package ledger

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	appErrors "ledger_two/internal/errors"
	"ledger_two/internal/http/middleware"
	"ledger_two/internal/http/response"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) CreateLedger(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserIDFromContext(r.Context())
	if userID == "" {
		response.WriteError(w, appErrors.NewAppError(http.StatusUnauthorized, appErrors.ErrCodeUnauthorized, "请先登录系统"))
		return
	}

	var req CreateLedgerReq
	if err := decodeLedgerJSON(r, &req); err != nil {
		response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeBadRequest, "请求参数解析失败"))
		return
	}

	l, err := h.svc.CreateLedger(r.Context(), userID, req)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	w.Header().Set("ETag", FormatLedgerETag(l.ID, l.Version))
	response.JSON(w, http.StatusCreated, l)
}

func (h *Handler) ListUserLedgers(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserIDFromContext(r.Context())
	if userID == "" {
		response.WriteError(w, appErrors.NewAppError(http.StatusUnauthorized, appErrors.ErrCodeUnauthorized, "请先登录系统"))
		return
	}

	status := LedgerListStatus(strings.TrimSpace(r.URL.Query().Get("status")))
	if status == "" {
		status = LedgerListActive
	}
	list, err := h.svc.ListUserLedgers(r.Context(), userID, status)
	if err != nil {
		response.WriteError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, list)
}

func (h *Handler) GetLedger(w http.ResponseWriter, r *http.Request) {
	lc, ok := LedgerContextFromContext(r.Context())
	if !ok {
		response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeLedgerRequired, "请选择账本后再执行此操作"))
		return
	}
	ledgerModel, err := h.svc.GetLedger(r.Context(), lc)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	w.Header().Set("ETag", FormatLedgerETag(ledgerModel.ID, ledgerModel.Version))
	response.JSON(w, http.StatusOK, ledgerModel)
}

func (h *Handler) RenameLedger(w http.ResponseWriter, r *http.Request) {
	lc, expectedVersion, ok := lifecycleMutationContext(w, r)
	if !ok {
		return
	}
	var req RenameLedgerReq
	if err := decodeLedgerJSON(r, &req); err != nil {
		response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeBadRequest, "请求参数解析失败"))
		return
	}
	ledgerModel, err := h.svc.RenameLedger(r.Context(), lc, expectedVersion, req)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	w.Header().Set("ETag", FormatLedgerETag(ledgerModel.ID, ledgerModel.Version))
	response.JSON(w, http.StatusOK, ledgerModel)
}

func (h *Handler) GetArchivePreflight(w http.ResponseWriter, r *http.Request) {
	lc, ok := LedgerContextFromContext(r.Context())
	if !ok {
		response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeLedgerRequired, "请选择账本后再执行此操作"))
		return
	}
	preflight, err := h.svc.GetArchivePreflight(r.Context(), lc)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	w.Header().Set("ETag", FormatLedgerETag(preflight.Ledger.ID, preflight.Ledger.Version))
	response.JSON(w, http.StatusOK, preflight)
}

func (h *Handler) ArchiveLedger(w http.ResponseWriter, r *http.Request) {
	lc, expectedVersion, ok := lifecycleMutationContext(w, r)
	if !ok {
		return
	}
	var req ArchiveLedgerReq
	if err := decodeLedgerJSON(r, &req); err != nil {
		response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeBadRequest, "请求参数解析失败"))
		return
	}
	ledgerModel, err := h.svc.ArchiveLedger(r.Context(), lc, expectedVersion, req)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	w.Header().Set("ETag", FormatLedgerETag(ledgerModel.ID, ledgerModel.Version))
	response.JSON(w, http.StatusOK, ledgerModel)
}

func (h *Handler) RestoreLedger(w http.ResponseWriter, r *http.Request) {
	lc, expectedVersion, ok := lifecycleMutationContext(w, r)
	if !ok {
		return
	}
	ledgerModel, err := h.svc.RestoreLedger(r.Context(), lc, expectedVersion)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	w.Header().Set("ETag", FormatLedgerETag(ledgerModel.ID, ledgerModel.Version))
	response.JSON(w, http.StatusOK, ledgerModel)
}

func lifecycleMutationContext(w http.ResponseWriter, r *http.Request) (LedgerContext, int64, bool) {
	lc, ok := LedgerContextFromContext(r.Context())
	if !ok {
		response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeLedgerRequired, "请选择账本后再执行此操作"))
		return LedgerContext{}, 0, false
	}
	version, err := ParseLedgerIfMatch(strings.TrimSpace(r.Header.Get("If-Match")), lc.LedgerID)
	if err != nil {
		response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeValidationError, "If-Match 缺失或格式错误"))
		return LedgerContext{}, 0, false
	}
	return lc, version, true
}

func decodeLedgerJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeBadRequest, "请求体只能包含一个 JSON 对象")
	}
	return nil
}

func (h *Handler) GetLedgerMembers(w http.ResponseWriter, r *http.Request) {
	lc, ok := LedgerContextFromContext(r.Context())
	if !ok {
		response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeLedgerRequired, "请选择账本后再执行此操作"))
		return
	}
	result, err := h.svc.GetMemberList(r.Context(), lc)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	writeMemberListResponse(w, http.StatusOK, result)
}

func (h *Handler) AddMember(w http.ResponseWriter, r *http.Request) {
	lc, expectedVersion, ok := lifecycleMutationContext(w, r)
	if !ok {
		return
	}

	var req AddMemberReq
	if err := decodeLedgerJSON(r, &req); err != nil {
		response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeBadRequest, "请求参数解析失败"))
		return
	}

	result, err := h.svc.AddMemberVersioned(r.Context(), lc, expectedVersion, req)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	writeMemberListResponse(w, http.StatusCreated, result)
}

func (h *Handler) UpdateMemberRole(w http.ResponseWriter, r *http.Request) {
	lc, expectedVersion, ok := lifecycleMutationContext(w, r)
	if !ok {
		return
	}
	targetUserID := chi.URLParam(r, "userId")

	var req UpdateMemberReq
	if err := decodeLedgerJSON(r, &req); err != nil {
		response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeBadRequest, "请求参数解析失败"))
		return
	}

	result, err := h.svc.UpdateMemberRoleVersioned(r.Context(), lc, expectedVersion, targetUserID, req)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	writeMemberListResponse(w, http.StatusOK, result)
}

func (h *Handler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	lc, expectedVersion, ok := lifecycleMutationContext(w, r)
	if !ok {
		return
	}
	targetUserID := chi.URLParam(r, "userId")

	result, err := h.svc.RemoveMemberVersioned(r.Context(), lc, expectedVersion, targetUserID)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	writeMemberListResponse(w, http.StatusOK, result)
}

func (h *Handler) TransferOwner(w http.ResponseWriter, r *http.Request) {
	lc, expectedVersion, ok := lifecycleMutationContext(w, r)
	if !ok {
		return
	}
	targetUserID := chi.URLParam(r, "userId")
	var req TransferOwnerReq
	if err := decodeLedgerJSON(r, &req); err != nil {
		response.WriteError(w, appErrors.NewAppError(http.StatusBadRequest, appErrors.ErrCodeBadRequest, "请求参数解析失败"))
		return
	}
	result, err := h.svc.TransferOwnerVersioned(r.Context(), lc, expectedVersion, targetUserID, req)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	writeMemberListResponse(w, http.StatusOK, result)
}

func (h *Handler) LeaveLedger(w http.ResponseWriter, r *http.Request) {
	lc, expectedVersion, ok := lifecycleMutationContext(w, r)
	if !ok {
		return
	}
	result, err := h.svc.LeaveLedgerVersioned(r.Context(), lc, expectedVersion)
	if err != nil {
		response.WriteError(w, err)
		return
	}
	w.Header().Set("ETag", FormatLedgerETag(result.LedgerID, result.Version))
	response.JSON(w, http.StatusOK, nil)
}

func writeMemberListResponse(w http.ResponseWriter, status int, result *MemberListData) {
	w.Header().Set("ETag", FormatLedgerETag(result.Ledger.ID, result.Ledger.Version))
	response.JSON(w, status, result)
}
