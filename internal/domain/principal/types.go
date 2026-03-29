package principal

type Type string

const (
	TypeHuman      Type = "human"
	TypeService    Type = "service"
	TypeAgentProxy Type = "agent_proxy"
)

const (
	RoleAssistantUser   = "assistant_user"
	RoleDBA             = "dba"
	RoleMySQLOperator   = "mysql_operator"
	RoleBackupOperator  = "backup_operator"
	RoleProdApprover    = "prod_approver"
	RolePlatformAdmin   = "platform_admin"
	RoleReadonlyAuditor = "readonly_auditor"
	RoleControlExecutor = "control_executor"
)

type Principal struct {
	PrincipalID      string              `json:"principal_id"`
	PrincipalType    Type                `json:"principal_type"`
	UserID           string              `json:"user_id,omitempty"`
	DisplayName      string              `json:"display_name,omitempty"`
	Roles            []string            `json:"roles"`
	Groups           []string            `json:"groups,omitempty"`
	Scopes           map[string][]string `json:"scopes,omitempty"`
	ApprovalRoles    []string            `json:"approval_roles,omitempty"`
	PolicyExemptions []string            `json:"policy_exemptions,omitempty"`
	IsActive         bool                `json:"is_active"`
}
