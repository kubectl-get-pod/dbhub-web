package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"dbhub-web/connector"
)

func ListDatabases(w http.ResponseWriter, r *http.Request) {
	connID := r.PathValue("connId")
	db, err := connector.GlobalPool.Get(connID)
	if err != nil {
		writeError(w, "connection not open", err, http.StatusBadRequest)
		return
	}
	cfg, err := connector.GlobalPool.GetConfig(connID)
	if err != nil {
		writeError(w, "get config failed", err, http.StatusInternalServerError)
		return
	}
	plugin, err := connector.GlobalRegistry.Get(cfg.Type)
	if err != nil {
		writeError(w, "plugin not found", err, http.StatusInternalServerError)
		return
	}
	databases, err := plugin.ListDatabases(db)
	if err != nil {
		writeError(w, "list databases failed", err, http.StatusInternalServerError)
		return
	}
	if databases == nil {
		databases = []string{}
	}
	writeJSON(w, http.StatusOK, databases)
}

func ListTables(w http.ResponseWriter, r *http.Request) {
	connID := r.PathValue("connId")
	database := r.PathValue("database")
	db, _, plugin, err := getDBAndPlugin(connID)
	if err != nil {
		writeError(w, err.Error(), nil, http.StatusBadRequest)
		return
	}
	tables, err := plugin.ListTables(db, database)
	if err != nil {
		writeError(w, "list tables failed", err, http.StatusInternalServerError)
		return
	}
	if tables == nil {
		tables = []connector.Table{}
	}
	writeJSON(w, http.StatusOK, tables)
}

func GetSchema(w http.ResponseWriter, r *http.Request) {
	connID := r.PathValue("connId")
	database := r.PathValue("database")
	table := r.PathValue("table")
	db, cfg, plugin, err := getDBAndPlugin(connID)
	if err != nil {
		writeError(w, err.Error(), nil, http.StatusBadRequest)
		return
	}
	columns, err := plugin.GetColumns(db, database, table)
	if err != nil {
		writeError(w, "get columns failed", err, http.StatusInternalServerError)
		return
	}
	indexes, err := plugin.GetIndexes(db, database, table)
	if err != nil {
		writeError(w, "get indexes failed", err, http.StatusInternalServerError)
		return
	}
	fks, err := plugin.GetForeignKeys(db, database, table)
	if err != nil {
		writeError(w, "get foreign keys failed", err, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"columns":     columns,
		"indexes":     indexes,
		"foreignKeys": fks,
		"connType":    cfg.Type,
	})
}

func GetDDL(w http.ResponseWriter, r *http.Request) {
	connID := r.PathValue("connId")
	database := r.PathValue("database")
	table := r.PathValue("table")
	db, _, plugin, err := getDBAndPlugin(connID)
	if err != nil {
		writeError(w, err.Error(), nil, http.StatusBadRequest)
		return
	}
	ddl, err := plugin.GetDDL(db, database, table)
	if err != nil {
		writeError(w, "get DDL failed", err, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"ddl": ddl})
}

func AlterColumn(w http.ResponseWriter, r *http.Request) {
	connID := r.PathValue("connId")
	database := r.PathValue("database")
	table := r.PathValue("table")
	db, _, plugin, err := getDBAndPlugin(connID)
	if err != nil {
		writeError(w, err.Error(), nil, http.StatusBadRequest)
		return
	}
	var change connector.ColumnChange
	if err := json.NewDecoder(r.Body).Decode(&change); err != nil {
		writeError(w, "invalid request body", err, http.StatusBadRequest)
		return
	}
	if err := plugin.AlterColumn(db, database, table, change); err != nil {
		writeError(w, "alter column failed", err, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func AddColumn(w http.ResponseWriter, r *http.Request) {
	connID := r.PathValue("connId")
	database := r.PathValue("database")
	table := r.PathValue("table")
	db, _, plugin, err := getDBAndPlugin(connID)
	if err != nil {
		writeError(w, err.Error(), nil, http.StatusBadRequest)
		return
	}
	var col connector.ColumnDef
	if err := json.NewDecoder(r.Body).Decode(&col); err != nil {
		writeError(w, "invalid request body", err, http.StatusBadRequest)
		return
	}
	if err := plugin.AddColumn(db, database, table, col); err != nil {
		writeError(w, "add column failed", err, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func DropColumn(w http.ResponseWriter, r *http.Request) {
	connID := r.PathValue("connId")
	database := r.PathValue("database")
	table := r.PathValue("table")
	db, _, plugin, err := getDBAndPlugin(connID)
	if err != nil {
		writeError(w, err.Error(), nil, http.StatusBadRequest)
		return
	}
	var req struct{ Column string `json:"column"` }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid request body", err, http.StatusBadRequest)
		return
	}
	if err := plugin.DropColumn(db, database, table, req.Column); err != nil {
		writeError(w, "drop column failed", err, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func ListData(w http.ResponseWriter, r *http.Request) {
	connID := r.PathValue("connId")
	database := r.PathValue("database")
	table := r.PathValue("table")
	db, _, plugin, err := getDBAndPlugin(connID)
	if err != nil {
		writeError(w, err.Error(), nil, http.StatusBadRequest)
		return
	}
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		fmt.Sscanf(v, "%d", &limit)
	}
	offset := 0
	if v := r.URL.Query().Get("offset"); v != "" {
		fmt.Sscanf(v, "%d", &offset)
	}
	sqlStr := fmt.Sprintf("SELECT * FROM %s", quoteIdent(database, table))
	result, err := plugin.Query(db, sqlStr, limit, offset)
	if err != nil {
		writeError(w, "query data failed", err, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func GetRowCount(w http.ResponseWriter, r *http.Request) {
	connID := r.PathValue("connId")
	database := r.PathValue("database")
	table := r.PathValue("table")
	db, _, plugin, err := getDBAndPlugin(connID)
	if err != nil {
		writeError(w, err.Error(), nil, http.StatusBadRequest)
		return
	}
	result, err := plugin.Query(db, fmt.Sprintf("SELECT COUNT(*) FROM %s", quoteIdent(database, table)), 0, 0)
	if err != nil {
		writeError(w, "count rows failed", err, http.StatusInternalServerError)
		return
	}
	count := int64(0)
	if len(result.Rows) > 0 && len(result.Rows[0]) > 0 {
		switch v := result.Rows[0][0].(type) {
		case float64:
			count = int64(v)
		case int64:
			count = v
		}
	}
	writeJSON(w, http.StatusOK, map[string]int64{"count": count})
}

func InsertRow(w http.ResponseWriter, r *http.Request) {
	connID := r.PathValue("connId")
	_ = r.PathValue("database")
	table := r.PathValue("table")
	db, _, _, err := getDBAndPlugin(connID)
	if err != nil {
		writeError(w, err.Error(), nil, http.StatusBadRequest)
		return
	}
	var row map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&row); err != nil {
		writeError(w, "invalid request body", err, http.StatusBadRequest)
		return
	}
	columns := make([]string, 0, len(row))
	placeholders := make([]string, 0, len(row))
	values := make([]interface{}, 0, len(row))
	for k, v := range row {
		columns = append(columns, k)
		placeholders = append(placeholders, "?")
		values = append(values, v)
	}
	sqlStr := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		safeIdent(table), strings.Join(columns, ", "), strings.Join(placeholders, ", "))
	result, err := db.Exec(sqlStr, values...)
	if err != nil {
		writeError(w, "insert row failed", err, http.StatusInternalServerError)
		return
	}
	affected, _ := result.RowsAffected()
	writeJSON(w, http.StatusOK, map[string]int64{"affected": affected})
}

func UpdateRow(w http.ResponseWriter, r *http.Request) {
	connID := r.PathValue("connId")
	_ = r.PathValue("database")
	table := r.PathValue("table")
	db, _, _, err := getDBAndPlugin(connID)
	if err != nil {
		writeError(w, err.Error(), nil, http.StatusBadRequest)
		return
	}
	var req struct {
		PK     map[string]interface{} `json:"pk"`
		Values map[string]interface{} `json:"values"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid request body", err, http.StatusBadRequest)
		return
	}
	setClauses := make([]string, 0, len(req.Values))
	values := make([]interface{}, 0)
	for k, v := range req.Values {
		setClauses = append(setClauses, k+" = ?")
		values = append(values, v)
	}
	whereClauses := make([]string, 0, len(req.PK))
	for k, v := range req.PK {
		whereClauses = append(whereClauses, k+" = ?")
		values = append(values, v)
	}
	sqlStr := fmt.Sprintf("UPDATE %s SET %s WHERE %s",
		safeIdent(table), strings.Join(setClauses, ", "), strings.Join(whereClauses, " AND "))
	result, err := db.Exec(sqlStr, values...)
	if err != nil {
		writeError(w, "update row failed", err, http.StatusInternalServerError)
		return
	}
	affected, _ := result.RowsAffected()
	writeJSON(w, http.StatusOK, map[string]int64{"affected": affected})
}

func DeleteRow(w http.ResponseWriter, r *http.Request) {
	connID := r.PathValue("connId")
	_ = r.PathValue("database")
	table := r.PathValue("table")
	db, _, _, err := getDBAndPlugin(connID)
	if err != nil {
		writeError(w, err.Error(), nil, http.StatusBadRequest)
		return
	}
	var pk map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&pk); err != nil {
		writeError(w, "invalid request body", err, http.StatusBadRequest)
		return
	}
	whereClauses := make([]string, 0, len(pk))
	values := make([]interface{}, 0)
	for k, v := range pk {
		whereClauses = append(whereClauses, k+" = ?")
		values = append(values, v)
	}
	sqlStr := fmt.Sprintf("DELETE FROM %s WHERE %s",
		safeIdent(table), strings.Join(whereClauses, " AND "))
	result, err := db.Exec(sqlStr, values...)
	if err != nil {
		writeError(w, "delete row failed", err, http.StatusInternalServerError)
		return
	}
	affected, _ := result.RowsAffected()
	writeJSON(w, http.StatusOK, map[string]int64{"affected": affected})
}

func getDBAndPlugin(connID string) (*sql.DB, *connector.ConnectionConfig, connector.DBPlugin, error) {
	db, err := connector.GlobalPool.Get(connID)
	if err != nil {
		return nil, nil, nil, err
	}
	cfg, err := connector.GlobalPool.GetConfig(connID)
	if err != nil {
		return nil, nil, nil, err
	}
	plugin, err := connector.GlobalRegistry.Get(cfg.Type)
	if err != nil {
		return nil, nil, nil, err
	}
	return db, cfg, plugin, nil
}

func quoteIdent(database, table string) string {
	if database != "" {
		return safeIdent(database) + "." + safeIdent(table)
	}
	return safeIdent(table)
}

func safeIdent(name string) string {
	for _, ch := range []string{"`", "\"", "[", "]", ";", "--"} {
		name = strings.ReplaceAll(name, ch, "")
	}
	return name
}
