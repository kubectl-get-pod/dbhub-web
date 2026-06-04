# dbhub-web

零依赖跨平台 Web 数据库管理工具。Go 后端 + React 前端，编译为 **单一可执行文件**。

## 支持的数据库

| 数据库 | 驱动 |
|--------|------|
| MySQL | go-sql-driver/mysql |
| PostgreSQL | jackc/pgx/v5 |
| Oracle | sijms/go-ora/v2 |
| SQL Server | microsoft/go-mssqldb |

## 下载

| 平台 | 下载 |
|------|------|
| Windows x64 | [dbhub-web.exe](https://github.com/kubectl-get-pod/dbhub-web/blob/releases/dbhub-web.exe?raw=true) |
| Linux x64 | [dbhub-web](https://github.com/kubectl-get-pod/dbhub-web/blob/releases/dbhub-web?raw=true) |
| macOS Intel | [dbhub-web-darwin-amd64](https://github.com/kubectl-get-pod/dbhub-web/blob/releases/dbhub-web-darwin-amd64?raw=true) |
| macOS Apple Silicon | [dbhub-web-darwin-arm64](https://github.com/kubectl-get-pod/dbhub-web/blob/releases/dbhub-web-darwin-arm64?raw=true) |

## 快速开始

```bash
# Windows
dbhub-web.exe

# Linux
./dbhub-web
```

浏览器打开 http://localhost:8080

## 功能

- 连接管理（分组/折叠/重命名）
- 数据库/表树形浏览 + 搜索
- 表结构查看（列/索引/外键/DDL）+ 列编辑
- 数据浏览分页 + 行内增删改
- CodeMirror 6 SQL 编辑器（语法高亮/自动补全/Tab 选中/格式化）
- CSV / Excel 导出，CSV 导入
- 查询历史 + 收藏查询
- 用户权限管理
- 暗色/亮色主题

## 构建

```bash
make build-win    # Windows
make build-linux  # Linux
make build        # 当前平台
```

CGO_ENABLED=0，纯 Go 静态编译，无需任何运行时依赖。

## 技术栈

| 层 | 技术 |
|----|------|
| 后端 | Go 1.26 (net/http) |
| 前端 | React 18 + TypeScript + Vite |
| 状态管理 | Zustand |
| 样式 | Tailwind CSS |
| SQL 编辑器 | CodeMirror 6 |
| 二进制 | go:embed 嵌入前端 → 单文件 |

## 许可

MIT
