package mssql

import (
	"database/sql"
	"fmt"
	"net/url"
	"strings"

	_ "github.com/microsoft/go-mssqldb"

	"dbhub-web/connector"
)

const PluginName = "mssql"

type Plugin struct{}

func New() *Plugin { return &Plugin{} }

func (p *Plugin) Open(cfg *connector.ConnectionConfig) (*sql.DB, error) {
	dsn := p.GetDSN(cfg)
	db, err := sql.Open("sqlserver", dsn)
	if err != nil {
		return nil, fmt.Errorf("mssql: 打开连接失败: %w", err)
	}
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	return db, nil
}

func (p *Plugin) Ping(db *sql.DB) error { return db.Ping() }

func (p *Plugin) GetDSN(cfg *connector.ConnectionConfig) string {
	host := cfg.Host
	port := cfg.Port
	if port == 0 {
		port = 1433
	}
	query := url.Values{}
	if cfg.Database != "" {
		query.Set("database", cfg.Database)
	}
	query.Set("encrypt", "disable")
	query.Set("connection timeout", "10")

	dsn := fmt.Sprintf("sqlserver://%s:%s@%s:%d?%s",
		url.QueryEscape(cfg.User),
		url.QueryEscape(cfg.Password),
		host, port,
		query.Encode(),
	)
	return dsn
}

// --- 数据库级 ---

func (p *Plugin) ListDatabases(db *sql.DB) ([]string, error) {
	rows, err := db.Query("SELECT name FROM sys.databases WHERE state_desc = 'ONLINE' ORDER BY name")
	if err != nil {
		return nil, fmt.Errorf("mssql: 列出数据库失败: %w", err)
	}
	defer rows.Close()

	var databases []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		databases = append(databases, name)
	}
	return databases, rows.Err()
}

// --- 表级 ---

func (p *Plugin) SetDatabase(db *sql.DB, database string) error {
	if database == "" {
		return nil
	}
	_, err := db.Exec(fmt.Sprintf("USE [%s]", escapeIdent(database)))
	return err
}

func (p *Plugin) ListTables(db *sql.DB, database string) ([]connector.Table, error) {
	query := fmt.Sprintf(`SELECT t.name, 'TABLE',
		(SELECT SUM(ps.row_count) FROM sys.dm_db_partition_stats ps WHERE ps.object_id = t.object_id AND ps.index_id < 2)
		FROM [%s].sys.tables t ORDER BY t.name
		UNION ALL
		SELECT v.name, 'VIEW', 0 FROM [%s].sys.views v ORDER BY name`,
		escapeIdent(database), escapeIdent(database))
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("mssql: 列出表失败: %w", err)
	}
	defer rows.Close()

	var tables []connector.Table
	for rows.Next() {
		var t connector.Table
		t.Schema = "dbo"
		if err := rows.Scan(&t.Name, &t.Type, &t.RowCount); err != nil {
			return nil, err
		}
		tables = append(tables, t)
	}
	return tables, rows.Err()
}

func (p *Plugin) GetColumns(db *sql.DB, database, table string) ([]connector.Column, error) {
	query := fmt.Sprintf(`SELECT c.COLUMN_NAME, c.DATA_TYPE,
		CASE WHEN c.IS_NULLABLE = 'YES' THEN 1 ELSE 0 END,
		COALESCE(c.COLUMN_DEFAULT, ''),
		CASE WHEN pk.COLUMN_NAME IS NOT NULL THEN 1 ELSE 0 END,
		COALESCE(ep.value, '')
	FROM [%s].INFORMATION_SCHEMA.COLUMNS c
	LEFT JOIN (
		SELECT ku.COLUMN_NAME FROM [%s].INFORMATION_SCHEMA.TABLE_CONSTRAINTS tc
		JOIN [%s].INFORMATION_SCHEMA.KEY_COLUMN_USAGE ku ON tc.CONSTRAINT_NAME = ku.CONSTRAINT_NAME
		WHERE tc.CONSTRAINT_TYPE = 'PRIMARY KEY' AND tc.TABLE_NAME = '%s'
	) pk ON c.COLUMN_NAME = pk.COLUMN_NAME
	LEFT JOIN sys.extended_properties ep ON ep.major_id = OBJECT_ID('%s.%s')
		AND ep.minor_id = c.ORDINAL_POSITION AND ep.name = 'MS_Description'
	WHERE c.TABLE_NAME = '%s'
	ORDER BY c.ORDINAL_POSITION`,
		escapeIdent(database), escapeIdent(database), escapeIdent(database), escapeString(table),
		escapeIdent(database), escapeIdent(table), escapeString(table))
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("mssql: 获取列信息失败: %w", err)
	}
	defer rows.Close()

	var columns []connector.Column
	for rows.Next() {
		var c connector.Column
		var isNullable int
		var defaultVal, comment sql.NullString
		if err := rows.Scan(&c.Name, &c.DataType, &isNullable, &defaultVal, &c.PrimaryKey, &comment); err != nil {
			return nil, err
		}
		c.Nullable = isNullable == 1
		if defaultVal.Valid {
			c.DefaultVal = defaultVal.String
		}
		if comment.Valid && comment.String != "" {
			c.Comment = comment.String
		}
		columns = append(columns, c)
	}
	if columns == nil {
		return []connector.Column{}, nil
	}
	return columns, rows.Err()
}

func (p *Plugin) GetIndexes(db *sql.DB, database, table string) ([]connector.Index, error) {
	query := fmt.Sprintf(`SELECT i.name, c.name as col_name, i.is_unique, i.type_desc
		FROM [%s].sys.indexes i
		JOIN [%s].sys.index_columns ic ON i.object_id = ic.object_id AND i.index_id = ic.index_id
		JOIN [%s].sys.columns c ON ic.object_id = c.object_id AND ic.column_id = c.column_id
		WHERE i.object_id = OBJECT_ID('%s.%s') AND i.is_primary_key = 0
		ORDER BY i.name, ic.key_ordinal`,
		escapeIdent(database), escapeIdent(database), escapeIdent(database),
		escapeIdent(database), escapeIdent(table))
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("mssql: 获取索引信息失败: %w", err)
	}
	defer rows.Close()

	indexMap := make(map[string]*connector.Index)
	var indexOrder []string
	for rows.Next() {
		var indexName, colName, indexType string
		var isUnique bool
		if err := rows.Scan(&indexName, &colName, &isUnique, &indexType); err != nil {
			return nil, err
		}
		if idx, ok := indexMap[indexName]; ok {
			idx.Columns = append(idx.Columns, colName)
		} else {
			indexMap[indexName] = &connector.Index{
				Name:    indexName,
				Columns: []string{colName},
				Unique:  isUnique,
				Type:    strings.ToUpper(indexType),
			}
			indexOrder = append(indexOrder, indexName)
		}
	}
	var indexes []connector.Index
	for _, name := range indexOrder {
		indexes = append(indexes, *indexMap[name])
	}
	if indexes == nil {
		return []connector.Index{}, nil
	}
	return indexes, rows.Err()
}

func (p *Plugin) GetForeignKeys(db *sql.DB, database, table string) ([]connector.ForeignKey, error) {
	query := fmt.Sprintf(`SELECT fk.name, pc.name as col_name, rt.name as ref_table, rc.name as ref_col
		FROM [%s].sys.foreign_keys fk
		JOIN [%s].sys.foreign_key_columns fkc ON fk.object_id = fkc.constraint_object_id
		JOIN [%s].sys.columns pc ON fkc.parent_object_id = pc.object_id AND fkc.parent_column_id = pc.column_id
		JOIN [%s].sys.tables rt ON fkc.referenced_object_id = rt.object_id
		JOIN [%s].sys.columns rc ON fkc.referenced_object_id = rc.object_id AND fkc.referenced_column_id = rc.column_id
		WHERE fk.parent_object_id = OBJECT_ID('%s.%s')`,
		escapeIdent(database), escapeIdent(database), escapeIdent(database),
		escapeIdent(database), escapeIdent(database),
		escapeIdent(database), escapeIdent(table))
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("mssql: 获取外键信息失败: %w", err)
	}
	defer rows.Close()

	var fks []connector.ForeignKey
	for rows.Next() {
		var fk connector.ForeignKey
		if err := rows.Scan(&fk.Name, &fk.Column, &fk.RefTable, &fk.RefColumn); err != nil {
			return nil, err
		}
		fks = append(fks, fk)
	}
	if fks == nil {
		return []connector.ForeignKey{}, nil
	}
	return fks, rows.Err()
}

func (p *Plugin) GetDDL(db *sql.DB, database, table string) (string, error) {
	cols, err := p.GetColumns(db, database, table)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("CREATE TABLE [%s] (\n", table))
	for i, c := range cols {
		sb.WriteString(fmt.Sprintf("  [%s] %s", c.Name, c.DataType))
		if !c.Nullable {
			sb.WriteString(" NOT NULL")
		}
		if c.DefaultVal != "" {
			sb.WriteString(fmt.Sprintf(" DEFAULT %s", c.DefaultVal))
		}
		if i < len(cols)-1 {
			sb.WriteString(",")
		}
		sb.WriteString("\n")
	}
	sb.WriteString(")")
	return sb.String(), nil
}

// --- 数据查询 ---

func (p *Plugin) Query(db *sql.DB, sqlStr string, limit, offset int) (*connector.QueryResult, error) {
	if limit > 0 {
		sqlStr = strings.TrimSpace(strings.TrimRight(sqlStr, ";"))
		if !strings.Contains(strings.ToUpper(sqlStr), "OFFSET") {
			sqlStr = fmt.Sprintf("%s OFFSET %d ROWS FETCH NEXT %d ROWS ONLY", sqlStr, offset, limit)
		}
	}
	rows, err := db.Query(sqlStr)
	if err != nil {
		return nil, fmt.Errorf("mssql: 查询失败: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var resultRows [][]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}
		for i, v := range values {
			if b, ok := v.([]byte); ok {
				values[i] = string(b)
			}
		}
		resultRows = append(resultRows, values)
	}
	if resultRows == nil {
		resultRows = [][]interface{}{}
	}
	return &connector.QueryResult{Columns: columns, Rows: resultRows, RowCount: len(resultRows)}, rows.Err()
}

func (p *Plugin) Execute(db *sql.DB, sqlStr string) (int64, error) {
	result, err := db.Exec(sqlStr)
	if err != nil {
		return 0, fmt.Errorf("mssql: 执行失败: %w", err)
	}
	return result.RowsAffected()
}

// --- 表结构修改 ---

func (p *Plugin) AlterColumn(db *sql.DB, database, table string, col connector.ColumnChange) error {
	columns, err := p.GetColumns(db, database, table)
	if err != nil {
		return err
	}
	var current *connector.Column
	for i := range columns {
		if columns[i].Name == col.OldName {
			current = &columns[i]
			break
		}
	}
	if current == nil {
		return fmt.Errorf("列 %s 不存在", col.OldName)
	}

	newName := col.OldName
	if col.NewName != nil && *col.NewName != "" {
		_, err = db.Exec(fmt.Sprintf("EXEC sp_rename '%s.%s', '%s', 'COLUMN'",
			table, col.OldName, *col.NewName))
		if err != nil {
			return fmt.Errorf("mssql: 重命名列失败: %w", err)
		}
		newName = *col.NewName
	}

	if col.NewType != nil && *col.NewType != "" {
		_, err = db.Exec(fmt.Sprintf("ALTER TABLE [%s] ALTER COLUMN [%s] %s",
			table, newName, *col.NewType))
		if err != nil {
			return fmt.Errorf("mssql: 修改列类型失败: %w", err)
		}
	}

	if col.Nullable != nil {
		if *col.Nullable {
			_, err = db.Exec(fmt.Sprintf("ALTER TABLE [%s] ALTER COLUMN [%s] %s NULL",
				table, newName, current.DataType))
		} else {
			_, err = db.Exec(fmt.Sprintf("ALTER TABLE [%s] ALTER COLUMN [%s] %s NOT NULL",
				table, newName, current.DataType))
		}
		if err != nil {
			return fmt.Errorf("mssql: 修改可空性失败: %w", err)
		}
	}

	if col.Default != nil {
		if *col.Default == "" {
			_, err = db.Exec(fmt.Sprintf("ALTER TABLE [%s] DROP CONSTRAINT DF_%s_%s",
				table, table, col.OldName))
			if err != nil {
				// 忽略没有默认约束的错误
			}
		} else {
			_, err = db.Exec(fmt.Sprintf("ALTER TABLE [%s] ADD DEFAULT '%s' FOR [%s]",
				table, escapeString(*col.Default), newName))
			if err != nil {
				return fmt.Errorf("mssql: 修改默认值失败: %w", err)
			}
		}
	}

	if col.Comment != nil {
		commentVal := "NULL"
		if *col.Comment != "" {
			commentVal = fmt.Sprintf("'%s'", escapeString(*col.Comment))
		}
		_, err = db.Exec(fmt.Sprintf(`EXEC sys.sp_dropextendedproperty @name=N'MS_Description', @level0type=N'SCHEMA', @level0name=N'dbo', @level1type=N'TABLE', @level1name=N'%s', @level2type=N'COLUMN', @level2name=N'%s'`,
			table, newName))
		// 忽略不存在属性的错误
		if *col.Comment != "" {
			_, err = db.Exec(fmt.Sprintf(`EXEC sys.sp_addextendedproperty @name=N'MS_Description', @value=%s, @level0type=N'SCHEMA', @level0name=N'dbo', @level1type=N'TABLE', @level1name=N'%s', @level2type=N'COLUMN', @level2name=N'%s'`,
				commentVal, table, newName))
			if err != nil {
				return fmt.Errorf("mssql: 修改注释失败: %w", err)
			}
		}
	}

	return nil
}

func (p *Plugin) AddColumn(db *sql.DB, database, table string, col connector.ColumnDef) error {
	sqlStr := fmt.Sprintf("ALTER TABLE [%s] ADD [%s] %s", table, col.Name, col.Type)
	if !col.Nullable {
		sqlStr += " NOT NULL"
	}
	if col.Default != "" {
		sqlStr += fmt.Sprintf(" DEFAULT '%s'", escapeString(col.Default))
	}
	_, err := db.Exec(sqlStr)
	if err != nil {
		return fmt.Errorf("mssql: 添加列失败: %w", err)
	}
	if col.Comment != "" {
		_, _ = db.Exec(fmt.Sprintf(`EXEC sys.sp_addextendedproperty @name=N'MS_Description', @value=N'%s', @level0type=N'SCHEMA', @level0name=N'dbo', @level1type=N'TABLE', @level1name=N'%s', @level2type=N'COLUMN', @level2name=N'%s'`,
			escapeString(col.Comment), table, col.Name))
	}
	return nil
}

func (p *Plugin) DropColumn(db *sql.DB, database, table, column string) error {
	_, err := db.Exec(fmt.Sprintf("ALTER TABLE [%s] DROP COLUMN [%s]", table, column))
	if err != nil {
		return fmt.Errorf("mssql: 删除列失败: %w", err)
	}
	return nil
}

// --- 用户管理 ---

func (p *Plugin) ListUsers(db *sql.DB) ([]connector.DatabaseUser, error) {
	rows, err := db.Query("SELECT name FROM sys.sql_logins WHERE is_disabled = 0 ORDER BY name")
	if err != nil {
		return nil, fmt.Errorf("mssql: 列出用户失败: %w", err)
	}
	defer rows.Close()

	var users []connector.DatabaseUser
	for rows.Next() {
		var u connector.DatabaseUser
		if err := rows.Scan(&u.Name); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	if users == nil {
		return []connector.DatabaseUser{}, nil
	}
	return users, rows.Err()
}

func (p *Plugin) CreateUser(db *sql.DB, user, password string) error {
	eu := escapeIdent(user)
	_, err := db.Exec(fmt.Sprintf("CREATE LOGIN [%s] WITH PASSWORD = '%s'", eu, escapeString(password)))
	if err != nil {
		return fmt.Errorf("mssql: 创建登录失败: %w", err)
	}
	_, _ = db.Exec(fmt.Sprintf("CREATE USER [%s] FOR LOGIN [%s]", eu, eu))
	return nil
}

func (p *Plugin) DropUser(db *sql.DB, user string) error {
	eu := escapeIdent(user)
	db.Exec(fmt.Sprintf("DROP USER IF EXISTS [%s]", eu))
	_, err := db.Exec(fmt.Sprintf("DROP LOGIN [%s]", eu))
	if err != nil {
		return fmt.Errorf("mssql: 删除登录失败: %w", err)
	}
	return nil
}

func (p *Plugin) GetPrivileges(db *sql.DB, user string) ([]connector.UserPrivilege, error) {
	rows, err := db.Query(fmt.Sprintf(`SELECT class_desc, major_id, permission_name
		FROM sys.database_permissions WHERE grantee_principal_id = DATABASE_PRINCIPAL_ID('%s')`,
		escapeString(user)))
	if err != nil {
		return nil, fmt.Errorf("mssql: 查询权限失败: %w", err)
	}
	defer rows.Close()

	var privs []connector.UserPrivilege
	for rows.Next() {
		var classDesc, permName string
		var majorID int
		if err := rows.Scan(&classDesc, &majorID, &permName); err != nil {
			continue
		}
		if classDesc == "DATABASE" {
			privs = append(privs, connector.UserPrivilege{
				User: user, Database: "*", Table: "*", Privileges: []string{permName},
			})
		}
	}
	if privs == nil {
		return []connector.UserPrivilege{}, nil
	}
	return privs, rows.Err()
}

func (p *Plugin) GrantPrivilege(db *sql.DB, user, database, table string, privs []string) error {
	for _, priv := range privs {
		_, err := db.Exec(fmt.Sprintf("GRANT %s ON [%s].[%s] TO [%s]",
			priv, database, table, user))
		if err != nil {
			return fmt.Errorf("mssql: 授予权限失败: %w", err)
		}
	}
	return nil
}

func (p *Plugin) RevokePrivilege(db *sql.DB, user, database, table string, privs []string) error {
	for _, priv := range privs {
		_, err := db.Exec(fmt.Sprintf("REVOKE %s ON [%s].[%s] FROM [%s]",
			priv, database, table, user))
		if err != nil {
			return fmt.Errorf("mssql: 撤销权限失败: %w", err)
		}
	}
	return nil
}

// --- 自动补全 ---

func (p *Plugin) GetAutoCompleteData(db *sql.DB, database string) (*connector.AutoCompleteData, error) {
	data := &connector.AutoCompleteData{
		Keywords:  mssqlKeywords,
		Functions: mssqlFunctions,
	}
	tables, err := p.ListTables(db, database)
	if err == nil {
		for _, t := range tables {
			data.Tables = append(data.Tables, connector.TableRef{Name: t.Name, Schema: t.Schema})
			cols, colErr := p.GetColumns(db, database, t.Name)
			if colErr == nil {
				for _, c := range cols {
					data.Columns = append(data.Columns, connector.ColumnRef{Table: t.Name, Column: c.Name, Type: c.DataType})
				}
			}
		}
	}
	return data, nil
}

func (p *Plugin) GetVersion(db *sql.DB) (string, error) {
	var version string
	err := db.QueryRow("SELECT @@VERSION").Scan(&version)
	if err != nil {
		return "", fmt.Errorf("mssql: 获取版本失败: %w", err)
	}
	return "SQL Server " + version, nil
}

func escapeString(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

func escapeIdent(s string) string {
	return strings.ReplaceAll(s, "]", "]]")
}

var mssqlKeywords = []string{
	"SELECT", "FROM", "WHERE", "INSERT", "UPDATE", "DELETE", "CREATE", "ALTER", "DROP",
	"TABLE", "INDEX", "VIEW", "SCHEMA", "INTO", "VALUES", "SET", "JOIN", "LEFT", "RIGHT",
	"INNER", "OUTER", "CROSS", "FULL", "ON", "AND", "OR", "NOT", "IN", "EXISTS", "BETWEEN", "LIKE",
	"IS", "NULL", "AS", "ORDER", "BY", "ASC", "DESC", "GROUP", "HAVING",
	"UNION", "ALL", "DISTINCT", "CASE", "WHEN", "THEN", "ELSE", "END",
	"PRIMARY", "KEY", "FOREIGN", "REFERENCES", "CONSTRAINT", "UNIQUE", "CHECK", "DEFAULT",
	"TRUNCATE", "RENAME", "EXEC", "EXECUTE",
	"GRANT", "REVOKE", "DENY", "BEGIN", "COMMIT", "ROLLBACK", "TRANSACTION",
	"TOP", "PERCENT", "WITH", "NOLOCK",
	"OFFSET", "FETCH", "NEXT", "ROWS", "ONLY",
	"IDENTITY", "PRINT", "RAISERROR", "THROW",
	"GO", "USE", "DECLARE",
}

var mssqlFunctions = []string{
	"COUNT", "SUM", "AVG", "MIN", "MAX", "STRING_AGG",
	"GETDATE", "GETUTCDATE", "SYSDATETIME", "CURRENT_TIMESTAMP",
	"DATEADD", "DATEDIFF", "DATENAME", "DATEPART", "YEAR", "MONTH", "DAY",
	"CONCAT", "+", "SUBSTRING", "LEFT", "RIGHT", "LEN", "CHARINDEX",
	"UPPER", "LOWER", "TRIM", "LTRIM", "RTRIM", "REPLACE", "REVERSE",
	"ABS", "CEILING", "FLOOR", "ROUND", "%", "RAND",
	"ISNULL", "COALESCE", "NULLIF", "IIF", "CHOOSE",
	"CAST", "CONVERT", "TRY_CAST", "TRY_CONVERT",
	"NEWID", "ROW_NUMBER", "RANK", "DENSE_RANK", "LAG", "LEAD", "NTILE",
	"ISJSON", "JSON_VALUE", "JSON_QUERY",
}
