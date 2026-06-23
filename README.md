<p align="center">
  <img src="https://img.shields.io/badge/Go-1.26-00ADD8?logo=go" alt="Go 1.26">
  <img src="https://img.shields.io/badge/React-18-61DAFB?logo=react" alt="React 18">
  <img src="https://img.shields.io/badge/license-MIT-green" alt="MIT">
  <img src="https://img.shields.io/badge/platform-Windows%20%7C%20Linux%20%7C%20macOS-blue" alt="Platform">
  <img src="https://img.shields.io/badge/release-v1.0.0-important" alt="v1.0.0">
</p>

<h1 align="center">dbhub-web</h1>
<p align="center">
  <b>一个文件，管好所有数据库</b><br>
  <sub>MySQL · PostgreSQL · Oracle · SQL Server — 下载即用，零依赖，零配置</sub>
</p>

---

## 为什么选 dbhub-web？

你只是想连上数据库跑条 SQL、看看表结构、改几行数据。你不想装 Java 运行时，不想配 ODBC 驱动，不想等半分钟启动。你只想要一个**双击就开、用完就关**的轻量工具。

**dbhub-web 就是答案。**

| 对比 | Navicat / DBeaver | HeidiSQL | dbhub-web |
|------|:--:|:--:|:--:|
| 需要安装运行时 | Java 17+ | 仅 Windows | **无** |
| 启动速度 | 慢 | 快 | **秒开** |
| 支持平台 | Win/Mac/Linux | 仅 Win | **Win/Linux/Mac** |
| 单一文件 | 否 | 否 | **是 (≈31MB)** |
| Oracle 支持 | 收费版 | 不支持 | **免费** |
| 完全离线 | 否 | 是 | **是** |
| 开源 | DBeaver 部分 | 是 | **MIT** |

---

## 快速开始

### 1. 下载

| 平台 | 链接 | 备注 |
|------|------|------|
| **Windows** | [dbhub-web.exe](https://github.com/kubectl-get-pod/dbhub-web/releases/download/v1.0.0/dbhub-web.exe) | x64, Windows 10+ |
| **Linux** | [dbhub-web](https://github.com/kubectl-get-pod/dbhub-web/releases/download/v1.0.0/dbhub-web) | x64, glibc |
| **macOS Intel** | [dbhub-web-darwin-amd64](https://github.com/kubectl-get-pod/dbhub-web/releases/download/v1.0.0/dbhub-web-darwin-amd64) | Intel 芯片 |
| **macOS M 系列** | [dbhub-web-darwin-arm64](https://github.com/kubectl-get-pod/dbhub-web/releases/download/v1.0.0/dbhub-web-darwin-arm64) | Apple Silicon |

> 所有版本均为**单个可执行文件**，无需安装任何依赖。完整 Release 见 [Releases 页面](https://github.com/kubectl-get-pod/dbhub-web/releases)。

### 2. 运行

```bash
# Windows — 双击即可
dbhub-web.exe

# Linux / macOS — 加执行权限后运行
chmod +x dbhub-web
./dbhub-web
```

### 3. 打开浏览器

访问 **http://localhost:8080**，开始管理你的数据库。

> 指定端口：`./dbhub-web.exe -port 3000`，也可通过 `PORT` 环境变量设置。

---

## 功能一览

### 连接管理
- 支持 MySQL、PostgreSQL、Oracle、SQL Server
- 连接分组、折叠、重命名
- AES-256-GCM 加密保存密码，本地存储不泄露

### 数据浏览
- 树形浏览数据库 / 表结构，支持搜索过滤
- 数据分页浏览，行内编辑、新增、删除，一键导出 Excel/CSV
- 列注释自动展示（表头下方灰字斜体），导出 Excel 第二行带注释
- SQL 删除操作前弹出确认框，防止误删

### SQL 编辑器
- CodeMirror 6 内核：语法高亮、智能补全、格式化
- 多标签页并行查询，支持 Ctrl+Enter 快捷执行
- 查询历史自动保存，收藏查询一键复用

### 表结构管理
- 查看列、索引、外键、建表 DDL
- 在线新增 / 编辑 / 删除列

### 导入导出
- CSV 导入（支持自定义分隔符、跳过表头）
- CSV / Excel 导出

### 其他
- 命令行参数 `-port 3000` 指定端口，优先级：命令行 > PORT 环境变量 > 默认 8080
- 用户权限查看与管理
- 亮色 / 暗色主题一键切换
- 纯 Go 编译，无 cgo 依赖，静态链接

---

## 技术架构

| 层 | 技术选型 | 说明 |
|----|----------|------|
| 后端 | Go 1.26 + net/http | 39 个 REST API，零框架 |
| 前端 | React 18 + TypeScript + Vite | 编译后由 go:embed 嵌入 |
| 状态管理 | Zustand | 轻量，无模板代码 |
| 样式 | Tailwind CSS | 实用优先，暗色模式内置 |
| 编辑器 | CodeMirror 6 | 专业 SQL 编辑体验 |
| 数据库驱动 | 全部纯 Go 实现 | 无 ODBC/JDBC 依赖 |
| 交付物 | 单一可执行文件 (≈31MB) | CGO_ENABLED=0 静态编译 |

---

## 从源码构建

```bash
# 构建当前平台
make build

# 交叉编译
make build-win     # Windows x64
make build-linux   # Linux x64

# macOS 交叉编译
GOOS=darwin GOARCH=amd64 make build    # Intel
GOOS=darwin GOARCH=arm64 make build    # Apple Silicon
```

要求：Go 1.26+、Node.js 22+。`make build` 会自动编译前端再嵌入后端。

---

## 许可

MIT © 2026 [kubectl-get-pod](https://github.com/kubectl-get-pod)
