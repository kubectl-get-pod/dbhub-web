package handler

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"dbhub-web/connector"
)

func ExportCSV(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ConnID   string `json:"connId"`
		SQL      string `json:"sql"`
		Filename string `json:"filename"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "请求格式错误", err, http.StatusBadRequest)
		return
	}
	db, _, plugin, err := getDBAndPlugin(req.ConnID)
	if err != nil {
		writeError(w, "连接未打开", err, http.StatusBadRequest)
		return
	}
	result, err := plugin.Query(db, req.SQL, 0, 0)
	if err != nil {
		writeError(w, "查询失败", err, http.StatusInternalServerError)
		return
	}

	filename := req.Filename
	if filename == "" {
		filename = "export_" + time.Now().Format("20060102_150405") + ".csv"
	}

	// UTF-8 BOM for Excel compatibility
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Write([]byte{0xEF, 0xBB, 0xBF}) // BOM

	writer := csv.NewWriter(w)
	writer.Write(result.Columns)
	for _, row := range result.Rows {
		strRow := make([]string, len(row))
		for i, val := range row {
			strRow[i] = fmt.Sprintf("%v", val)
		}
		writer.Write(strRow)
	}
	writer.Flush()
}

func ExportExcel(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ConnID   string `json:"connId"`
		SQL      string `json:"sql"`
		Filename string `json:"filename"`
		Database string `json:"database,omitempty"`
		Table    string `json:"table,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "请求格式错误", err, http.StatusBadRequest)
		return
	}
	db, _, plugin, err := getDBAndPlugin(req.ConnID)
	if err != nil {
		writeError(w, "连接未打开", err, http.StatusBadRequest)
		return
	}

	// 设置数据库上下文
	if req.Database != "" {
		plugin.SetDatabase(db, req.Database)
	}

	result, err := plugin.Query(db, req.SQL, 0, 0)
	if err != nil {
		writeError(w, "查询失败", err, http.StatusInternalServerError)
		return
	}

	// 如果 table 为空，尝试从 SQL 中提取表名
	table := req.Table
	if table == "" && req.Database != "" {
		table = extractTableFromSQL(req.SQL)
	}

	// 获取列注释
	commentMap := map[string]string{}
	if req.Database != "" && table != "" {
		cols, err := plugin.GetColumns(db, req.Database, table)
		if err == nil {
			for _, c := range cols {
				if c.Comment != "" {
					commentMap[c.Name] = c.Comment
				}
			}
		}
	}

	f := excelize.NewFile()
	sheet := "Sheet1"

	// Header style
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 11, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"4472C4"}, Pattern: 1},
	})

	// Comment row style
	hasComments := false
	for _, col := range result.Columns {
		if commentMap[col] != "" {
			hasComments = true
			break
		}
	}
	var commentStyle int
	dataStartRow := 2
	if hasComments {
		commentStyle, _ = f.NewStyle(&excelize.Style{
			Font: &excelize.Font{Italic: true, Size: 10, Color: "808080"},
		})
		dataStartRow = 3
	}

	// Write header row
	for i, col := range result.Columns {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, col)
	}
	f.SetRowStyle(sheet, 1, 1, headerStyle)

	// Write comment row (if available)
	if hasComments {
		for i, col := range result.Columns {
			cell, _ := excelize.CoordinatesToCellName(i+1, 2)
			f.SetCellValue(sheet, cell, commentMap[col])
		}
		f.SetRowStyle(sheet, 2, 2, commentStyle)
	}

	// Data rows
	for rIdx, row := range result.Rows {
		for cIdx, val := range row {
			cell, _ := excelize.CoordinatesToCellName(cIdx+1, rIdx+dataStartRow)
			f.SetCellValue(sheet, cell, val)
		}
	}

	filename := req.Filename
	if filename == "" {
		filename = "export_" + time.Now().Format("20060102_150405") + ".xlsx"
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	f.Write(w)
}

func ImportCSV(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, "文件上传失败", err, http.StatusBadRequest)
		return
	}
	connID := r.FormValue("connId")
	database := r.FormValue("database")
	tableName := r.FormValue("table")
	skipHeader := r.FormValue("skipHeader") == "true"

	delimiter := r.FormValue("delimiter")
	if delimiter == "" {
		delimiter = ","
	}
	runes := []rune(delimiter)
	if len(runes) != 1 {
		writeError(w, "分隔符必须为单个字符", nil, http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, "读取文件失败", err, http.StatusBadRequest)
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = runes[0]
	reader.LazyQuotes = true

	var headers []string
	if !skipHeader {
		headers, err = reader.Read()
		if err != nil {
			writeError(w, "CSV格式错误", err, http.StatusBadRequest)
			return
		}
	}

	db, _, _, err := getDBAndPlugin(connID)
	if err != nil {
		writeError(w, "连接未打开", err, http.StatusBadRequest)
		return
	}

	columnsStr := strings.TrimSpace(r.FormValue("columns"))
	var columnMap []int
	if columnsStr != "" {
		colNames := strings.Split(columnsStr, ",")
		columnMap = make([]int, len(colNames))
		for i, cn := range colNames {
			found := -1
			for j, h := range headers {
				if strings.TrimSpace(cn) == strings.TrimSpace(h) {
					found = j
					break
				}
			}
			if found < 0 {
				found = i // fallback: use position
			}
			columnMap[i] = found
		}
	} else {
		columnMap = make([]int, len(headers))
		for i := range columnMap {
			columnMap[i] = i
		}
	}

	inserted := 0
	failed := 0
	placeholders := "(" + strings.Repeat("?,", len(columnMap)-1) + "?)"
	for {
		record, err := reader.Read()
		if err != nil {
			break
		}
		values := make([]interface{}, len(columnMap))
		for i, idx := range columnMap {
			if idx < len(record) {
				values[i] = record[idx]
			}
		}
		_, err = db.Exec("INSERT INTO "+safeIdent(tableName)+" VALUES "+placeholders, values...)
		if err != nil {
			failed++
		} else {
			inserted++
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"inserted": inserted,
		"failed":   failed,
		"table":    tableName,
		"database": database,
	})
}

func GetAutoComplete(w http.ResponseWriter, r *http.Request) {
	connID := r.PathValue("connId")
	database := r.PathValue("database")
	db, _, plugin, err := getDBAndPlugin(connID)
	if err != nil {
		writeError(w, "连接未打开", err, http.StatusBadRequest)
		return
	}
	data, err := plugin.GetAutoCompleteData(db, database)
	if err != nil {
		writeError(w, "获取补全数据失败", err, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, data)
}

func GetAutoCompleteEmpty(w http.ResponseWriter, r *http.Request) {
	connID := r.PathValue("connId")
	db, _, plugin, err := getDBAndPlugin(connID)
	if err != nil {
		writeError(w, "连接未打开", err, http.StatusBadRequest)
		return
	}
	// 无数据库时返回仅关键字和函数的补全
	data := &connector.AutoCompleteData{}
	if d, e := plugin.GetAutoCompleteData(db, ""); e == nil {
		data = d
	}
	writeJSON(w, http.StatusOK, data)
}

func ListUsers(w http.ResponseWriter, r *http.Request) {
	connID := r.PathValue("connId")
	db, _, plugin, err := getDBAndPlugin(connID)
	if err != nil {
		writeError(w, "连接未打开", err, http.StatusBadRequest)
		return
	}
	users, err := plugin.ListUsers(db)
	if err != nil {
		writeError(w, "列出用户失败", err, http.StatusInternalServerError)
		return
	}
	if users == nil {
		users = []connector.DatabaseUser{}
	}
	writeJSON(w, http.StatusOK, users)
}

func CreateUser(w http.ResponseWriter, r *http.Request) {
	connID := r.PathValue("connId")
	db, _, plugin, err := getDBAndPlugin(connID)
	if err != nil {
		writeError(w, "连接未打开", err, http.StatusBadRequest)
		return
	}
	var req struct {
		User     string `json:"user"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "请求格式错误", err, http.StatusBadRequest)
		return
	}
	if err := plugin.CreateUser(db, req.User, req.Password); err != nil {
		writeError(w, "创建用户失败", err, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func DeleteUser(w http.ResponseWriter, r *http.Request) {
	connID := r.PathValue("connId")
	user := r.PathValue("user")
	db, _, plugin, err := getDBAndPlugin(connID)
	if err != nil {
		writeError(w, "连接未打开", err, http.StatusBadRequest)
		return
	}
	if err := plugin.DropUser(db, user); err != nil {
		writeError(w, "删除用户失败", err, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func GetPrivileges(w http.ResponseWriter, r *http.Request) {
	connID := r.PathValue("connId")
	user := r.PathValue("user")
	db, _, plugin, err := getDBAndPlugin(connID)
	if err != nil {
		writeError(w, "连接未打开", err, http.StatusBadRequest)
		return
	}
	privs, err := plugin.GetPrivileges(db, user)
	if err != nil {
		writeError(w, "获取权限失败", err, http.StatusInternalServerError)
		return
	}
	if privs == nil {
		privs = []connector.UserPrivilege{}
	}
	writeJSON(w, http.StatusOK, privs)
}

func GrantPrivilege(w http.ResponseWriter, r *http.Request) {
	connID := r.PathValue("connId")
	user := r.PathValue("user")
	db, _, plugin, err := getDBAndPlugin(connID)
	if err != nil {
		writeError(w, "连接未打开", err, http.StatusBadRequest)
		return
	}
	var req struct {
		Database   string   `json:"database"`
		Table      string   `json:"table"`
		Privileges []string `json:"privileges"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "请求格式错误", err, http.StatusBadRequest)
		return
	}
	if err := plugin.GrantPrivilege(db, user, req.Database, req.Table, req.Privileges); err != nil {
		writeError(w, "授予权限失败", err, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func RevokePrivilege(w http.ResponseWriter, r *http.Request) {
	connID := r.PathValue("connId")
	user := r.PathValue("user")
	db, _, plugin, err := getDBAndPlugin(connID)
	if err != nil {
		writeError(w, "连接未打开", err, http.StatusBadRequest)
		return
	}
	var req struct {
		Database   string   `json:"database"`
		Table      string   `json:"table"`
		Privileges []string `json:"privileges"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "请求格式错误", err, http.StatusBadRequest)
		return
	}
	if err := plugin.RevokePrivilege(db, user, req.Database, req.Table, req.Privileges); err != nil {
		writeError(w, "撤销权限失败", err, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func GetVersion(w http.ResponseWriter, r *http.Request) {
	connID := r.PathValue("connId")
	db, _, plugin, err := getDBAndPlugin(connID)
	if err != nil {
		writeError(w, "连接未打开", err, http.StatusBadRequest)
		return
	}
	version, err := plugin.GetVersion(db)
	if err != nil {
		writeError(w, "获取版本失败", err, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"version": version})
}

// extractTableFromSQL 从 SQL 中提取 FROM 后的第一个表名
func extractTableFromSQL(sql string) string {
	s := toUpper(strings.TrimSpace(sql))
	idx := strings.Index(s, "FROM ")
	if idx < 0 {
		return ""
	}
	rest := strings.TrimSpace(sql[idx+5:])
	// 去掉反引号、方括号、双引号
	rest = strings.TrimLeft(rest, "`\"[")
	end := 0
	for end < len(rest) {
		c := rest[end]
		if c == ' ' || c == '\t' || c == '\n' || c == ',' || c == ';' || c == ')' ||
			c == '`' || c == '"' || c == ']' {
			break
		}
		// 检测 JOIN/WHERE 等关键字（转为大写比较）
		remaining := strings.ToUpper(rest[end:])
		if strings.HasPrefix(remaining, "JOIN") || strings.HasPrefix(remaining, "WHERE") ||
			strings.HasPrefix(remaining, "ORDER") || strings.HasPrefix(remaining, "GROUP") ||
			strings.HasPrefix(remaining, "LIMIT") || strings.HasPrefix(remaining, "HAVING") ||
			strings.HasPrefix(remaining, "UNION") || strings.HasPrefix(remaining, "INNER") ||
			strings.HasPrefix(remaining, "LEFT") || strings.HasPrefix(remaining, "RIGHT") ||
			strings.HasPrefix(remaining, "CROSS") || strings.HasPrefix(remaining, "ON ") {
			break
		}
		end++
	}
	table := rest[:end]
	// 处理 schema.table 格式（取 table 部分）
	if dot := strings.Index(table, "."); dot >= 0 {
		table = table[dot+1:]
	}
	return strings.Trim(table, "`\"[]")
}
