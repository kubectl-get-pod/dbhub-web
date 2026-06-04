import { create } from 'zustand'
import type { Connection, ConnectionInfo, DBType } from '../types'

interface ConnectionState {
  connections: Connection[]
  activeConnections: Map<string, ConnectionInfo>
  loading: boolean
  setConnections: (conns: Connection[]) => void
  addConnection: (conn: Connection) => void
  removeConnection: (id: string) => void
  updateConnection: (id: string, conn: Connection) => void
  setActive: (id: string, info: ConnectionInfo) => void
  removeActive: (id: string) => void
  setLoading: (v: boolean) => void
}

export const useConnectionStore = create<ConnectionState>((set) => ({
  connections: [],
  activeConnections: new Map(),
  loading: false,
  setConnections: (conns) => set({ connections: conns }),
  addConnection: (conn) => set((s) => ({ connections: [...s.connections, conn] })),
  removeConnection: (id) => set((s) => ({
    connections: s.connections.filter((c) => c.id !== id),
    activeConnections: (() => { const m = new Map(s.activeConnections); m.delete(id); return m })(),
  })),
  updateConnection: (id, conn) => set((s) => ({
    connections: s.connections.map((c) => (c.id === id ? conn : c)),
  })),
  setActive: (id, info) => set((s) => {
    const m = new Map(s.activeConnections); m.set(id, info); return { activeConnections: m }
  }),
  removeActive: (id) => set((s) => {
    const m = new Map(s.activeConnections); m.delete(id); return { activeConnections: m }
  }),
  setLoading: (v) => set({ loading: v }),
}))
