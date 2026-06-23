import { GET, POST, PUT, DELETE } from './client'
import type { Table, SchemaData, QueryResult, AutoCompleteData, DatabaseUser, UserPrivilege, QueryHistoryItem, QueryFavorite } from '../types'

// 浏览
export function listDatabases(connId: string) { return GET<string[]>('/databases/' + connId) }
export function listTables(connId: string, database: string) { return GET<Table[]>('/tables/' + connId + '/' + encodeURIComponent(database)) }

// 结构
export function getSchema(connId: string, database: string, table: string) {
  return GET<SchemaData>('/schema/' + connId + '/' + encodeURIComponent(database) + '/' + encodeURIComponent(table))
}
export function getDDL(connId: string, database: string, table: string) {
  return GET<{ ddl: string }>('/ddl/' + connId + '/' + encodeURIComponent(database) + '/' + encodeURIComponent(table))
}

// 结构修改
export function alterColumn(connId: string, database: string, table: string, change: unknown) {
  return PUT('/schema/' + connId + '/' + encodeURIComponent(database) + '/' + encodeURIComponent(table) + '/column', change)
}
export function addColumn(connId: string, database: string, table: string, col: unknown) {
  return POST('/schema/' + connId + '/' + encodeURIComponent(database) + '/' + encodeURIComponent(table) + '/column', col)
}
export function dropColumn(connId: string, database: string, table: string, column: string) {
  return DELETE('/schema/' + connId + '/' + encodeURIComponent(database) + '/' + encodeURIComponent(table) + '/column')
}

// 数据
export function listData(connId: string, database: string, table: string, limit = 50, offset = 0) {
  return GET<QueryResult>('/data/' + connId + '/' + encodeURIComponent(database) + '/' + encodeURIComponent(table) + '?limit=' + limit + '&offset=' + offset)
}
export function getRowCount(connId: string, database: string, table: string) {
  return GET<{ count: number }>('/rowcount/' + connId + '/' + encodeURIComponent(database) + '/' + encodeURIComponent(table))
}
export function insertRow(connId: string, database: string, table: string, row: Record<string, unknown>) {
  return POST('/data/' + connId + '/' + encodeURIComponent(database) + '/' + encodeURIComponent(table), row)
}
export function updateRow(connId: string, database: string, table: string, pk: Record<string, unknown>, values: Record<string, unknown>) {
  return PUT('/data/' + connId + '/' + encodeURIComponent(database) + '/' + encodeURIComponent(table), { pk, values })
}
export function deleteRow(connId: string, database: string, table: string, pk: Record<string, unknown>) {
  return DELETE('/data/' + connId + '/' + encodeURIComponent(database) + '/' + encodeURIComponent(table))
}

// 查询
export function executeQuery(connId: string, sql: string, limit = 0, offset = 0, database?: string, table?: string) {
  return POST<QueryResult>('/query', { connId, sql, limit, offset, database: database || '', table: table || '' })
}

// 历史
export function listHistory() { return GET<QueryHistoryItem[]>('/history') }
export function deleteHistory(id: string) { return DELETE('/history/' + id) }

// 收藏
export function listFavorites() { return GET<QueryFavorite[]>('/favorites') }
export function createFavorite(name: string, sql: string, connType: string) {
  return POST<QueryFavorite>('/favorites', { name, sql, connType })
}
export function deleteFavorite(id: string) { return DELETE<{ status: string }>('/favorites/' + id) }

// 自动补全
export function getAutoComplete(connId: string, database: string) {
  return GET<AutoCompleteData>('/autocomplete/' + connId + '/' + encodeURIComponent(database))
}

// 用户
export function listUsers(connId: string) { return GET<DatabaseUser[]>('/users/' + connId) }
export function createUser(connId: string, user: string, password: string) {
  return POST('/users/' + connId, { user, password })
}
export function deleteUser(connId: string, user: string) { return DELETE('/users/' + connId + '/' + user) }
export function getPrivileges(connId: string, user: string) {
  return GET<UserPrivilege[]>('/privileges/' + connId + '/' + user)
}
export function grantPrivilege(connId: string, user: string, database: string, table: string, privileges: string[]) {
  return POST('/privileges/' + connId + '/' + user + '/grant', { database, table, privileges })
}
export function revokePrivilege(connId: string, user: string, database: string, table: string, privileges: string[]) {
  return POST('/privileges/' + connId + '/' + user + '/revoke', { database, table, privileges })
}

// 导出
export function exportCSV(connId: string, sql: string, filename: string, database?: string, table?: string) {
  return fetch('/api/export/csv', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ connId, sql, filename, database, table }) })
    .then(r => r.blob()).then(b => { const a = document.createElement('a'); a.href = URL.createObjectURL(b); a.download = filename; a.click() })
}
export function exportExcel(connId: string, sql: string, filename: string, database?: string, table?: string) {
  return fetch('/api/export/excel', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ connId, sql, filename, database, table }) })
    .then(r => r.blob()).then(b => { const a = document.createElement('a'); a.href = URL.createObjectURL(b); a.download = filename; a.click() })
}

// 系统
export function getVersion(connId: string) { return GET<{ version: string }>('/version/' + connId) }
