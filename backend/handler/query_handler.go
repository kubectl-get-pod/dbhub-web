package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"dbhub-web/store"
)

func ExecuteQuery(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ConnID   string `json:"connId"`
		SQL      string `json:"sql"`
		Limit    int    `json:"limit"`
		Offset   int    `json:"offset"`
		Database string `json:"database,omitempty"` // 可选：设置当前数据库上下文
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "请求格式错误", err, http.StatusBadRequest)
		return
	}
	if req.SQL == "" {
		writeError(w, "SQL不能为空", nil, http.StatusBadRequest)
		return
	}
	db, cfg, plugin, err := getDBAndPlugin(req.ConnID)
	if err != nil {
		writeError(w, "连接未打开", err, http.StatusBadRequest)
		return
	}

	// 设置数据库上下文（如果提供）
	if req.Database != "" {
		if err := plugin.SetDatabase(db, req.Database); err != nil {
			writeError(w, "设置数据库失败", err, http.StatusBadRequest)
			return
		}
	}

	sqlUpper := toUpper(req.SQL)
	isSelect := startsWith(sqlUpper, "SELECT") || startsWith(sqlUpper, "SHOW") || startsWith(sqlUpper, "DESCRIBE") || startsWith(sqlUpper, "EXPLAIN")

	start := time.Now()
	if isSelect {
		result, qErr := plugin.Query(db, req.SQL, req.Limit, req.Offset)
		if qErr != nil {
			writeError(w, "查询失败", qErr, http.StatusBadRequest)
			return
		}
		result.Duration = time.Since(start).String()
		// 保存历史
		saveHistory(req.ConnID, cfg.Name, req.SQL, result.Duration)
		writeJSON(w, http.StatusOK, result)
	} else {
		affected, qErr := plugin.Execute(db, req.SQL)
		if qErr != nil {
			writeError(w, "执行失败", qErr, http.StatusBadRequest)
			return
		}
		saveHistory(req.ConnID, cfg.Name, req.SQL, time.Since(start).String())
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"affected": affected,
			"duration": time.Since(start).String(),
		})
	}
}

func ListHistory(w http.ResponseWriter, r *http.Request) {
	s := GetStore()
	items, err := s.LoadHistory()
	if err != nil {
		writeJSON(w, http.StatusOK, []store.QueryHistoryItem{})
		return
	}
	if items == nil {
		items = []store.QueryHistoryItem{}
	}
	writeJSON(w, http.StatusOK, items)
}

func DeleteHistory(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	s := GetStore()
	items, err := s.LoadHistory()
	if err != nil {
		writeError(w, "加载历史失败", err, http.StatusInternalServerError)
		return
	}
	found := false
	for i, item := range items {
		if item.ID == id {
			items = append(items[:i], items[i+1:]...)
			found = true
			break
		}
	}
	if !found {
		writeError(w, "历史记录不存在", nil, http.StatusNotFound)
		return
	}
	if err := s.SaveHistory(items); err != nil {
		writeError(w, "保存历史失败", err, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func ListFavorites(w http.ResponseWriter, r *http.Request) {
	s := GetStore()
	items, err := s.LoadFavorites()
	if err != nil {
		writeJSON(w, http.StatusOK, []store.FavoriteItem{})
		return
	}
	if items == nil {
		items = []store.FavoriteItem{}
	}
	writeJSON(w, http.StatusOK, items)
}

func CreateFavorite(w http.ResponseWriter, r *http.Request) {
	var item store.FavoriteItem
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		writeError(w, "请求格式错误", err, http.StatusBadRequest)
		return
	}
	item.ID = generateID()
	item.CreatedAt = time.Now().Format("2006-01-02 15:04:05")
	s := GetStore()
	items, err := s.LoadFavorites()
	if err != nil {
		writeError(w, "加载收藏失败", err, http.StatusInternalServerError)
		return
	}
	items = append(items, item)
	if err := s.SaveFavorites(items); err != nil {
		writeError(w, "保存收藏失败", err, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func UpdateFavorite(w http.ResponseWriter, r *http.Request) { writeJSON(w, http.StatusOK, map[string]string{"status": "ok"}) }
func DeleteFavorite(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	s := GetStore()
	items, err := s.LoadFavorites()
	if err != nil {
		writeError(w, "加载收藏失败", err, http.StatusInternalServerError)
		return
	}
	for i, item := range items {
		if item.ID == id {
			items = append(items[:i], items[i+1:]...)
			break
		}
	}
	if err := s.SaveFavorites(items); err != nil {
		writeError(w, "保存收藏失败", err, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func saveHistory(connID, connName, sql, duration string) {
	s := GetStore()
	items, _ := s.LoadHistory()
	items = append(items, store.QueryHistoryItem{
		ID:        generateID(),
		SQL:       sql,
		ConnName:  connName,
		CreatedAt: time.Now().Format("2006-01-02 15:04:05"),
		Duration:  duration,
	})
	s.SaveHistory(items)
}

func toUpper(s string) string {
	r := []byte(s)
	for i, c := range r {
		if c >= 'a' && c <= 'z' {
			r[i] = c - 32
		}
	}
	return string(r)
}

func startsWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
