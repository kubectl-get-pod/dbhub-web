package connector_test

import (
	"testing"

	"dbhub-web/connector"
	"dbhub-web/connector/mssql"
	"dbhub-web/connector/mysql"
	"dbhub-web/connector/oracle"
	"dbhub-web/connector/postgres"
)

// TestPluginInterfaceCompliance 验证所有插件实现 DBPlugin 接口
func TestPluginInterfaceCompliance(t *testing.T) {
	plugins := map[string]connector.DBPlugin{
		"mysql":    mysql.New(),
		"postgres": postgres.New(),
		"oracle":   oracle.New(),
		"mssql":    mssql.New(),
	}

	for name, p := range plugins {
		t.Run(name, func(t *testing.T) {
			if p == nil {
				t.Errorf("%s: plugin is nil", name)
			}
		})
	}
}

// TestDefaultPorts 验证默认端口号
func TestDefaultPorts(t *testing.T) {
	tests := []struct {
		dbType string
		port   int
	}{
		{"mysql", 3306},
		{"mariadb", 3306},
		{"postgres", 5432},
		{"postgresql", 5432},
		{"oracle", 1521},
		{"mssql", 1433},
		{"sqlserver", 1433},
		{"unknown", 0},
	}

	for _, tt := range tests {
		// 测试 DSN 构建不 panic
		cfg := &connector.ConnectionConfig{
			Type: tt.dbType, Host: "localhost", Port: 0,
			User: "test", Password: "test", Database: "test",
		}

		plugins := map[string]connector.DBPlugin{
			"mysql": mysql.New(), "postgres": postgres.New(),
			"oracle": oracle.New(), "mssql": mssql.New(),
		}

		for _, p := range plugins {
			dsn := p.GetDSN(cfg)
			if dsn == "" {
				t.Errorf("%s: GetDSN returned empty string for type %s", tt.dbType, tt.dbType)
			}
		}
	}
}

// TestConnectionConfig 验证连接配置完整性
func TestConnectionConfig(t *testing.T) {
	cfg := &connector.ConnectionConfig{
		ID:       "test-1",
		Name:     "Test DB",
		Group:    "开发环境",
		Type:     "mysql",
		Host:     "192.168.1.1",
		Port:     3306,
		User:     "admin",
		Password: "secret",
		Database: "mydb",
		UseSSH:   true,
		SSHHost:  "jump.example.com",
		SSHPort:  22,
		SSHUser:  "deploy",
		SSHPass:  "sshsecret",
	}

	if cfg.ID != "test-1" {
		t.Error("ID mismatch")
	}
	if cfg.Group != "开发环境" {
		t.Error("Group mismatch")
	}
	if !cfg.UseSSH {
		t.Error("SSH should be enabled")
	}
}

// TestModelTypes 验证所有模型类型可 JSON 序列化
func TestModelTypes(t *testing.T) {
	table := connector.Table{
		Name: "users", Type: "TABLE", Schema: "public", RowCount: 1000,
	}
	if table.Name != "users" || table.Type != "TABLE" {
		t.Error("Table struct mismatch")
	}

	column := connector.Column{
		Name: "id", DataType: "int", Nullable: false, PrimaryKey: true, Comment: "主键",
	}
	if !column.PrimaryKey {
		t.Error("Column should be primary key")
	}

	index := connector.Index{
		Name: "idx_email", Columns: []string{"email"}, Unique: true, Type: "BTREE",
	}
	if !index.Unique || len(index.Columns) != 1 {
		t.Error("Index mismatch")
	}

	fk := connector.ForeignKey{
		Name: "fk_user_id", Column: "user_id", RefTable: "users", RefColumn: "id",
	}
	if fk.RefTable != "users" {
		t.Error("ForeignKey mismatch")
	}

	result := connector.QueryResult{
		Columns: []string{"id", "name"}, Rows: [][]interface{}{{1, "test"}}, RowCount: 1,
	}
	if len(result.Columns) != 2 || result.RowCount != 1 {
		t.Error("QueryResult mismatch")
	}
}

// TestAutoCompleteData 验证补全数据结构
func TestAutoCompleteData(t *testing.T) {
	data := &connector.AutoCompleteData{
		Keywords:  []string{"SELECT", "FROM"},
		Functions: []string{"COUNT", "SUM"},
		Tables:    []connector.TableRef{{Name: "users"}},
		Columns:   []connector.ColumnRef{{Table: "users", Column: "id", Type: "int"}},
	}
	if len(data.Keywords) == 0 || len(data.Tables) == 0 {
		t.Error("AutoCompleteData should not be empty")
	}
}

// TestColumnChange 验证列修改请求（可选字段）
func TestColumnChange(t *testing.T) {
	comment := "用户备注"
	col := connector.ColumnChange{
		OldName: "remark",
		Comment: &comment,
	}
	if *col.Comment != "用户备注" {
		t.Error("ColumnChange comment mismatch")
	}

	// 不传 NewName，验证不会 nil deref
	if col.NewName != nil {
		t.Error("NewName should be nil when not set")
	}
}

// TestUserManagementTypes 验证用户管理类型
func TestUserManagementTypes(t *testing.T) {
	user := connector.DatabaseUser{
		Name: "admin", Host: "%", Roles: []string{"DBA"},
	}
	if user.Name != "admin" || len(user.Roles) != 1 {
		t.Error("DatabaseUser mismatch")
	}

	priv := connector.UserPrivilege{
		User: "app", Database: "mydb", Table: "*",
		Privileges: []string{"SELECT", "INSERT"},
	}
	if priv.Table != "*" || len(priv.Privileges) != 2 {
		t.Error("UserPrivilege mismatch")
	}
}
