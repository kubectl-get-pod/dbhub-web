import { GET, POST, PUT, DELETE } from './client'
import type { Connection, ConnectionInfo } from '../types'

export function listConnections() {
  return GET<Connection[]>('/connections')
}

export function createConnection(conn: Omit<Connection, 'id'>) {
  return POST<Connection>('/connections', conn)
}

export function updateConnection(id: string, conn: Partial<Connection>) {
  return PUT<Connection>('/connections/' + id, conn)
}

export function deleteConnection(id: string) {
  return DELETE<{ status: string }>('/connections/' + id)
}

export function testConnection(conn: Omit<Connection, 'id'>) {
  return POST<{ status: string; version: string }>('/connections/test', conn)
}

export function connectDB(id: string) {
  return POST<ConnectionInfo>('/connect/' + id)
}

export function disconnectDB(id: string) {
  return POST<{ status: string }>('/disconnect/' + id)
}
