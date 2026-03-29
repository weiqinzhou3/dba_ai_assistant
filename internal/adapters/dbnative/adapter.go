package dbnative

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"

	appexec "dba_ai_assistant/internal/application/execution"
)

const (
	IssueUnsupportedAction       = "UNSUPPORTED_ACTION"
	IssueUnsupportedTarget       = "UNSUPPORTED_TARGET"
	IssueConnectionRefMissing    = "CONNECTION_REF_MISSING"
	IssueConnectionRefUnresolved = "CONNECTION_REF_UNRESOLVED"
	IssueTargetEngineUnsupported = "TARGET_ENGINE_UNSUPPORTED"
	IssueDatabaseNameInvalid     = "DATABASE_NAME_INVALID"
	IssueMySQLUnreachable        = "MYSQL_UNREACHABLE"
	IssueDatabaseAlreadyExists   = "DATABASE_ALREADY_EXISTS"
)

var databaseNamePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]{0,63}$`)

type Dependencies struct {
	ConnectionResolver ConnectionResolver
	MySQL              MySQLAdmin
	Now                func() time.Time
}

type ConnectionConfig struct {
	Engine string
	DSN    string
}

type ConnectionResolver interface {
	Resolve(ctx context.Context, connectionRef string) (ConnectionConfig, error)
}

type MySQLAdmin interface {
	Ping(ctx context.Context, cfg ConnectionConfig) error
	DatabaseExists(ctx context.Context, cfg ConnectionConfig, databaseName string) (bool, error)
	CreateDatabase(ctx context.Context, cfg ConnectionConfig, databaseName string) error
}

type DBNativeAdapter struct {
	connectionResolver ConnectionResolver
	mysql              MySQLAdmin
	now                func() time.Time
}

func New(deps Dependencies) *DBNativeAdapter {
	resolver := deps.ConnectionResolver
	if resolver == nil {
		resolver = NewEnvConnectionResolver()
	}
	mysqlAdmin := deps.MySQL
	if mysqlAdmin == nil {
		mysqlAdmin = NewSQLAdmin()
	}
	now := deps.Now
	if now == nil {
		now = func() time.Time {
			return time.Now().UTC()
		}
	}
	return &DBNativeAdapter{
		connectionResolver: resolver,
		mysql:              mysqlAdmin,
		now:                now,
	}
}

func (a *DBNativeAdapter) Type() appexec.AdapterType {
	return appexec.AdapterTypeDBNative
}

func (a *DBNativeAdapter) Supports(_ context.Context, req appexec.AdapterCapabilityRequest) (bool, error) {
	return req.ActionName == "mysql.database.create" && req.TargetAssetType == "DatabaseTarget", nil
}

func (a *DBNativeAdapter) DryRun(ctx context.Context, req appexec.AdapterExecutionRequest) (appexec.AdapterDryRunResult, error) {
	preview := map[string]any{
		"action_name":   req.ActionName,
		"database_name": stringValue(req.Parameters["database_name"]),
		"sql_summary":   sqlSummary(stringValue(req.Parameters["database_name"])),
	}

	cfg, issues, err := a.validate(ctx, req, preview)
	if err != nil {
		return appexec.AdapterDryRunResult{}, err
	}
	if len(issues) > 0 {
		return appexec.AdapterDryRunResult{
			Supported:       !containsIssue(issues, IssueUnsupportedAction) && !containsIssue(issues, IssueUnsupportedTarget),
			Ready:           false,
			Issues:          issues,
			RenderedPreview: preview,
		}, nil
	}

	exists, err := a.mysql.DatabaseExists(ctx, cfg, stringValue(req.Parameters["database_name"]))
	if err != nil {
		preview["database_exists"] = false
		return appexec.AdapterDryRunResult{
			Supported:       true,
			Ready:           false,
			Issues:          []string{IssueMySQLUnreachable},
			RenderedPreview: preview,
		}, nil
	}
	preview["database_exists"] = exists

	result := appexec.AdapterDryRunResult{
		Supported:       true,
		Ready:           !exists,
		RenderedPreview: preview,
	}
	if exists {
		result.Issues = []string{IssueDatabaseAlreadyExists}
	}
	return result, nil
}

func (a *DBNativeAdapter) Execute(ctx context.Context, req appexec.AdapterExecutionRequest) (appexec.AdapterExecutionResult, error) {
	startedAt := a.now()
	dryRun, err := a.DryRun(ctx, req)
	if err != nil {
		return appexec.AdapterExecutionResult{}, err
	}
	if !dryRun.Supported {
		return a.failureResult(startedAt, req, dryRun.RenderedPreview, IssueUnsupportedAction, "dbnative adapter does not support this action"), nil
	}
	if !dryRun.Ready {
		code := firstIssueOrDefault(dryRun.Issues, IssueMySQLUnreachable)
		message := failureMessageForIssue(code)
		return a.failureResult(startedAt, req, dryRun.RenderedPreview, code, message), nil
	}

	cfg, _, err := a.validate(ctx, req, map[string]any{})
	if err != nil {
		return appexec.AdapterExecutionResult{}, err
	}

	databaseName := stringValue(req.Parameters["database_name"])
	sqlText := sqlSummary(databaseName)
	createErr := a.mysql.CreateDatabase(ctx, cfg, databaseName)
	if createErr != nil {
		afterExists, _ := a.mysql.DatabaseExists(ctx, cfg, databaseName)
		return appexec.AdapterExecutionResult{
			Success:   false,
			Status:    "FAILED",
			Summary:   "create database failed",
			StartedAt: startedAt,
			EndedAt:   a.now(),
			Outputs: map[string]any{
				"database_name":          databaseName,
				"database_exists_before": false,
				"database_exists_after":  afterExists,
				"sql_summary":            sqlText,
				"target_connection_ref":  stringValue(req.Target["connection_ref"]),
				"target_asset_id":        stringValue(req.Target["asset_id"]),
				"idempotency_key":        req.IdempotencyKey,
			},
			Artifacts: executionArtifacts(req, false),
			Error: map[string]any{
				"code":    "MYSQL_EXECUTION_ERROR",
				"message": createErr.Error(),
			},
		}, nil
	}

	afterExists, err := a.mysql.DatabaseExists(ctx, cfg, databaseName)
	if err != nil {
		return appexec.AdapterExecutionResult{
			Success:   false,
			Status:    "FAILED",
			Summary:   "post-create verification failed",
			StartedAt: startedAt,
			EndedAt:   a.now(),
			Outputs: map[string]any{
				"database_name":          databaseName,
				"database_exists_before": false,
				"sql_summary":            sqlText,
				"target_connection_ref":  stringValue(req.Target["connection_ref"]),
				"target_asset_id":        stringValue(req.Target["asset_id"]),
				"idempotency_key":        req.IdempotencyKey,
			},
			Artifacts: executionArtifacts(req, false),
			Error: map[string]any{
				"code":    IssueMySQLUnreachable,
				"message": err.Error(),
			},
		}, nil
	}
	if !afterExists {
		return appexec.AdapterExecutionResult{
			Success:   false,
			Status:    "FAILED",
			Summary:   "database creation could not be verified",
			StartedAt: startedAt,
			EndedAt:   a.now(),
			Outputs: map[string]any{
				"database_name":          databaseName,
				"database_exists_before": false,
				"database_exists_after":  false,
				"sql_summary":            sqlText,
				"target_connection_ref":  stringValue(req.Target["connection_ref"]),
				"target_asset_id":        stringValue(req.Target["asset_id"]),
				"idempotency_key":        req.IdempotencyKey,
			},
			Artifacts: executionArtifacts(req, false),
			Error: map[string]any{
				"code":    "DATABASE_VERIFICATION_FAILED",
				"message": "database does not exist after CREATE DATABASE returned success",
			},
		}, nil
	}

	return appexec.AdapterExecutionResult{
		Success:   true,
		Status:    "SUCCEEDED",
		Summary:   "database created",
		StartedAt: startedAt,
		EndedAt:   a.now(),
		Outputs: map[string]any{
			"database_name":          databaseName,
			"database_exists_before": false,
			"database_exists_after":  true,
			"sql_summary":            sqlText,
			"target_connection_ref":  stringValue(req.Target["connection_ref"]),
			"target_asset_id":        stringValue(req.Target["asset_id"]),
			"idempotency_key":        req.IdempotencyKey,
		},
		Artifacts: executionArtifacts(req, true),
	}, nil
}

func (a *DBNativeAdapter) validate(ctx context.Context, req appexec.AdapterExecutionRequest, preview map[string]any) (ConnectionConfig, []string, error) {
	issues := make([]string, 0, 4)
	if req.ActionName != "mysql.database.create" {
		issues = append(issues, IssueUnsupportedAction)
	}
	if stringValue(req.Target["asset_type"]) != "DatabaseTarget" {
		issues = append(issues, IssueUnsupportedTarget)
	}

	connectionRef := stringValue(req.Target["connection_ref"])
	if connectionRef == "" {
		issues = append(issues, IssueConnectionRefMissing)
	}
	preview["connection_ref"] = connectionRef
	preview["target_asset_id"] = stringValue(req.Target["asset_id"])
	preview["asset_type"] = stringValue(req.Target["asset_type"])

	engine := strings.ToLower(strings.TrimSpace(stringValue(req.Target["engine"])))
	if engine == "" {
		engine = "mysql"
	}
	preview["engine"] = engine
	if engine != "mysql" {
		issues = append(issues, IssueTargetEngineUnsupported)
	}

	databaseName := stringValue(req.Parameters["database_name"])
	if !databaseNamePattern.MatchString(databaseName) {
		issues = append(issues, IssueDatabaseNameInvalid)
	}
	if len(issues) > 0 {
		return ConnectionConfig{}, issues, nil
	}

	cfg, err := a.connectionResolver.Resolve(ctx, connectionRef)
	if err != nil {
		return ConnectionConfig{}, []string{IssueConnectionRefUnresolved}, nil
	}
	if cfg.Engine != "" && strings.ToLower(cfg.Engine) != "mysql" {
		return ConnectionConfig{}, []string{IssueTargetEngineUnsupported}, nil
	}
	if err := a.mysql.Ping(ctx, cfg); err != nil {
		return ConnectionConfig{}, []string{IssueMySQLUnreachable}, nil
	}
	return cfg, nil, nil
}

func (a *DBNativeAdapter) failureResult(startedAt time.Time, req appexec.AdapterExecutionRequest, preview map[string]any, code string, message string) appexec.AdapterExecutionResult {
	return appexec.AdapterExecutionResult{
		Success:   false,
		Status:    "FAILED",
		Summary:   message,
		StartedAt: startedAt,
		EndedAt:   a.now(),
		Outputs: map[string]any{
			"database_name":          stringValue(req.Parameters["database_name"]),
			"database_exists_before": preview["database_exists"],
			"sql_summary":            preview["sql_summary"],
			"target_connection_ref":  stringValue(req.Target["connection_ref"]),
			"target_asset_id":        stringValue(req.Target["asset_id"]),
			"idempotency_key":        req.IdempotencyKey,
		},
		Artifacts: executionArtifacts(req, false),
		Error: map[string]any{
			"code":    code,
			"message": message,
		},
	}
}

type EnvConnectionResolver struct {
	getenv func(string) string
}

func NewEnvConnectionResolver() *EnvConnectionResolver {
	return &EnvConnectionResolver{
		getenv: os.Getenv,
	}
}

func (r *EnvConnectionResolver) Resolve(_ context.Context, connectionRef string) (ConnectionConfig, error) {
	if strings.TrimSpace(connectionRef) == "" {
		return ConnectionConfig{}, errors.New("connection ref is required")
	}

	if raw := strings.TrimSpace(r.getenv("DBNATIVE_CONNECTIONS_JSON")); raw != "" {
		mappings := map[string]string{}
		if err := json.Unmarshal([]byte(raw), &mappings); err == nil {
			if dsn := strings.TrimSpace(mappings[connectionRef]); dsn != "" {
				return ConnectionConfig{
					Engine: "mysql",
					DSN:    dsn,
				}, nil
			}
		}
	}

	envKey := "DBNATIVE_DSN_" + sanitizeEnvSuffix(connectionRef)
	if dsn := strings.TrimSpace(r.getenv(envKey)); dsn != "" {
		return ConnectionConfig{
			Engine: "mysql",
			DSN:    dsn,
		}, nil
	}
	return ConnectionConfig{}, fmt.Errorf("connection ref %s is not configured", connectionRef)
}

type SQLAdmin struct{}

func NewSQLAdmin() *SQLAdmin {
	return &SQLAdmin{}
}

func (a *SQLAdmin) Ping(ctx context.Context, cfg ConnectionConfig) error {
	db, err := sql.Open("mysql", cfg.DSN)
	if err != nil {
		return err
	}
	defer db.Close()
	return db.PingContext(ctx)
}

func (a *SQLAdmin) DatabaseExists(ctx context.Context, cfg ConnectionConfig, databaseName string) (bool, error) {
	db, err := sql.Open("mysql", cfg.DSN)
	if err != nil {
		return false, err
	}
	defer db.Close()

	var exists bool
	if err := db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM information_schema.schemata WHERE schema_name = ?)", databaseName).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

func (a *SQLAdmin) CreateDatabase(ctx context.Context, cfg ConnectionConfig, databaseName string) error {
	db, err := sql.Open("mysql", cfg.DSN)
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.ExecContext(ctx, sqlSummary(databaseName))
	return err
}

func executionArtifacts(req appexec.AdapterExecutionRequest, succeeded bool) []map[string]any {
	base := req.TaskID
	if base == "" {
		base = req.OrderID
	}
	result := []map[string]any{
		{
			"type": "sql_summary",
			"ref":  "artifact://dbnative/sql/" + base,
		},
		{
			"type": "before_snapshot",
			"ref":  "artifact://dbnative/before/" + base,
		},
		{
			"type": "after_snapshot",
			"ref":  "artifact://dbnative/after/" + base,
		},
	}
	if !succeeded {
		result = append(result, map[string]any{
			"type": "failure_detail",
			"ref":  "artifact://dbnative/failure/" + base,
		})
	}
	return result
}

func sqlSummary(databaseName string) string {
	if !databaseNamePattern.MatchString(databaseName) {
		return ""
	}
	return fmt.Sprintf("CREATE DATABASE `%s`", databaseName)
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}

func containsIssue(issues []string, target string) bool {
	for _, issue := range issues {
		if issue == target {
			return true
		}
	}
	return false
}

func firstIssueOrDefault(issues []string, fallback string) string {
	if len(issues) == 0 {
		return fallback
	}
	return issues[0]
}

func failureMessageForIssue(code string) string {
	switch code {
	case IssueDatabaseAlreadyExists:
		return "database already exists"
	case IssueDatabaseNameInvalid:
		return "database name violates platform naming convention"
	case IssueConnectionRefMissing:
		return "target connection ref is missing"
	case IssueConnectionRefUnresolved:
		return "target connection ref could not be resolved"
	case IssueTargetEngineUnsupported:
		return "target engine is not mysql"
	case IssueUnsupportedAction:
		return "unsupported action"
	case IssueUnsupportedTarget:
		return "unsupported target asset type"
	default:
		return "mysql target is not ready"
	}
}

func sanitizeEnvSuffix(value string) string {
	builder := strings.Builder{}
	builder.Grow(len(value))
	for _, r := range value {
		switch {
		case r >= 'A' && r <= 'Z':
			builder.WriteRune(r)
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r - 'a' + 'A')
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
		default:
			builder.WriteByte('_')
		}
	}
	return builder.String()
}
