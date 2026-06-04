package mysql

import (
	"database/sql"
	"fmt"
	"net/url"
	"strings"

	_ "github.com/go-sql-driver/mysql"

	"dbhub-web/connector"
)

const PluginName = "mysql"

type Plugin struct{}

func New() *Plugin { return &Plugin{} }

// --- 连接 ---

func (p *Plugin) Open(cfg *connector.ConnectionConfig) (*sql.DB, error) {
	dsn := p.GetDSN(cfg)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("mysql: 打开连接失败: %w", err)
	}
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	return db, nil
}

func (p *Plugin) Ping(db *sql.DB) error {
	return db.Ping()
}

func (p *Plugin) GetDSN(cfg *connector.ConnectionConfig) string {
	host := cfg.Host
	port := cfg.Port
	if port == 0 {
		port = 3306
	}
	params := url.Values{}
	params.Set("charset", "utf8mb4")
	params.Set("parseTime", "true")
	params.Set("timeout", "10s")
	params.Set("readTimeout", "30s")

	dbName := cfg.Database
	if dbName == "" {
		dbName = "/" // MySQL driver requires at least a slash
	} else {
		dbName = "/" + url.QueryEscape(dbName)
	}
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)%s?%s",
		url.QueryEscape(cfg.User),
		url.QueryEscape(cfg.Password),
		host, port,
		dbName,
		params.Encode(),
	)
	return dsn
}

// --- 数据库级 ---

func (p *Plugin) ListDatabases(db *sql.DB) ([]string, error) {
	rows, err := db.Query("SHOW DATABASES")
	if err != nil {
		return nil, fmt.Errorf("mysql: 列出数据库失败: %w", err)
	}
	defer rows.Close()

	var databases []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		// 过滤系统数据库
		if name == "information_schema" || name == "mysql" || name == "performance_schema" || name == "sys" {
			continue
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
	_, err := db.Exec(fmt.Sprintf("USE `%s`", escapeIdent(database)))
	return err
}

func (p *Plugin) ListTables(db *sql.DB, database string) ([]connector.Table, error) {
	query := fmt.Sprintf("SELECT TABLE_NAME, TABLE_TYPE, TABLE_ROWS FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = '%s' ORDER BY TABLE_NAME",
		escapeString(database))
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("mysql: 列出表失败: %w", err)
	}
	defer rows.Close()

	var tables []connector.Table
	for rows.Next() {
		var t connector.Table
		t.Schema = database
		var rowCount sql.NullInt64
		var tableType string
		if err := rows.Scan(&t.Name, &tableType, &rowCount); err != nil {
			return nil, err
		}
		if tableType == "BASE TABLE" {
			t.Type = "TABLE"
		} else if tableType == "VIEW" {
			t.Type = "VIEW"
		}
		if rowCount.Valid {
			t.RowCount = rowCount.Int64
		}
		tables = append(tables, t)
	}
	return tables, rows.Err()
}

func (p *Plugin) GetColumns(db *sql.DB, database, table string) ([]connector.Column, error) {
	query := fmt.Sprintf(`SELECT COLUMN_NAME, DATA_TYPE, IS_NULLABLE, COLUMN_DEFAULT, COLUMN_KEY, COLUMN_COMMENT
FROM INFORMATION_SCHEMA.COLUMNS
WHERE TABLE_SCHEMA = '%s' AND TABLE_NAME = '%s'
ORDER BY ORDINAL_POSITION`, escapeString(database), escapeString(table))
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("mysql: 获取列信息失败: %w", err)
	}
	defer rows.Close()

	var columns []connector.Column
	for rows.Next() {
		var c connector.Column
		var isNullable, colKey string
		var defaultVal, comment sql.NullString
		if err := rows.Scan(&c.Name, &c.DataType, &isNullable, &defaultVal, &colKey, &comment); err != nil {
			return nil, err
		}
		c.Nullable = isNullable == "YES"
		if defaultVal.Valid {
			c.DefaultVal = defaultVal.String
		}
		c.PrimaryKey = colKey == "PRI"
		if comment.Valid {
			c.Comment = comment.String
		}
		columns = append(columns, c)
	}
	return columns, rows.Err()
}

func (p *Plugin) GetIndexes(db *sql.DB, database, table string) ([]connector.Index, error) {
	query := fmt.Sprintf(`SELECT INDEX_NAME, COLUMN_NAME, NON_UNIQUE, INDEX_TYPE
FROM INFORMATION_SCHEMA.STATISTICS
WHERE TABLE_SCHEMA = '%s' AND TABLE_NAME = '%s'
ORDER BY INDEX_NAME, SEQ_IN_INDEX`, escapeString(database), escapeString(table))
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("mysql: 获取索引信息失败: %w", err)
	}
	defer rows.Close()

	indexMap := make(map[string]*connector.Index)
	var indexOrder []string

	for rows.Next() {
		var indexName, colName, indexType string
		var nonUnique int
		if err := rows.Scan(&indexName, &colName, &nonUnique, &indexType); err != nil {
			return nil, err
		}
		if idx, ok := indexMap[indexName]; ok {
			idx.Columns = append(idx.Columns, colName)
		} else {
			indexMap[indexName] = &connector.Index{
				Name:    indexName,
				Columns: []string{colName},
				Unique:  nonUnique == 0,
				Type:    indexType,
			}
			indexOrder = append(indexOrder, indexName)
		}
	}

	var indexes []connector.Index
	for _, name := range indexOrder {
		indexes = append(indexes, *indexMap[name])
	}
	if len(indexes) == 0 {
		return []connector.Index{}, nil
	}
	return indexes, rows.Err()
}

func (p *Plugin) GetForeignKeys(db *sql.DB, database, table string) ([]connector.ForeignKey, error) {
	query := fmt.Sprintf(`SELECT CONSTRAINT_NAME, COLUMN_NAME, REFERENCED_TABLE_NAME, REFERENCED_COLUMN_NAME
FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE
WHERE TABLE_SCHEMA = '%s' AND TABLE_NAME = '%s' AND REFERENCED_TABLE_NAME IS NOT NULL`,
		escapeString(database), escapeString(table))
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("mysql: 获取外键信息失败: %w", err)
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
	query := fmt.Sprintf("SHOW CREATE TABLE `%s`.`%s`",
		escapeIdent(database), escapeIdent(table))
	var name, ddl string
	err := db.QueryRow(query).Scan(&name, &ddl)
	if err != nil {
		return "", fmt.Errorf("mysql: 获取DDL失败: %w", err)
	}
	return ddl, nil
}

// --- 数据查询 ---

func (p *Plugin) Query(db *sql.DB, sqlStr string, limit, offset int) (*connector.QueryResult, error) {
	if limit > 0 {
		sqlStr = fmt.Sprintf("%s LIMIT %d OFFSET %d", strings.TrimSpace(sqlStr), limit, offset)
	}
	rows, err := db.Query(sqlStr)
	if err != nil {
		return nil, fmt.Errorf("mysql: 查询失败: %w", err)
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
		// 将 []byte 转为 string，避免 JSON 序列化为 base64
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

	return &connector.QueryResult{
		Columns:  columns,
		Rows:     resultRows,
		RowCount: len(resultRows),
	}, rows.Err()
}

func (p *Plugin) Execute(db *sql.DB, sqlStr string) (int64, error) {
	result, err := db.Exec(sqlStr)
	if err != nil {
		return 0, fmt.Errorf("mysql: 执行失败: %w", err)
	}
	return result.RowsAffected()
}

// --- 表结构修改 ---

func (p *Plugin) AlterColumn(db *sql.DB, database, table string, col connector.ColumnChange) error {
	// 先获取当前列定义
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
		newName = *col.NewName
	}
	dataType := current.DataType
	if col.NewType != nil && *col.NewType != "" {
		dataType = *col.NewType
	}

	nullable := "NULL"
	if current.Nullable {
		nullable = "NULL"
	} else {
		nullable = "NOT NULL"
	}
	if col.Nullable != nil {
		if *col.Nullable {
			nullable = "NULL"
		} else {
			nullable = "NOT NULL"
		}
	}

	sqlStr := fmt.Sprintf("ALTER TABLE `%s`.`%s` CHANGE `%s` `%s` %s %s",
		escapeString(database), escapeString(table),
		col.OldName, newName, dataType, nullable)

	if col.Default != nil && *col.Default != "" {
		sqlStr += fmt.Sprintf(" DEFAULT '%s'", escapeString(*col.Default))
	}
	if col.Comment != nil {
		if *col.Comment == "" {
			sqlStr += " COMMENT ''"
		} else {
			sqlStr += fmt.Sprintf(" COMMENT '%s'", escapeString(*col.Comment))
		}
	}

	_, err = db.Exec(sqlStr)
	if err != nil {
		return fmt.Errorf("mysql: 修改列失败: %w", err)
	}
	return nil
}

func (p *Plugin) AddColumn(db *sql.DB, database, table string, col connector.ColumnDef) error {
	nullable := ""
	if col.Nullable {
		nullable = "NULL"
	} else {
		nullable = "NOT NULL"
	}

	sqlStr := fmt.Sprintf("ALTER TABLE `%s`.`%s` ADD COLUMN `%s` %s %s",
		escapeString(database), escapeString(table),
		col.Name, col.Type, nullable)

	if col.Default != "" {
		sqlStr += fmt.Sprintf(" DEFAULT '%s'", escapeString(col.Default))
	}
	if col.Comment != "" {
		sqlStr += fmt.Sprintf(" COMMENT '%s'", escapeString(col.Comment))
	}

	_, err := db.Exec(sqlStr)
	if err != nil {
		return fmt.Errorf("mysql: 添加列失败: %w", err)
	}
	return nil
}

func (p *Plugin) DropColumn(db *sql.DB, database, table, column string) error {
	sqlStr := fmt.Sprintf("ALTER TABLE `%s`.`%s` DROP COLUMN `%s`",
		escapeString(database), escapeString(table), column)
	_, err := db.Exec(sqlStr)
	if err != nil {
		return fmt.Errorf("mysql: 删除列失败: %w", err)
	}
	return nil
}

// --- 用户管理 ---

func (p *Plugin) ListUsers(db *sql.DB) ([]connector.DatabaseUser, error) {
	rows, err := db.Query("SELECT User, Host FROM mysql.user ORDER BY User, Host")
	if err != nil {
		return nil, fmt.Errorf("mysql: 列出用户失败: %w", err)
	}
	defer rows.Close()

	var users []connector.DatabaseUser
	for rows.Next() {
		var u connector.DatabaseUser
		if err := rows.Scan(&u.Name, &u.Host); err != nil {
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
	// MySQL 格式: CREATE USER 'user'@'%' IDENTIFIED BY 'password'
	query := fmt.Sprintf("CREATE USER '%s'@'%%' IDENTIFIED BY '%s'",
		escapeString(user), escapeString(password))
	_, err := db.Exec(query)
	if err != nil {
		return fmt.Errorf("mysql: 创建用户失败: %w", err)
	}
	return nil
}

func (p *Plugin) DropUser(db *sql.DB, user string) error {
	query := fmt.Sprintf("DROP USER '%s'@'%%'", escapeString(user))
	_, err := db.Exec(query)
	if err != nil {
		return fmt.Errorf("mysql: 删除用户失败: %w", err)
	}
	return nil
}

func (p *Plugin) GetPrivileges(db *sql.DB, user string) ([]connector.UserPrivilege, error) {
	var privs []connector.UserPrivilege

	// 库级别权限 (mysql.db)
	dbQuery := fmt.Sprintf("SELECT Db, Select_priv, Insert_priv, Update_priv, Delete_priv, Create_priv, Drop_priv, Grant_priv FROM mysql.db WHERE User='%s'", escapeString(user))
	dbRows, err := db.Query(dbQuery)
	if err != nil {
		return nil, fmt.Errorf("mysql: 查询库权限失败: %w", err)
	}
	defer dbRows.Close()
	for dbRows.Next() {
		var dbName, s, i, u, d, c, dr, g string
		if err := dbRows.Scan(&dbName, &s, &i, &u, &d, &c, &dr, &g); err != nil {
			return nil, err
		}
		up := connector.UserPrivilege{User: user, Database: dbName, Table: "*"}
		for p, v := range map[string]string{"SELECT": s, "INSERT": i, "UPDATE": u, "DELETE": d, "CREATE": c, "DROP": dr, "GRANT": g} {
			if v == "Y" {
				up.Privileges = append(up.Privileges, p)
			}
		}
		privs = append(privs, up)
	}

	// 表级别权限 (mysql.tables_priv)
	tableQuery := fmt.Sprintf("SELECT Db, Table_name, Table_priv FROM mysql.tables_priv WHERE User='%s'", escapeString(user))
	tableRows, err := db.Query(tableQuery)
	if err == nil {
		defer tableRows.Close()
		for tableRows.Next() {
			var dbName, tableName, privStr string
			if err := tableRows.Scan(&dbName, &tableName, &privStr); err != nil {
				continue
			}
			up := connector.UserPrivilege{User: user, Database: dbName, Table: tableName}
			for _, p := range strings.Split(privStr, ",") {
				up.Privileges = append(up.Privileges, strings.TrimSpace(p))
			}
			privs = append(privs, up)
		}
	}

	if privs == nil {
		return []connector.UserPrivilege{}, nil
	}
	return privs, nil
}

func (p *Plugin) GrantPrivilege(db *sql.DB, user, database, table string, privs []string) error {
	for _, priv := range privs {
		query := fmt.Sprintf("GRANT %s ON `%s`.`%s` TO '%s'@'%%'",
			priv, escapeString(database), escapeString(table), escapeString(user))
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("mysql: 授予权限失败: %w", err)
		}
	}
	return nil
}

func (p *Plugin) RevokePrivilege(db *sql.DB, user, database, table string, privs []string) error {
	for _, priv := range privs {
		query := fmt.Sprintf("REVOKE %s ON `%s`.`%s` FROM '%s'@'%%'",
			priv, escapeString(database), escapeString(table), escapeString(user))
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("mysql: 撤销权限失败: %w", err)
		}
	}
	return nil
}

// --- 自动补全 ---

func (p *Plugin) GetAutoCompleteData(db *sql.DB, database string) (*connector.AutoCompleteData, error) {
	data := &connector.AutoCompleteData{
		Keywords:  mysqlKeywords,
		Functions: mysqlFunctions,
	}

	tables, err := p.ListTables(db, database)
	if err == nil {
		for _, t := range tables {
			data.Tables = append(data.Tables, connector.TableRef{Name: t.Name})
			// 获取每个表的列
			cols, colErr := p.GetColumns(db, database, t.Name)
			if colErr == nil {
				for _, c := range cols {
					data.Columns = append(data.Columns, connector.ColumnRef{
						Table:  t.Name,
						Column: c.Name,
						Type:   c.DataType,
					})
				}
			}
		}
	}

	return data, nil
}

// --- 元数据 ---

func (p *Plugin) GetVersion(db *sql.DB) (string, error) {
	var version string
	err := db.QueryRow("SELECT VERSION()").Scan(&version)
	if err != nil {
		return "", fmt.Errorf("mysql: 获取版本失败: %w", err)
	}
	return "MySQL " + version, nil
}

// --- 工具函数 ---

func escapeIdent(s string) string {
	return strings.ReplaceAll(s, "`", "``")
}

func escapeString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	return strings.ReplaceAll(s, "'", "\\'")
}

// SQL 关键字和函数（静态定义）

var mysqlKeywords = []string{
	"SELECT", "FROM", "WHERE", "INSERT", "UPDATE", "DELETE", "CREATE", "ALTER", "DROP",
	"TABLE", "INDEX", "VIEW", "DATABASE", "INTO", "VALUES", "SET", "JOIN", "LEFT", "RIGHT",
	"INNER", "OUTER", "CROSS", "ON", "AND", "OR", "NOT", "IN", "EXISTS", "BETWEEN", "LIKE",
	"IS", "NULL", "AS", "ORDER", "BY", "ASC", "DESC", "GROUP", "HAVING", "LIMIT", "OFFSET",
	"UNION", "ALL", "DISTINCT", "CASE", "WHEN", "THEN", "ELSE", "END", "IF", "PRIMARY",
	"KEY", "FOREIGN", "REFERENCES", "CONSTRAINT", "UNIQUE", "CHECK", "DEFAULT", "AUTO_INCREMENT",
	"COMMENT", "ENGINE", "CHARSET", "COLLATE", "TRUNCATE", "RENAME", "SHOW", "DESCRIBE",
	"EXPLAIN", "USE", "GRANT", "REVOKE", "FLUSH", "BEGIN", "COMMIT", "ROLLBACK", "START",
	"TRANSACTION", "DECLARE", "CURSOR", "OPEN", "FETCH", "CLOSE", "PROCEDURE", "FUNCTION",
	"TRIGGER", "EVENT", "TEMPORARY", "IF", "REPLACE",
}

var mysqlFunctions = []string{
	"COUNT", "SUM", "AVG", "MIN", "MAX", "GROUP_CONCAT",
	"NOW", "CURDATE", "CURTIME", "DATE", "TIME", "YEAR", "MONTH", "DAY", "HOUR", "MINUTE", "SECOND",
	"DATE_FORMAT", "TIMESTAMPDIFF", "DATEDIFF", "DATE_ADD", "DATE_SUB",
	"CONCAT", "CONCAT_WS", "SUBSTRING", "SUBSTR", "LEFT", "RIGHT", "LENGTH", "CHAR_LENGTH",
	"UPPER", "LOWER", "TRIM", "LTRIM", "RTRIM", "REPLACE", "REVERSE", "LOCATE", "INSTR",
	"ABS", "CEIL", "CEILING", "FLOOR", "ROUND", "MOD", "RAND",
	"IFNULL", "COALESCE", "NULLIF", "IF",
	"CAST", "CONVERT", "BINARY",
	"MD5", "SHA1", "SHA2", "UUID",
	"JSON_EXTRACT", "JSON_ARRAY", "JSON_OBJECT",
	"ROW_NUMBER", "RANK", "DENSE_RANK", "LAG", "LEAD",
}
