package handler

import (
	"net/http"
)

// RegisterRoutes 注册所有 API 路由
func RegisterRoutes(mux *http.ServeMux, dataDir string) {
	// 包装所有 handler：安全头 → CORS → 日志 → panic 恢复
	api := func(h http.HandlerFunc) http.HandlerFunc {
		return SecurityHeaders(CorsMiddleware(LoggerMiddleware(RecoveryMiddleware(h))))
	}

	// 初始化数据存储（后续 handler 使用）
	initStore(dataDir)

	// 连接管理
	mux.HandleFunc("GET /api/connections", api(ListConnections))
	mux.HandleFunc("POST /api/connections", api(CreateConnection))
	mux.HandleFunc("PUT /api/connections/{id}", api(UpdateConnection))
	mux.HandleFunc("DELETE /api/connections/{id}", api(DeleteConnection))
	mux.HandleFunc("POST /api/connections/test", api(TestConnection))

	// 连接打开/关闭
	mux.HandleFunc("POST /api/connect/{id}", api(ConnectDB))
	mux.HandleFunc("POST /api/disconnect/{id}", api(DisconnectDB))

	// 数据库/表浏览
	mux.HandleFunc("GET /api/databases/{connId}", api(ListDatabases))
	mux.HandleFunc("GET /api/tables/{connId}/{database}", api(ListTables))

	// 表结构
	mux.HandleFunc("GET /api/schema/{connId}/{database}/{table}", api(GetSchema))
	mux.HandleFunc("GET /api/ddl/{connId}/{database}/{table}", api(GetDDL))

	// 表结构修改
	mux.HandleFunc("PUT /api/schema/{connId}/{database}/{table}/column", api(AlterColumn))
	mux.HandleFunc("POST /api/schema/{connId}/{database}/{table}/column", api(AddColumn))
	mux.HandleFunc("DELETE /api/schema/{connId}/{database}/{table}/column", api(DropColumn))

	// 数据
	mux.HandleFunc("GET /api/data/{connId}/{database}/{table}", api(ListData))
	mux.HandleFunc("GET /api/rowcount/{connId}/{database}/{table}", api(GetRowCount))
	mux.HandleFunc("POST /api/data/{connId}/{database}/{table}", api(InsertRow))
	mux.HandleFunc("PUT /api/data/{connId}/{database}/{table}", api(UpdateRow))
	mux.HandleFunc("DELETE /api/data/{connId}/{database}/{table}", api(DeleteRow))

	// SQL 查询
	mux.HandleFunc("POST /api/query", api(ExecuteQuery))

	// 查询历史
	mux.HandleFunc("GET /api/history", api(ListHistory))
	mux.HandleFunc("DELETE /api/history/{id}", api(DeleteHistory))

	// 收藏查询
	mux.HandleFunc("GET /api/favorites", api(ListFavorites))
	mux.HandleFunc("POST /api/favorites", api(CreateFavorite))
	mux.HandleFunc("PUT /api/favorites/{id}", api(UpdateFavorite))
	mux.HandleFunc("DELETE /api/favorites/{id}", api(DeleteFavorite))

	// 导出
	mux.HandleFunc("POST /api/export/csv", api(ExportCSV))
	mux.HandleFunc("POST /api/export/excel", api(ExportExcel))

	// 导入
	mux.HandleFunc("POST /api/import/csv", api(ImportCSV))

	// 自动补全
	mux.HandleFunc("GET /api/autocomplete/{connId}/{database}", api(GetAutoComplete))
	mux.HandleFunc("GET /api/autocomplete/{connId}", api(GetAutoCompleteEmpty))

	// 用户管理
	mux.HandleFunc("GET /api/users/{connId}", api(ListUsers))
	mux.HandleFunc("POST /api/users/{connId}", api(CreateUser))
	mux.HandleFunc("DELETE /api/users/{connId}/{user}", api(DeleteUser))
	mux.HandleFunc("GET /api/privileges/{connId}/{user}", api(GetPrivileges))
	mux.HandleFunc("POST /api/privileges/{connId}/{user}/grant", api(GrantPrivilege))
	mux.HandleFunc("POST /api/privileges/{connId}/{user}/revoke", api(RevokePrivilege))

	// 系统
	mux.HandleFunc("GET /api/version/{connId}", api(GetVersion))
	mux.HandleFunc("GET /api/health", api(HealthCheck))
}
