package connector

import "database/sql"

// ========== 连接配置 ==========

// ConnectionConfig 数据库连接配置
type ConnectionConfig struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Group    string `json:"group,omitempty"`
	Type     string `json:"type"` // mysql | postgres | oracle | mssql
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	Database string `json:"database"`
	SSLMode  string `json:"sslMode,omitempty"` // PostgreSQL
	SID      string `json:"sid,omitempty"`     // Oracle SID（可选）

	// SSH 隧道
	UseSSH  bool   `json:"useSSH,omitempty"`
	SSHHost string `json:"sshHost,omitempty"`
	SSHPort int    `json:"sshPort,omitempty"`
	SSHUser string `json:"sshUser,omitempty"`
	SSHPass string `json:"sshPass,omitempty"`
	SSHKey  string `json:"sshKey,omitempty"`
}

// ========== 数据库对象模型 ==========

// Table 表/视图
type Table struct {
	Name     string `json:"name"`
	Type     string `json:"type"` // TABLE | VIEW
	Schema   string `json:"schema"`
	RowCount int64  `json:"rowCount"`
}

// Column 列定义
type Column struct {
	Name       string `json:"name"`
	DataType   string `json:"dataType"`
	Nullable   bool   `json:"nullable"`
	DefaultVal string `json:"defaultVal,omitempty"`
	PrimaryKey bool   `json:"primaryKey"`
	Comment    string `json:"comment,omitempty"`
}

// Index 索引
type Index struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
	Unique  bool     `json:"unique"`
	Type    string   `json:"type"` // BTREE | HASH | GIN | ...
}

// ForeignKey 外键
type ForeignKey struct {
	Name      string `json:"name"`
	Column    string `json:"column"`
	RefTable  string `json:"refTable"`
	RefColumn string `json:"refColumn"`
}

// ========== 查询结果 ==========

// QueryResult 查询返回结果
type QueryResult struct {
	Columns  []string        `json:"columns"`
	Rows     [][]interface{} `json:"rows"`
	RowCount int             `json:"rowCount"`
	Duration string          `json:"duration"`
}

// ========== 自动补全 ==========

// AutoCompleteData SQL 补全提示数据
type AutoCompleteData struct {
	Keywords  []string    `json:"keywords"`
	Functions []string    `json:"functions"`
	Tables    []TableRef  `json:"tables"`
	Columns   []ColumnRef `json:"columns"`
}

// TableRef 表引用（用于补全）
type TableRef struct {
	Name   string `json:"name"`
	Schema string `json:"schema,omitempty"`
}

// ColumnRef 列引用（用于补全）
type ColumnRef struct {
	Table  string `json:"table"`
	Column string `json:"column"`
	Type   string `json:"type"`
}

// ========== 表结构修改 ==========

// ColumnChange 修改列的请求（所有字段可选）
type ColumnChange struct {
	OldName  string  `json:"oldName"`
	NewName  *string `json:"newName,omitempty"`
	NewType  *string `json:"newType,omitempty"`
	Comment  *string `json:"comment,omitempty"`  // 空字符串 = 删除注释
	Default  *string `json:"default,omitempty"`
	Nullable *bool   `json:"nullable,omitempty"`
}

// ColumnDef 新增列的请求
type ColumnDef struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
	Default  string `json:"default,omitempty"`
	Comment  string `json:"comment,omitempty"`
}

// ========== 用户管理 ==========

// DatabaseUser 数据库用户
type DatabaseUser struct {
	Name  string   `json:"name"`
	Host  string   `json:"host,omitempty"`
	Roles []string `json:"roles,omitempty"`
}

// UserPrivilege 用户权限
type UserPrivilege struct {
	User       string   `json:"user"`
	Database   string   `json:"database,omitempty"`
	Table      string   `json:"table,omitempty"` // "*" 表示全局
	Privileges []string `json:"privileges"`
}

// ========== DBPlugin 接口 ==========

// DBPlugin 数据库插件接口，每个数据库类型实现此接口
type DBPlugin interface {
	// --- 连接 ---
	Open(cfg *ConnectionConfig) (*sql.DB, error)
	Ping(db *sql.DB) error
	GetDSN(cfg *ConnectionConfig) string // 生成连接串，供 SSH 隧道使用

	// --- 数据库级 ---
	ListDatabases(db *sql.DB) ([]string, error)
	SetDatabase(db *sql.DB, database string) error // 切换当前数据库上下文

	// --- 表级 ---
	ListTables(db *sql.DB, database string) ([]Table, error)
	GetColumns(db *sql.DB, database, table string) ([]Column, error)
	GetIndexes(db *sql.DB, database, table string) ([]Index, error)
	GetForeignKeys(db *sql.DB, database, table string) ([]ForeignKey, error)
	GetDDL(db *sql.DB, database, table string) (string, error)

	// --- 数据查询与修改 ---
	Query(db *sql.DB, sqlStr string, limit, offset int) (*QueryResult, error)
	Execute(db *sql.DB, sqlStr string) (int64, error) // 返回影响行数

	// --- 表结构修改 ---
	AlterColumn(db *sql.DB, database, table string, col ColumnChange) error
	AddColumn(db *sql.DB, database, table string, col ColumnDef) error
	DropColumn(db *sql.DB, database, table, column string) error

	// --- 用户管理 ---
	ListUsers(db *sql.DB) ([]DatabaseUser, error)
	CreateUser(db *sql.DB, user, password string) error
	DropUser(db *sql.DB, user string) error
	GetPrivileges(db *sql.DB, user string) ([]UserPrivilege, error)
	GrantPrivilege(db *sql.DB, user, database, table string, privs []string) error
	RevokePrivilege(db *sql.DB, user, database, table string, privs []string) error

	// --- 自动补全 ---
	GetAutoCompleteData(db *sql.DB, database string) (*AutoCompleteData, error)

	// --- 元数据 ---
	GetVersion(db *sql.DB) (string, error)
}
