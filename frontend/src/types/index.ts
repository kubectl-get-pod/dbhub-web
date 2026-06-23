// ========== 连接 ==========

export type DBType = 'mysql' | 'postgres' | 'oracle' | 'mssql'

export interface Connection {
  id: string
  name: string
  group: string
  type: DBType
  host: string
  port: number
  user: string
  password: string
  database: string
  sslMode: string
  sid: string
  useSSH: boolean
  sshHost: string
  sshPort: number
  sshUser: string
  sshPass: string
  sshKey: string
}

export interface ConnectionInfo {
  id: string
  name: string
  type: DBType
  version: string
  status: 'connected' | 'disconnected'
}

// ========== 数据库对象 ==========

export interface Table {
  name: string
  type: 'TABLE' | 'VIEW'
  schema: string
  rowCount: number
}

export interface Column {
  name: string
  dataType: string
  nullable: boolean
  defaultVal: string
  primaryKey: boolean
  comment: string
}

export interface Index {
  name: string
  columns: string[]
  unique: boolean
  type: string
}

export interface ForeignKey {
  name: string
  column: string
  refTable: string
  refColumn: string
}

export interface SchemaData {
  columns: Column[]
  indexes: Index[]
  foreignKeys: ForeignKey[]
  connType: string
}

// ========== 查询 ==========

export interface QueryResult {
  columns: string[]
  rows: unknown[][]
  rowCount: number
  duration: string
  comments?: Record<string, string>
}

export interface QueryHistoryItem {
  id: string
  sql: string
  connName: string
  createdAt: string
  duration: string
}

export interface QueryFavorite {
  id: string
  name: string
  sql: string
  connType: string
  createdAt: string
}

// ========== 用户管理 ==========

export interface DatabaseUser {
  name: string
  host: string
  roles: string[]
}

export interface UserPrivilege {
  user: string
  database: string
  table: string
  privileges: string[]
}

// ========== 自动补全 ==========

export interface AutoCompleteData {
  keywords: string[]
  functions: string[]
  tables: TableRef[]
  columns: ColumnRef[]
}

export interface TableRef {
  name: string
  schema: string
}

export interface ColumnRef {
  table: string
  column: string
  type: string
}

// ========== 标签页 ==========

export type TabType = 'schema' | 'data' | 'query' | 'users' | 'welcome'

export interface Tab {
  id: string
  type: TabType
  title: string
  connId?: string
  database?: string
  table?: string
  sql?: string
}

// ========== 导出/导入 ==========

export interface ExportRequest {
  connId: string
  sql: string
  filename: string
  database?: string
  table?: string
}

export interface ImportRequest {
  connId: string
  database: string
  table: string
  columns: string
}

// ========== API 错误 ==========

export interface ApiError {
  error: string
  detail?: string
}
