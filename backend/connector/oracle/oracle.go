package oracle

import (
	"database/sql"
	"fmt"
	"strings"

	go_ora "github.com/sijms/go-ora/v2"

	"dbhub-web/connector"
)

const PluginName = "oracle"

type Plugin struct{}

func New() *Plugin { return &Plugin{} }

func (p *Plugin) Open(cfg *connector.ConnectionConfig) (*sql.DB, error) {
	connStr := p.GetDSN(cfg)
	db, err := sql.Open("oracle", connStr)
	if err != nil {
		return nil, fmt.Errorf("oracle: 打开连接失败: %w", err)
	}
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(1)
	return db, nil
}

func (p *Plugin) Ping(db *sql.DB) error { return db.Ping() }

func (p *Plugin) GetDSN(cfg *connector.ConnectionConfig) string {
	host := cfg.Host
	port := cfg.Port
	if port == 0 {
		port = 1521
	}

	opts := map[string]string{}
	if cfg.SID != "" {
		opts["SID"] = cfg.SID
	}

	return go_ora.BuildUrl(host, port, cfg.Database, cfg.User, cfg.Password, opts)
}

// --- 数据库级 ---

func (p *Plugin) ListDatabases(db *sql.DB) ([]string, error) {
	// Oracle: 列出可访问的 schema (username)
	rows, err := db.Query("SELECT USERNAME FROM ALL_USERS ORDER BY USERNAME")
	if err != nil {
		return nil, fmt.Errorf("oracle: 列出 schema 失败: %w", err)
	}
	defer rows.Close()

	var schemas []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		schemas = append(schemas, name)
	}
	return schemas, rows.Err()
}

// --- 表级 ---

func (p *Plugin) SetDatabase(db *sql.DB, database string) error {
	if database == "" {
		return nil
	}
	_, err := db.Exec(fmt.Sprintf("ALTER SESSION SET CURRENT_SCHEMA = \"%s\"", escape(database)))
	return err
}

func (p *Plugin) ListTables(db *sql.DB, database string) ([]connector.Table, error) {
	query := fmt.Sprintf(`SELECT TABLE_NAME, 'TABLE' as TABLE_TYPE, NUM_ROWS FROM ALL_TABLES WHERE OWNER = '%s'
		UNION ALL
		SELECT VIEW_NAME, 'VIEW' as TABLE_TYPE, 0 FROM ALL_VIEWS WHERE OWNER = '%s'
		ORDER BY TABLE_NAME`, upper(database), upper(database))
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("oracle: 列出表失败: %w", err)
	}
	defer rows.Close()

	var tables []connector.Table
	for rows.Next() {
		var t connector.Table
		t.Schema = database
		var rowCount sql.NullInt64
		if err := rows.Scan(&t.Name, &t.Type, &rowCount); err != nil {
			return nil, err
		}
		if rowCount.Valid {
			t.RowCount = rowCount.Int64
		}
		tables = append(tables, t)
	}
	return tables, rows.Err()
}

func (p *Plugin) GetColumns(db *sql.DB, database, table string) ([]connector.Column, error) {
	query := fmt.Sprintf(`SELECT c.COLUMN_NAME, c.DATA_TYPE, c.NULLABLE, c.DATA_DEFAULT,
		CASE WHEN pk.COLUMN_NAME IS NOT NULL THEN 1 ELSE 0 END as IS_PK,
		COALESCE(cm.COMMENTS, '') as COMMENTS
	FROM ALL_TAB_COLUMNS c
	LEFT JOIN (
		SELECT cc.COLUMN_NAME FROM ALL_CONS_COLUMNS cc, ALL_CONSTRAINTS ac
		WHERE cc.OWNER = '%s' AND cc.TABLE_NAME = '%s'
		AND ac.CONSTRAINT_TYPE = 'P' AND ac.CONSTRAINT_NAME = cc.CONSTRAINT_NAME
	) pk ON c.COLUMN_NAME = pk.COLUMN_NAME
	LEFT JOIN ALL_COL_COMMENTS cm ON cm.OWNER = c.OWNER AND cm.TABLE_NAME = c.TABLE_NAME AND cm.COLUMN_NAME = c.COLUMN_NAME
	WHERE c.OWNER = '%s' AND c.TABLE_NAME = '%s'
	ORDER BY c.COLUMN_ID`, upper(database), upper(table), upper(database), upper(table))
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("oracle: 获取列信息失败: %w", err)
	}
	defer rows.Close()

	var columns []connector.Column
	for rows.Next() {
		var c connector.Column
		var isNullable string
		var defaultVal, comment sql.NullString
		var isPK int
		if err := rows.Scan(&c.Name, &c.DataType, &isNullable, &defaultVal, &isPK, &comment); err != nil {
			return nil, err
		}
		c.Nullable = isNullable == "Y"
		c.PrimaryKey = isPK == 1
		if defaultVal.Valid {
			c.DefaultVal = defaultVal.String
		}
		if comment.Valid {
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
	query := fmt.Sprintf(`SELECT i.INDEX_NAME, ic.COLUMN_NAME,
		CASE WHEN i.UNIQUENESS = 'UNIQUE' THEN 1 ELSE 0 END as IS_UNIQUE,
		i.INDEX_TYPE
	FROM ALL_INDEXES i
	JOIN ALL_IND_COLUMNS ic ON i.INDEX_NAME = ic.INDEX_NAME AND i.OWNER = ic.INDEX_OWNER
	WHERE i.OWNER = '%s' AND i.TABLE_NAME = '%s'
	ORDER BY i.INDEX_NAME, ic.COLUMN_POSITION`, upper(database), upper(table))
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("oracle: 获取索引信息失败: %w", err)
	}
	defer rows.Close()

	indexMap := make(map[string]*connector.Index)
	var indexOrder []string
	for rows.Next() {
		var indexName, colName, indexType string
		var isUnique int
		if err := rows.Scan(&indexName, &colName, &isUnique, &indexType); err != nil {
			return nil, err
		}
		if idx, ok := indexMap[indexName]; ok {
			idx.Columns = append(idx.Columns, colName)
		} else {
			indexMap[indexName] = &connector.Index{
				Name:    indexName,
				Columns: []string{colName},
				Unique:  isUnique == 1,
				Type:    indexType,
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
	query := fmt.Sprintf(`SELECT ac.CONSTRAINT_NAME, acc.COLUMN_NAME, acc2.TABLE_NAME, acc2.COLUMN_NAME
	FROM ALL_CONSTRAINTS ac
	JOIN ALL_CONS_COLUMNS acc ON ac.CONSTRAINT_NAME = acc.CONSTRAINT_NAME AND ac.OWNER = acc.OWNER
	JOIN ALL_CONS_COLUMNS acc2 ON ac.R_CONSTRAINT_NAME = acc2.CONSTRAINT_NAME
	WHERE ac.CONSTRAINT_TYPE = 'R' AND ac.OWNER = '%s' AND ac.TABLE_NAME = '%s'`,
		upper(database), upper(table))
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("oracle: 获取外键信息失败: %w", err)
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
	var ddl sql.NullString
	err := db.QueryRow(fmt.Sprintf(
		"SELECT DBMS_METADATA.GET_DDL('TABLE','%s','%s') FROM DUAL",
		upper(table), upper(database),
	)).Scan(&ddl)
	if err != nil {
		return "", fmt.Errorf("oracle: 获取DDL失败: %w", err)
	}
	if ddl.Valid {
		return ddl.String, nil
	}
	return "", nil
}

// --- 数据查询 ---

func (p *Plugin) Query(db *sql.DB, sqlStr string, limit, offset int) (*connector.QueryResult, error) {
	if limit > 0 || offset > 0 {
		sqlStr = strings.TrimSpace(strings.TrimRight(sqlStr, ";"))
		if offset > 0 {
			// Oracle 12c+: OFFSET ... FETCH NEXT
			// 兼容旧版: 双 ROWNUM 子查询
			sqlStr = fmt.Sprintf(
				"SELECT * FROM (SELECT a.*, ROWNUM rnum FROM (%s) a WHERE ROWNUM <= %d) WHERE rnum > %d",
				sqlStr, offset+limit, offset)
		} else {
			sqlStr = fmt.Sprintf("SELECT * FROM (%s) WHERE ROWNUM <= %d", sqlStr, limit)
		}
	}
	rows, err := db.Query(sqlStr)
	if err != nil {
		return nil, fmt.Errorf("oracle: 查询失败: %w", err)
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
		return 0, fmt.Errorf("oracle: 执行失败: %w", err)
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
		_, err = db.Exec(fmt.Sprintf(`ALTER TABLE "%s"."%s" RENAME COLUMN "%s" TO "%s"`,
			database, table, col.OldName, *col.NewName))
		if err != nil {
			return fmt.Errorf("oracle: 重命名列失败: %w", err)
		}
		newName = *col.NewName
	}

	if col.NewType != nil && *col.NewType != "" {
		_, err = db.Exec(fmt.Sprintf(`ALTER TABLE "%s"."%s" MODIFY "%s" %s`,
			database, table, newName, *col.NewType))
		if err != nil {
			return fmt.Errorf("oracle: 修改列类型失败: %w", err)
		}
	}

	if col.Nullable != nil {
		if *col.Nullable {
			_, err = db.Exec(fmt.Sprintf(`ALTER TABLE "%s"."%s" MODIFY "%s" NULL`,
				database, table, newName))
		} else {
			_, err = db.Exec(fmt.Sprintf(`ALTER TABLE "%s"."%s" MODIFY "%s" NOT NULL`,
				database, table, newName))
		}
		if err != nil {
			return fmt.Errorf("oracle: 修改可空性失败: %w", err)
		}
	}

	if col.Default != nil {
		if *col.Default == "" {
			_, err = db.Exec(fmt.Sprintf(`ALTER TABLE "%s"."%s" MODIFY "%s" DEFAULT NULL`,
				database, table, newName))
		} else {
			_, err = db.Exec(fmt.Sprintf(`ALTER TABLE "%s"."%s" MODIFY "%s" DEFAULT '%s'`,
				database, table, newName, escape(*col.Default)))
		}
		if err != nil {
			return fmt.Errorf("oracle: 修改默认值失败: %w", err)
		}
	}

	if col.Comment != nil {
		commentVal := "''"
		if *col.Comment != "" {
			commentVal = fmt.Sprintf("'%s'", escape(*col.Comment))
		}
		_, err = db.Exec(fmt.Sprintf(`COMMENT ON COLUMN "%s"."%s"."%s" IS %s`,
			database, table, newName, commentVal))
		if err != nil {
			return fmt.Errorf("oracle: 修改注释失败: %w", err)
		}
	}

	return nil
}

func (p *Plugin) AddColumn(db *sql.DB, database, table string, col connector.ColumnDef) error {
	nullable := ""
	if !col.Nullable {
		nullable = " NOT NULL"
	}
	sqlStr := fmt.Sprintf(`ALTER TABLE "%s"."%s" ADD "%s" %s%s`,
		database, table, col.Name, col.Type, nullable)
	if col.Default != "" {
		sqlStr += fmt.Sprintf(" DEFAULT '%s'", escape(col.Default))
	}
	_, err := db.Exec(sqlStr)
	if err != nil {
		return fmt.Errorf("oracle: 添加列失败: %w", err)
	}
	if col.Comment != "" {
		_, _ = db.Exec(fmt.Sprintf(`COMMENT ON COLUMN "%s"."%s"."%s" IS '%s'`,
			database, table, col.Name, escape(col.Comment)))
	}
	return nil
}

func (p *Plugin) DropColumn(db *sql.DB, database, table, column string) error {
	_, err := db.Exec(fmt.Sprintf(`ALTER TABLE "%s"."%s" DROP COLUMN "%s"`,
		database, table, column))
	if err != nil {
		return fmt.Errorf("oracle: 删除列失败: %w", err)
	}
	return nil
}

// --- 用户管理 ---

func (p *Plugin) ListUsers(db *sql.DB) ([]connector.DatabaseUser, error) {
	rows, err := db.Query("SELECT USERNAME FROM DBA_USERS ORDER BY USERNAME")
	if err != nil {
		// 如果没有 DBA 权限，尝试 ALL_USERS
		rows, err = db.Query("SELECT USERNAME FROM ALL_USERS ORDER BY USERNAME")
		if err != nil {
			return nil, fmt.Errorf("oracle: 列出用户失败: %w", err)
		}
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
	_, err := db.Exec(fmt.Sprintf(`CREATE USER "%s" IDENTIFIED BY "%s"`, user, escape(password)))
	if err != nil {
		return fmt.Errorf("oracle: 创建用户失败: %w", err)
	}
	// 赋予基本权限
	db.Exec(fmt.Sprintf(`GRANT CREATE SESSION TO "%s"`, user))
	return nil
}

func (p *Plugin) DropUser(db *sql.DB, user string) error {
	_, err := db.Exec(fmt.Sprintf(`DROP USER "%s" CASCADE`, user))
	if err != nil {
		return fmt.Errorf("oracle: 删除用户失败: %w", err)
	}
	return nil
}

func (p *Plugin) GetPrivileges(db *sql.DB, user string) ([]connector.UserPrivilege, error) {
	rows, err := db.Query(fmt.Sprintf(`SELECT GRANTEE, TABLE_NAME, PRIVILEGE
		FROM DBA_TAB_PRIVS WHERE GRANTEE = '%s' ORDER BY TABLE_NAME`, upper(user)))
	if err != nil {
		return nil, fmt.Errorf("oracle: 查询权限失败: %w", err)
	}
	defer rows.Close()

	privMap := make(map[string]*connector.UserPrivilege)
	for rows.Next() {
		var grantee, tableName, privilege string
		if err := rows.Scan(&grantee, &tableName, &privilege); err != nil {
			continue
		}
		if up, ok := privMap[tableName]; ok {
			up.Privileges = append(up.Privileges, privilege)
		} else {
			privMap[tableName] = &connector.UserPrivilege{
				User:       user,
				Table:      tableName,
				Privileges: []string{privilege},
			}
		}
	}
	var result []connector.UserPrivilege
	for _, v := range privMap {
		result = append(result, *v)
	}
	if result == nil {
		return []connector.UserPrivilege{}, nil
	}
	return result, rows.Err()
}

func (p *Plugin) GrantPrivilege(db *sql.DB, user, database, table string, privs []string) error {
	for _, priv := range privs {
		_, err := db.Exec(fmt.Sprintf(`GRANT %s ON "%s"."%s" TO "%s"`, priv, database, table, user))
		if err != nil {
			return fmt.Errorf("oracle: 授予权限失败: %w", err)
		}
	}
	return nil
}

func (p *Plugin) RevokePrivilege(db *sql.DB, user, database, table string, privs []string) error {
	for _, priv := range privs {
		_, err := db.Exec(fmt.Sprintf(`REVOKE %s ON "%s"."%s" FROM "%s"`, priv, database, table, user))
		if err != nil {
			return fmt.Errorf("oracle: 撤销权限失败: %w", err)
		}
	}
	return nil
}

// --- 自动补全 ---

func (p *Plugin) GetAutoCompleteData(db *sql.DB, database string) (*connector.AutoCompleteData, error) {
	data := &connector.AutoCompleteData{
		Keywords:  oracleKeywords,
		Functions: oracleFunctions,
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
	err := db.QueryRow("SELECT BANNER FROM V$VERSION WHERE ROWNUM = 1").Scan(&version)
	if err != nil {
		return "", fmt.Errorf("oracle: 获取版本失败: %w", err)
	}
	return version, nil
}

func escape(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

func upper(s string) string {
	return strings.ToUpper(s)
}

var oracleKeywords = []string{
	"SELECT", "FROM", "WHERE", "INSERT", "UPDATE", "DELETE", "CREATE", "ALTER", "DROP",
	"TABLE", "INDEX", "VIEW", "SCHEMA", "INTO", "VALUES", "SET", "JOIN", "LEFT", "RIGHT",
	"INNER", "OUTER", "CROSS", "FULL", "ON", "AND", "OR", "NOT", "IN", "EXISTS", "BETWEEN", "LIKE",
	"IS", "NULL", "AS", "ORDER", "BY", "ASC", "DESC", "GROUP", "HAVING",
	"UNION", "ALL", "DISTINCT", "CASE", "WHEN", "THEN", "ELSE", "END",
	"PRIMARY", "KEY", "FOREIGN", "REFERENCES", "CONSTRAINT", "UNIQUE", "CHECK", "DEFAULT",
	"TRUNCATE", "RENAME", "EXPLAIN", "PLAN",
	"GRANT", "REVOKE", "COMMIT", "ROLLBACK", "SAVEPOINT",
	"SEQUENCE", "SYNONYM", "PACKAGE", "PROCEDURE", "FUNCTION", "TRIGGER",
	"DECLARE", "BEGIN", "EXCEPTION", "LOOP", "FOR", "WHILE", "CURSOR", "RETURN",
	"ROWNUM", "ROWID", "DUAL", "CONNECT", "LEVEL", "PRIOR",
	"FETCH", "FIRST", "NEXT", "ROWS", "ONLY", "WITH",
}

var oracleFunctions = []string{
	"COUNT", "SUM", "AVG", "MIN", "MAX", "LISTAGG",
	"SYSDATE", "CURRENT_DATE", "CURRENT_TIMESTAMP", "SYSTIMESTAMP",
	"TO_CHAR", "TO_DATE", "TO_NUMBER", "TO_TIMESTAMP",
	"TRUNC", "ROUND", "FLOOR", "CEIL", "MOD", "ABS", "SIGN",
	"UPPER", "LOWER", "INITCAP", "SUBSTR", "INSTR", "LENGTH", "REPLACE", "TRIM", "LTRIM", "RTRIM",
	"CONCAT", "||", "LPAD", "RPAD", "REGEXP_LIKE", "REGEXP_REPLACE", "REGEXP_SUBSTR",
	"ADD_MONTHS", "MONTHS_BETWEEN", "LAST_DAY", "NEXT_DAY",
	"NVL", "NVL2", "COALESCE", "NULLIF", "DECODE",
	"ROW_NUMBER", "RANK", "DENSE_RANK", "LAG", "LEAD", "NTILE",
}
