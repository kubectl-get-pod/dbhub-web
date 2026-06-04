package postgres

import (
	"database/sql"
	"fmt"
	"net/url"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"

	"dbhub-web/connector"
)

const PluginName = "postgres"

type Plugin struct{}

func New() *Plugin { return &Plugin{} }

func (p *Plugin) Open(cfg *connector.ConnectionConfig) (*sql.DB, error) {
	dsn := p.GetDSN(cfg)
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres: open connection failed: %w", err)
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
		port = 5432
	}
	sslMode := cfg.SSLMode
	if sslMode == "" {
		sslMode = "disable"
	}
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		url.QueryEscape(cfg.User), url.QueryEscape(cfg.Password),
		host, port, url.QueryEscape(cfg.Database), sslMode)
}

func (p *Plugin) ListDatabases(db *sql.DB) ([]string, error) {
	rows, err := db.Query("SELECT datname FROM pg_database WHERE datistemplate = false ORDER BY datname")
	if err != nil {
		return nil, fmt.Errorf("postgres: list databases failed: %w", err)
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

func (p *Plugin) SetDatabase(db *sql.DB, database string) error {
	if database == "" {
		return nil
	}
	_, err := db.Exec(fmt.Sprintf("SET search_path TO \"%s\"", escIdent(database)))
	return err
}

func (p *Plugin) ListTables(db *sql.DB, database string) ([]connector.Table, error) {
	schema := database
	if schema == "" {
		schema = "public"
	}
	query := fmt.Sprintf(`SELECT tablename, 'TABLE' as table_type,
		COALESCE((SELECT n_live_tup FROM pg_stat_user_tables WHERE schemaname='%s' AND relname=tablename), 0)
		FROM pg_tables WHERE schemaname='%s'
		UNION ALL
		SELECT viewname, 'VIEW' as table_type, 0
		FROM pg_views WHERE schemaname='%s'
		ORDER BY tablename`, esc(schema), esc(schema), esc(schema))
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("postgres: list tables failed: %w", err)
	}
	defer rows.Close()
	var tables []connector.Table
	for rows.Next() {
		var t connector.Table
		t.Schema = schema
		if err := rows.Scan(&t.Name, &t.Type, &t.RowCount); err != nil {
			return nil, err
		}
		tables = append(tables, t)
	}
	return tables, rows.Err()
}

func (p *Plugin) GetColumns(db *sql.DB, database, table string) ([]connector.Column, error) {
	schema := database
	if schema == "" {
		schema = "public"
	}
	query := fmt.Sprintf(`SELECT c.column_name, c.data_type, c.is_nullable, c.column_default,
		CASE WHEN pk.column_name IS NOT NULL THEN true ELSE false END as is_pk,
		COALESCE(pg_catalog.col_description(
			(SELECT c2.oid FROM pg_class c2 JOIN pg_namespace n ON c2.relnamespace=n.oid
			 WHERE n.nspname='%s' AND c2.relname='%s'), c.ordinal_position), '')
	FROM information_schema.columns c
	LEFT JOIN (
		SELECT ku.column_name FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage ku ON tc.constraint_name=ku.constraint_name
		WHERE tc.constraint_type='PRIMARY KEY' AND tc.table_schema='%s' AND tc.table_name='%s'
	) pk ON c.column_name=pk.column_name
	WHERE c.table_schema='%s' AND c.table_name='%s'
	ORDER BY c.ordinal_position`, esc(schema), esc(table), esc(schema), esc(table), esc(schema), esc(table))
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("postgres: get columns failed: %w", err)
	}
	defer rows.Close()
	var columns []connector.Column
	for rows.Next() {
		var c connector.Column
		var isNullable string
		var defaultVal, comment sql.NullString
		if err := rows.Scan(&c.Name, &c.DataType, &isNullable, &defaultVal, &c.PrimaryKey, &comment); err != nil {
			return nil, err
		}
		c.Nullable = isNullable == "YES"
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
	schema := database
	if schema == "" {
		schema = "public"
	}
	query := fmt.Sprintf(`SELECT i.relname, a.attname, ix.indisunique, am.amname
		FROM pg_class t, pg_class i, pg_index ix, pg_attribute a, pg_am am, pg_namespace n
		WHERE t.oid=ix.indrelid AND i.oid=ix.indexrelid AND a.attrelid=t.oid
		AND a.attnum=ANY(ix.indkey) AND t.relkind='r'
		AND i.relam=am.oid AND t.relnamespace=n.oid
		AND n.nspname='%s' AND t.relname='%s'
		ORDER BY i.relname, a.attnum`, esc(schema), esc(table))
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("postgres: get indexes failed: %w", err)
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
				Name: indexName, Columns: []string{colName},
				Unique: isUnique, Type: strings.ToUpper(indexType),
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
	schema := database
	if schema == "" {
		schema = "public"
	}
	query := fmt.Sprintf(`SELECT tc.constraint_name, kcu.column_name,
		ccu.table_name, ccu.column_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu ON tc.constraint_name=kcu.constraint_name
		JOIN information_schema.constraint_column_usage ccu ON ccu.constraint_name=tc.constraint_name
		WHERE tc.constraint_type='FOREIGN KEY' AND tc.table_schema='%s' AND tc.table_name='%s'`,
		esc(schema), esc(table))
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("postgres: get foreign keys failed: %w", err)
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
	sb.WriteString(fmt.Sprintf("CREATE TABLE \"%s\" (\n", escIdent(table)))
	for i, c := range cols {
		sb.WriteString(fmt.Sprintf("  \"%s\" %s", escIdent(c.Name), c.DataType))
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
	sb.WriteString(");")
	return sb.String(), nil
}

func (p *Plugin) Query(db *sql.DB, sqlStr string, limit, offset int) (*connector.QueryResult, error) {
	if limit > 0 {
		sqlStr = fmt.Sprintf("%s LIMIT %d OFFSET %d",
			strings.TrimSpace(strings.TrimRight(sqlStr, ";")), limit, offset)
	}
	rows, err := db.Query(sqlStr)
	if err != nil {
		return nil, fmt.Errorf("postgres: query failed: %w", err)
	}
	defer rows.Close()
	columns, _ := rows.Columns()
	var resultRows [][]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		rows.Scan(valuePtrs...)
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
		return 0, fmt.Errorf("postgres: execute failed: %w", err)
	}
	return result.RowsAffected()
}

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
		return fmt.Errorf("column %s does not exist", col.OldName)
	}
	newName := col.OldName
	if col.NewName != nil && *col.NewName != "" {
		_, err = db.Exec(fmt.Sprintf("ALTER TABLE \"%s\" RENAME COLUMN \"%s\" TO \"%s\"",
			escIdent(table), escIdent(col.OldName), escIdent(*col.NewName)))
		if err != nil {
			return fmt.Errorf("postgres: rename column failed: %w", err)
		}
		newName = *col.NewName
	}
	if col.NewType != nil && *col.NewType != "" {
		_, err = db.Exec(fmt.Sprintf("ALTER TABLE \"%s\" ALTER COLUMN \"%s\" TYPE %s",
			escIdent(table), escIdent(newName), *col.NewType))
		if err != nil {
			return fmt.Errorf("postgres: alter column type failed: %w", err)
		}
	}
	if col.Nullable != nil {
		if *col.Nullable {
			_, err = db.Exec(fmt.Sprintf("ALTER TABLE \"%s\" ALTER COLUMN \"%s\" DROP NOT NULL",
				escIdent(table), escIdent(newName)))
		} else {
			_, err = db.Exec(fmt.Sprintf("ALTER TABLE \"%s\" ALTER COLUMN \"%s\" SET NOT NULL",
				escIdent(table), escIdent(newName)))
		}
		if err != nil {
			return fmt.Errorf("postgres: alter nullable failed: %w", err)
		}
	}
	if col.Default != nil {
		if *col.Default == "" {
			_, err = db.Exec(fmt.Sprintf("ALTER TABLE \"%s\" ALTER COLUMN \"%s\" DROP DEFAULT",
				escIdent(table), escIdent(newName)))
		} else {
			_, err = db.Exec(fmt.Sprintf("ALTER TABLE \"%s\" ALTER COLUMN \"%s\" SET DEFAULT '%s'",
				escIdent(table), escIdent(newName), esc(*col.Default)))
		}
		if err != nil {
			return fmt.Errorf("postgres: alter default failed: %w", err)
		}
	}
	if col.Comment != nil {
		commentVal := "NULL"
		if *col.Comment != "" {
			commentVal = fmt.Sprintf("'%s'", esc(*col.Comment))
		}
		_, err = db.Exec(fmt.Sprintf("COMMENT ON COLUMN \"%s\".\"%s\" IS %s",
			escIdent(table), escIdent(newName), commentVal))
		if err != nil {
			return fmt.Errorf("postgres: alter comment failed: %w", err)
		}
	}
	return nil
}

func (p *Plugin) AddColumn(db *sql.DB, database, table string, col connector.ColumnDef) error {
	sqlStr := fmt.Sprintf("ALTER TABLE \"%s\" ADD COLUMN \"%s\" %s",
		escIdent(table), escIdent(col.Name), col.Type)
	if !col.Nullable {
		sqlStr += " NOT NULL"
	}
	if col.Default != "" {
		sqlStr += fmt.Sprintf(" DEFAULT '%s'", esc(col.Default))
	}
	_, err := db.Exec(sqlStr)
	if err != nil {
		return fmt.Errorf("postgres: add column failed: %w", err)
	}
	if col.Comment != "" {
		db.Exec(fmt.Sprintf("COMMENT ON COLUMN \"%s\".\"%s\" IS '%s'",
			escIdent(table), escIdent(col.Name), esc(col.Comment)))
	}
	return nil
}

func (p *Plugin) DropColumn(db *sql.DB, database, table, column string) error {
	_, err := db.Exec(fmt.Sprintf("ALTER TABLE \"%s\" DROP COLUMN \"%s\"",
		escIdent(table), escIdent(column)))
	if err != nil {
		return fmt.Errorf("postgres: drop column failed: %w", err)
	}
	return nil
}

func (p *Plugin) ListUsers(db *sql.DB) ([]connector.DatabaseUser, error) {
	rows, err := db.Query("SELECT rolname FROM pg_roles WHERE rolcanlogin=true ORDER BY rolname")
	if err != nil {
		return nil, fmt.Errorf("postgres: list users failed: %w", err)
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
	_, err := db.Exec(fmt.Sprintf("CREATE ROLE \"%s\" WITH LOGIN PASSWORD '%s'",
		escIdent(user), esc(password)))
	if err != nil {
		return fmt.Errorf("postgres: create user failed: %w", err)
	}
	return nil
}

func (p *Plugin) DropUser(db *sql.DB, user string) error {
	_, err := db.Exec(fmt.Sprintf("DROP ROLE \"%s\"", escIdent(user)))
	if err != nil {
		return fmt.Errorf("postgres: drop user failed: %w", err)
	}
	return nil
}

func (p *Plugin) GetPrivileges(db *sql.DB, user string) ([]connector.UserPrivilege, error) {
	rows, err := db.Query(fmt.Sprintf(`SELECT table_schema, table_name, privilege_type
		FROM information_schema.table_privileges WHERE grantee='%s'
		ORDER BY table_schema, table_name`, esc(user)))
	if err != nil {
		return nil, fmt.Errorf("postgres: get privileges failed: %w", err)
	}
	defer rows.Close()
	privMap := make(map[string]*connector.UserPrivilege)
	for rows.Next() {
		var schema, table, priv string
		if err := rows.Scan(&schema, &table, &priv); err != nil {
			return nil, err
		}
		key := schema + "." + table
		if up, ok := privMap[key]; ok {
			up.Privileges = append(up.Privileges, priv)
		} else {
			privMap[key] = &connector.UserPrivilege{
				User: user, Database: schema, Table: table,
				Privileges: []string{priv},
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
		_, err := db.Exec(fmt.Sprintf("GRANT %s ON \"%s\".\"%s\" TO \"%s\"",
			priv, escIdent(database), escIdent(table), escIdent(user)))
		if err != nil {
			return fmt.Errorf("postgres: grant privilege failed: %w", err)
		}
	}
	return nil
}

func (p *Plugin) RevokePrivilege(db *sql.DB, user, database, table string, privs []string) error {
	for _, priv := range privs {
		_, err := db.Exec(fmt.Sprintf("REVOKE %s ON \"%s\".\"%s\" FROM \"%s\"",
			priv, escIdent(database), escIdent(table), escIdent(user)))
		if err != nil {
			return fmt.Errorf("postgres: revoke privilege failed: %w", err)
		}
	}
	return nil
}

func (p *Plugin) GetAutoCompleteData(db *sql.DB, database string) (*connector.AutoCompleteData, error) {
	data := &connector.AutoCompleteData{Keywords: pgKeywords, Functions: pgFunctions}
	tables, err := p.ListTables(db, database)
	if err == nil {
		for _, t := range tables {
			data.Tables = append(data.Tables, connector.TableRef{Name: t.Name, Schema: t.Schema})
			cols, _ := p.GetColumns(db, database, t.Name)
			for _, c := range cols {
				data.Columns = append(data.Columns, connector.ColumnRef{Table: t.Name, Column: c.Name, Type: c.DataType})
			}
		}
	}
	return data, nil
}

func (p *Plugin) GetVersion(db *sql.DB) (string, error) {
	var version string
	err := db.QueryRow("SELECT version()").Scan(&version)
	if err != nil {
		return "", fmt.Errorf("postgres: get version failed: %w", err)
	}
	return "PostgreSQL " + version, nil
}

func esc(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

func escIdent(s string) string {
	return strings.ReplaceAll(s, "\"", "\"\"")
}

var pgKeywords = []string{
	"SELECT", "FROM", "WHERE", "INSERT", "UPDATE", "DELETE", "CREATE", "ALTER", "DROP",
	"TABLE", "INDEX", "VIEW", "SCHEMA", "INTO", "VALUES", "SET", "JOIN", "LEFT", "RIGHT",
	"INNER", "OUTER", "CROSS", "FULL", "ON", "AND", "OR", "NOT", "IN", "EXISTS", "BETWEEN", "LIKE", "ILIKE",
	"IS", "NULL", "AS", "ORDER", "BY", "ASC", "DESC", "GROUP", "HAVING", "LIMIT", "OFFSET",
	"UNION", "ALL", "DISTINCT", "CASE", "WHEN", "THEN", "ELSE", "END",
	"PRIMARY", "KEY", "FOREIGN", "REFERENCES", "CONSTRAINT", "UNIQUE", "CHECK", "DEFAULT",
	"TRUNCATE", "RENAME", "EXPLAIN", "ANALYZE", "VACUUM",
	"GRANT", "REVOKE", "BEGIN", "COMMIT", "ROLLBACK",
	"SERIAL", "BIGSERIAL", "BOOLEAN", "INTEGER", "BIGINT", "TEXT", "VARCHAR", "TIMESTAMP",
	"RETURNING", "CASCADE", "IF", "NOT", "EXISTS",
	"WHERE", "HAVING", "UNION",
}

var pgFunctions = []string{
	"COUNT", "SUM", "AVG", "MIN", "MAX", "STRING_AGG", "ARRAY_AGG",
	"NOW", "CURRENT_DATE", "CURRENT_TIME", "CURRENT_TIMESTAMP",
	"DATE_TRUNC", "DATE_PART", "AGE", "EXTRACT",
	"CONCAT", "SUBSTRING", "LEFT", "RIGHT", "LENGTH", "CHAR_LENGTH",
	"UPPER", "LOWER", "TRIM", "LTRIM", "RTRIM", "REPLACE", "SPLIT_PART",
	"ABS", "CEIL", "FLOOR", "ROUND", "MOD", "RANDOM",
	"COALESCE", "NULLIF", "GREATEST", "LEAST",
	"CAST", "TO_CHAR", "TO_DATE", "TO_TIMESTAMP", "TO_NUMBER",
	"MD5", "GEN_RANDOM_UUID",
	"ROW_NUMBER", "RANK", "DENSE_RANK", "LAG", "LEAD", "NTILE",
	"JSON_BUILD_OBJECT", "JSON_AGG", "JSONB_EXTRACT_PATH",
	"ARRAY", "UNNEST",
}
