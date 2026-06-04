package handler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"dbhub-web/connector"
)

// ========== 连接 CRUD ==========

func ListConnections(w http.ResponseWriter, r *http.Request) {
	store := GetStore()
	conns, err := store.LoadConnections()
	if err != nil {
		writeError(w, "加载连接列表失败", err, http.StatusInternalServerError)
		return
	}
	if conns == nil {
		conns = []connector.ConnectionConfig{}
	}
	writeJSON(w, http.StatusOK, conns)
}

func CreateConnection(w http.ResponseWriter, r *http.Request) {
	var cfg connector.ConnectionConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeError(w, "请求格式错误", err, http.StatusBadRequest)
		return
	}
	cfg.ID = generateID()
	store := GetStore()
	conns, err := store.LoadConnections()
	if err != nil {
		writeError(w, "加载连接失败", err, http.StatusInternalServerError)
		return
	}
	conns = append(conns, cfg)
	if err := store.SaveConnections(conns); err != nil {
		writeError(w, "保存连接失败", err, http.StatusInternalServerError)
		return
	}
	// 返回时不暴露密码
	cfg.Password = ""
	cfg.SSHPass = ""
	writeJSON(w, http.StatusCreated, cfg)
}

func UpdateConnection(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var cfg connector.ConnectionConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeError(w, "请求格式错误", err, http.StatusBadRequest)
		return
	}
	cfg.ID = id
	store := GetStore()
	conns, err := store.LoadConnections()
	if err != nil {
		writeError(w, "加载连接失败", err, http.StatusInternalServerError)
		return
	}
	found := false
	for i, c := range conns {
		if c.ID == id {
			// 保留原密码（如果新配置未提供）
			if cfg.Password == "" {
				cfg.Password = c.Password
			}
			if cfg.SSHPass == "" {
				cfg.SSHPass = c.SSHPass
			}
			conns[i] = cfg
			found = true
			break
		}
	}
	if !found {
		writeError(w, "连接不存在", nil, http.StatusNotFound)
		return
	}
	if err := store.SaveConnections(conns); err != nil {
		writeError(w, "保存连接失败", err, http.StatusInternalServerError)
		return
	}
	cfg.Password = ""
	cfg.SSHPass = ""
	writeJSON(w, http.StatusOK, cfg)
}

func DeleteConnection(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	store := GetStore()
	conns, err := store.LoadConnections()
	if err != nil {
		writeError(w, "加载连接失败", err, http.StatusInternalServerError)
		return
	}
	found := false
	for i, c := range conns {
		if c.ID == id {
			conns = append(conns[:i], conns[i+1:]...)
			found = true
			break
		}
	}
	if !found {
		writeError(w, "连接不存在", nil, http.StatusNotFound)
		return
	}
	// 也关闭打开的连接
	connector.GlobalPool.Close(id)
	if err := store.SaveConnections(conns); err != nil {
		writeError(w, "保存连接失败", err, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func TestConnection(w http.ResponseWriter, r *http.Request) {
	var cfg connector.ConnectionConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		writeError(w, "请求格式错误", err, http.StatusBadRequest)
		return
	}
	plugin, err := connector.GlobalRegistry.Get(cfg.Type)
	if err != nil {
		writeError(w, "数据库类型不支持", err, http.StatusBadRequest)
		return
	}
	db, err := plugin.Open(&cfg)
	if err != nil {
		writeError(w, "连接失败", err, http.StatusBadRequest)
		return
	}
	defer db.Close()
	if err := plugin.Ping(db); err != nil {
		writeError(w, "连接测试失败", err, http.StatusBadRequest)
		return
	}
	version, _ := plugin.GetVersion(db)
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"version": version,
	})
}

// ========== 连接打开/关闭 ==========

func ConnectDB(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	store := GetStore()
	conns, err := store.LoadConnections()
	if err != nil {
		writeError(w, "加载连接失败", err, http.StatusInternalServerError)
		return
	}
	var cfg *connector.ConnectionConfig
	for _, c := range conns {
		if c.ID == id {
			cfg = &c
			break
		}
	}
	if cfg == nil {
		writeError(w, "连接不存在", nil, http.StatusNotFound)
		return
	}
	plugin, err := connector.GlobalRegistry.Get(cfg.Type)
	if err != nil {
		writeError(w, "数据库类型不支持", err, http.StatusBadRequest)
		return
	}
	db, err := plugin.Open(cfg)
	if err != nil {
		writeError(w, "打开连接失败", err, http.StatusInternalServerError)
		return
	}
	if err := plugin.Ping(db); err != nil {
		db.Close()
		writeError(w, "连接失败", err, http.StatusBadRequest)
		return
	}
	connector.GlobalPool.Open(id, db, cfg)
	version, _ := plugin.GetVersion(db)
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "connected",
		"id":      id,
		"name":    cfg.Name,
		"type":    cfg.Type,
		"version": version,
	})
}

func DisconnectDB(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := connector.GlobalPool.Close(id); err != nil {
		writeError(w, "关闭连接失败", err, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "disconnected"})
}

// ========== 工具函数 ==========

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, msg string, err error, status int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	resp := map[string]string{"error": msg}
	if err != nil {
		resp["detail"] = err.Error()
	}
	json.NewEncoder(w).Encode(resp)
}

func generateID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b) + time.Now().Format("150405")
}

func HealthCheck(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
		"name":   "dbhub-web",
	})
}
