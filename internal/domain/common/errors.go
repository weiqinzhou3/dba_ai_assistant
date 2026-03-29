package common

import "fmt"

type Code string

const (
	CodeRequestInvalid         Code = "REQ_INVALID"
	CodePrincipalNotFound      Code = "PRINCIPAL_NOT_FOUND"
	CodeAssetNotFound          Code = "ASSET_NOT_FOUND"
	CodeAssetAmbiguous         Code = "ASSET_AMBIGUOUS"
	CodeActionNotAllowed       Code = "ACTION_NOT_ALLOWED"
	CodeScopeDenied            Code = "SCOPE_DENIED"
	CodeApprovalRequired       Code = "APPROVAL_REQUIRED"
	CodeApprovalRejected       Code = "APPROVAL_REJECTED"
	CodeSelfApprovalForbidden  Code = "SELF_APPROVAL_FORBIDDEN"
	CodeExecutorNotAllowed     Code = "EXECUTOR_NOT_ALLOWED"
	CodePlanBuildFailed        Code = "PLAN_BUILD_FAILED"
	CodePlanStale              Code = "PLAN_STALE"
	CodePlanRevalidationFailed Code = "PLAN_REVALIDATION_FAILED"
	CodeAdapterNotAvailable    Code = "ADAPTER_NOT_AVAILABLE"
	CodeExecutionFailed        Code = "EXECUTION_FAILED"
	CodeOrderAlreadyExecuted   Code = "ORDER_ALREADY_EXECUTED"
	CodeOrderNotExecutable     Code = "ORDER_NOT_EXECUTABLE"
	CodeIdempotencyConflict    Code = "IDEMPOTENCY_CONFLICT"
	CodeSystemInternalError    Code = "SYSTEM_INTERNAL_ERROR"
)

type CodedError struct {
	Code    Code
	Message string
	Details map[string]any
}

func (e *CodedError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func NewError(code Code, message string, details map[string]any) error {
	return &CodedError{
		Code:    code,
		Message: message,
		Details: details,
	}
}

func ErrorCodeOf(err error) Code {
	var coded *CodedError
	if err == nil {
		return ""
	}
	if ok := As(err, &coded); ok {
		return coded.Code
	}
	return CodeSystemInternalError
}

func ErrorCode(err error) Code {
	return ErrorCodeOf(err)
}
