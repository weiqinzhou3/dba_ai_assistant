package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	appaction "dba_ai_assistant/internal/application/actionrequest"
	appapproval "dba_ai_assistant/internal/application/approval"
	appaudit "dba_ai_assistant/internal/application/audit"
	appauth "dba_ai_assistant/internal/application/authorization"
	appevidence "dba_ai_assistant/internal/application/evidence"
	"dba_ai_assistant/internal/domain/common"
)

type Dependencies struct {
	ActionRequests appaction.Service
	Approvals      appapproval.Service
	Audit          appaudit.Service
	Evidence       appevidence.Service
}

func NewServer(deps Dependencies) http.Handler {
	server := &Server{
		actionRequests: deps.ActionRequests,
		approvals:      deps.Approvals,
		audit:          deps.Audit,
		evidence:       deps.Evidence,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/action-requests", server.handleSubmitActionRequest)
	mux.HandleFunc("GET /api/v1/orders/{order_id}", server.handleGetOrder)
	mux.HandleFunc("GET /api/v1/tasks/{task_id}", server.handleGetTask)
	mux.HandleFunc("POST /api/v1/orders/{order_id}/approvals", server.handleDecideApproval)
	mux.HandleFunc("POST /api/v1/orders/{order_id}/execute", server.handleExecuteOrder)
	mux.HandleFunc("GET /api/v1/audit-ledger/{request_id}", server.handleGetAuditLedger)
	mux.HandleFunc("GET /api/v1/evidence-packs/{order_id}", server.handleGetEvidencePack)
	return mux
}

type Server struct {
	actionRequests appaction.Service
	approvals      appapproval.Service
	audit          appaudit.Service
	evidence       appevidence.Service
}

func (s *Server) handleSubmitActionRequest(w http.ResponseWriter, r *http.Request) {
	var req appaction.ActionRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r.Context(), common.NewError(common.CodeRequestInvalid, "invalid request body", nil), "")
		return
	}
	authCtx := authContextFromRequest(r)
	if authCtx.AuthenticatedPrincipalID != "" {
		if req.PrincipalID != "" && req.PrincipalID != authCtx.AuthenticatedPrincipalID {
			writeError(w, r.Context(), common.NewError(common.CodeRequestInvalid, "principal_id does not match authenticated principal", map[string]any{
				"principal_id":               req.PrincipalID,
				"authenticated_principal_id": authCtx.AuthenticatedPrincipalID,
			}), "")
			return
		}
		req.PrincipalID = authCtx.AuthenticatedPrincipalID
	}

	result, err := s.actionRequests.Submit(r.Context(), req)
	if err != nil {
		writeError(w, r.Context(), err, "")
		return
	}
	writeJSON(w, http.StatusAccepted, result)
}

func (s *Server) handleGetOrder(w http.ResponseWriter, r *http.Request) {
	result, err := s.actionRequests.GetOrder(r.Context(), r.PathValue("order_id"))
	if err != nil {
		writeError(w, r.Context(), err, "")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleGetTask(w http.ResponseWriter, r *http.Request) {
	result, err := s.actionRequests.GetTask(r.Context(), r.PathValue("task_id"))
	if err != nil {
		writeError(w, r.Context(), err, "")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleDecideApproval(w http.ResponseWriter, r *http.Request) {
	if s.approvals == nil {
		writeError(w, r.Context(), common.NewError(common.CodeSystemInternalError, "approval service not configured", nil), "")
		return
	}

	var req appapproval.DecisionInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r.Context(), common.NewError(common.CodeRequestInvalid, "invalid request body", nil), "")
		return
	}
	authCtx := authContextFromRequest(r)
	if authCtx.AuthenticatedPrincipalID != "" {
		if req.ApproverID != "" && req.ApproverID != authCtx.AuthenticatedPrincipalID {
			writeError(w, r.Context(), common.NewError(common.CodeRequestInvalid, "approver_id does not match authenticated principal", map[string]any{
				"approver_id":                req.ApproverID,
				"authenticated_principal_id": authCtx.AuthenticatedPrincipalID,
			}), "")
			return
		}
		req.ApproverID = authCtx.AuthenticatedPrincipalID
	}

	result, err := s.approvals.Decide(r.Context(), r.PathValue("order_id"), req)
	if err != nil {
		writeError(w, r.Context(), err, "")
		return
	}
	writeJSON(w, http.StatusAccepted, result)
}

func (s *Server) handleExecuteOrder(w http.ResponseWriter, r *http.Request) {
	var req appaction.ExecuteOrderInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r.Context(), common.NewError(common.CodeRequestInvalid, "invalid request body", nil), "")
		return
	}
	req.OrderID = r.PathValue("order_id")

	result, err := s.actionRequests.ExecuteApprovedOrder(r.Context(), authContextFromRequest(r), req)
	if err != nil {
		writeError(w, r.Context(), err, "")
		return
	}
	writeJSON(w, http.StatusAccepted, result)
}

func (s *Server) handleGetAuditLedger(w http.ResponseWriter, r *http.Request) {
	if s.audit == nil {
		writeError(w, r.Context(), common.NewError(common.CodeSystemInternalError, "audit service not configured", nil), "")
		return
	}

	result, err := s.audit.GetViewByRequestID(r.Context(), r.PathValue("request_id"))
	if err != nil {
		writeError(w, r.Context(), err, "")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleGetEvidencePack(w http.ResponseWriter, r *http.Request) {
	if s.evidence == nil {
		writeError(w, r.Context(), common.NewError(common.CodeSystemInternalError, "evidence service not configured", nil), "")
		return
	}

	result, err := s.evidence.GetByOrderID(r.Context(), r.PathValue("order_id"))
	if err != nil {
		writeError(w, r.Context(), err, "")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

type errorEnvelope struct {
	Error struct {
		Code    string         `json:"code"`
		Message string         `json:"message"`
		Details map[string]any `json:"details,omitempty"`
	} `json:"error"`
	TraceID string `json:"trace_id"`
}

func authContextFromRequest(r *http.Request) appauth.AuthContext {
	rolesHeader := strings.TrimSpace(r.Header.Get("X-Roles"))
	var roles []string
	if rolesHeader != "" {
		for _, role := range strings.Split(rolesHeader, ",") {
			trimmed := strings.TrimSpace(role)
			if trimmed != "" {
				roles = append(roles, trimmed)
			}
		}
	}

	return appauth.AuthContext{
		AuthenticatedPrincipalID: r.Header.Get("X-Principal-ID"),
		Roles:                    roles,
		Source:                   "http",
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, _ context.Context, err error, traceID string) {
	if traceID == "" {
		traceID = "trace_api_" + time.Now().UTC().Format("20060102150405")
	}

	status := http.StatusInternalServerError
	switch common.ErrorCode(err) {
	case common.CodeRequestInvalid:
		status = http.StatusBadRequest
	case common.CodeApprovalRequired, common.CodeOrderNotExecutable, common.CodeOrderAlreadyExecuted, common.CodePlanStale, common.CodeIdempotencyConflict:
		status = http.StatusConflict
	case common.CodeActionNotAllowed, common.CodeScopeDenied, common.CodeExecutorNotAllowed, common.CodeApprovalRejected, common.CodeSelfApprovalForbidden:
		status = http.StatusForbidden
	case common.CodeAssetNotFound, common.CodePrincipalNotFound:
		status = http.StatusNotFound
	case common.CodeAssetAmbiguous:
		status = http.StatusConflict
	}

	response := errorEnvelope{TraceID: traceID}
	response.Error.Code = string(common.ErrorCode(err))
	response.Error.Message = err.Error()
	if coded, ok := err.(*common.CodedError); ok {
		response.Error.Message = coded.Message
		response.Error.Details = coded.Details
	}
	writeJSON(w, status, response)
}
