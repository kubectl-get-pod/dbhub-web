import { useState, useEffect } from 'react'
import { Database, Plug, Trash2, Pen, ChevronRight, ChevronDown, Pencil, Check, X } from 'lucide-react'
import { useConnectionStore } from '../../stores/connectionStore'
import { useTabStore } from '../../stores/tabStore'
import { useUIStore } from '../../stores/uiStore'
import * as api from '../../api/connections'
import { ConnectionForm } from './ConnectionForm'
import type { Connection, ConnectionInfo } from '../../types'

export function ConnectionList() {
  const connections = useConnectionStore((s) => s.connections)
  const updateConnection = useConnectionStore((s) => s.updateConnection)
  const setConnections = useConnectionStore((s) => s.setConnections)
  const activeConnections = useConnectionStore((s) => s.activeConnections)
  const setActive = useConnectionStore((s) => s.setActive)
  const removeActive = useConnectionStore((s) => s.removeActive)
  const removeConnection = useConnectionStore((s) => s.removeConnection)
  const addToast = useUIStore((s) => s.addToast)
  const addTab = useTabStore((s) => s.addTab)

  const [formOpen, setFormOpen] = useState(false)
  const [editConn, setEditConn] = useState<Connection | null>(null)
  const [collapsedGroups, setCollapsedGroups] = useState<Set<string>>(new Set())
  const [renamingGroup, setRenamingGroup] = useState<string | null>(null)
  const [renameValue, setRenameValue] = useState('')

  useEffect(() => {
    api.listConnections().then(setConnections).catch(() => {})
  }, [])

  const groups = new Map<string, Connection[]>()
  for (const c of connections) {
    const g = c.group || '未分组'
    if (!groups.has(g)) groups.set(g, [])
    groups.get(g)!.push(c)
  }

  const toggleGroup = (group: string) => {
    setCollapsedGroups((prev) => {
      const next = new Set(prev)
      if (next.has(group)) next.delete(group)
      else next.add(group)
      return next
    })
  }

  const startRename = (group: string) => {
    setRenamingGroup(group)
    setRenameValue(group === '未分组' ? '' : group)
  }

  const saveRename = async () => {
    if (!renamingGroup || !renameValue.trim() || renameValue === renamingGroup) {
      setRenamingGroup(null)
      return
    }
    // 更新该组下所有连接的 group 字段
    const groupConns = groups.get(renamingGroup) || []
    for (const conn of groupConns) {
      try {
        await api.updateConnection(conn.id, { ...conn, group: renameValue.trim() })
        updateConnection(conn.id, { ...conn, group: renameValue.trim() })
      } catch { /* ignore */ }
    }
    setRenamingGroup(null)
    addToast(`分组已重命名为 ${renameValue.trim()}`, 'success')
  }

  const handleConnect = async (conn: Connection) => {
    try {
      const info = await api.connectDB(conn.id)
      setActive(conn.id, info as ConnectionInfo)
      addToast(`已连接 ${conn.name}`, 'success')
    } catch (e) {
      addToast(`连接失败: ${(e as Error).message}`, 'error')
    }
  }

  const handleDisconnect = async (id: string) => {
    try {
      await api.disconnectDB(id)
      removeActive(id)
      addToast('已断开连接', 'info')
    } catch (e) {
      addToast(`断开失败: ${(e as Error).message}`, 'error')
    }
  }

  const handleDelete = async (id: string) => {
    await api.deleteConnection(id)
    removeConnection(id)
    removeActive(id)
    addToast('连接已删除', 'info')
  }

  const handleEdit = (conn: Connection) => {
    setEditConn(conn)
    setFormOpen(true)
  }

  const handleFormClose = () => {
    setFormOpen(false)
    setEditConn(null)
  }

  const isConnected = (id: string) => activeConnections.has(id)

  if (connections.length === 0) {
    return (
      <>
        <div className="px-2 py-4 text-xs text-center" style={{ color: 'var(--text-muted)' }}>
          暂无连接，点击 + 新建
        </div>
        <ConnectionForm open={formOpen} onClose={handleFormClose} editConnection={editConn} />
      </>
    )
  }

  return (
    <>
      {Array.from(groups.entries()).map(([groupName, conns]) => {
        const isCollapsed = collapsedGroups.has(groupName)
        return (
          <div key={groupName} className="mb-0.5">
            {/* 分组头（可折叠） */}
            <div onClick={() => toggleGroup(groupName)}
              className="flex items-center px-3 py-1 text-xs cursor-pointer hover:opacity-80 group"
              style={{ color: 'var(--text-muted)' }}>
              {isCollapsed ? <ChevronRight size={12} /> : <ChevronDown size={12} />}
              {renamingGroup === groupName ? (
                <span className="flex items-center gap-1 flex-1 ml-1" onClick={(e) => e.stopPropagation()}>
                  <input
                    autoFocus
                    value={renameValue}
                    onChange={(e) => setRenameValue(e.target.value)}
                    onKeyDown={(e) => { if (e.key === 'Enter') saveRename(); if (e.key === 'Escape') setRenamingGroup(null) }}
                    className="w-24 px-1 py-0 rounded text-xs outline-none"
                    style={{ backgroundColor: 'var(--bg-tertiary)', color: 'var(--text-primary)', border: '1px solid var(--accent)' }}
                  />
                  <button onClick={saveRename} className="p-0.5 rounded hover:opacity-70" style={{ color: 'var(--success)' }}><Check size={11} /></button>
                  <button onClick={() => setRenamingGroup(null)} className="p-0.5 rounded hover:opacity-70" style={{ color: 'var(--danger)' }}><X size={11} /></button>
                </span>
              ) : (
                <>
                  <span className="ml-1 flex-1">📁 {groupName}</span>
                  <span className="opacity-40 mr-1">({conns.length})</span>
                  <button
                    onClick={(e) => { e.stopPropagation(); startRename(groupName) }}
                    className="opacity-0 group-hover:opacity-50 p-0.5 rounded hover:opacity-80"
                    title="重命名分组">
                    <Pencil size={10} />
                  </button>
                </>
              )}
            </div>

            {/* 分组下的连接列表（折叠时隐藏） */}
            {!isCollapsed && conns.map((conn) => (
              <div key={conn.id} className="group flex items-center px-3 py-1 ml-2 rounded cursor-pointer text-xs"
                style={{ color: isConnected(conn.id) ? 'var(--success)' : 'var(--text-secondary)' }}
                onClick={() => isConnected(conn.id)
                  ? addTab({ type: 'query', title: `查询: ${conn.name}`, connId: conn.id, database: conn.database })
                  : handleConnect(conn)}>
                <Database size={11} className="mr-1.5 flex-shrink-0" />
                <span className="truncate flex-1">{conn.name}</span>
                <span className="text-xs ml-1 opacity-40">{conn.type}</span>
                <div className="hidden group-hover:flex items-center gap-0.5 ml-1">
                  {isConnected(conn.id) ? (
                    <button onClick={(e) => { e.stopPropagation(); handleDisconnect(conn.id) }}
                      title="断开" className="p-0.5 rounded hover:bg-red-500/20">
                      <Plug size={11} />
                    </button>
                  ) : null}
                  <button onClick={(e) => { e.stopPropagation(); handleEdit(conn) }}
                    title="编辑" className="p-0.5 rounded hover:opacity-70">
                    <Pen size={11} />
                  </button>
                  <button onClick={(e) => { e.stopPropagation(); handleDelete(conn.id) }}
                    title="删除" className="p-0.5 rounded hover:bg-red-500/20">
                    <Trash2 size={11} />
                  </button>
                </div>
              </div>
            ))}
          </div>
        )
      })}
      <ConnectionForm open={formOpen} onClose={handleFormClose} editConnection={editConn} />
    </>
  )
}
