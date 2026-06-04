import { useState, useEffect, useCallback } from 'react'
import { ChevronRight, ChevronDown, Database, Table2, Eye, Search, Server } from 'lucide-react'
import { useConnectionStore } from '../../stores/connectionStore'
import { useTabStore } from '../../stores/tabStore'
import * as api from '../../api/database'
import type { Table, ConnectionInfo } from '../../types'

interface ConnNode {
  info: ConnectionInfo
  databases: string[]
  loaded: boolean
  expanded: boolean
  loading: boolean
}

export function TreeBrowser() {
  const activeConnections = useConnectionStore((s) => s.activeConnections)
  const [connNodes, setConnNodes] = useState<ConnNode[]>([])
  const [search, setSearch] = useState('')

  useEffect(() => {
    const conns = Array.from(activeConnections.values())
    setConnNodes(conns.map((info) => ({
      info,
      databases: [],
      loaded: false,
      expanded: false,
      loading: false,
    })))
  }, [activeConnections])

  const toggleConn = async (idx: number) => {
    const node = connNodes[idx]
    if (node.expanded) {
      setConnNodes(prev => prev.map((n, i) => i === idx ? { ...n, expanded: false } : n))
      return
    }

    if (!node.loaded) {
      setConnNodes(prev => prev.map((n, i) => i === idx ? { ...n, loading: true } : n))
      try {
        const dbs = await api.listDatabases(node.info.id)
        setConnNodes(prev => prev.map((n, i) => i === idx ? { ...n, databases: dbs, loaded: true, expanded: true, loading: false } : n))
      } catch {
        setConnNodes(prev => prev.map((n, i) => i === idx ? { ...n, loading: false } : n))
      }
    } else {
      setConnNodes(prev => prev.map((n, i) => i === idx ? { ...n, expanded: true } : n))
    }
  }

  return (
    <div className="flex flex-col flex-1 overflow-hidden">
      <div className="px-2 py-1.5">
        <div className="flex items-center gap-1 px-2 py-1 rounded text-xs"
          style={{ backgroundColor: 'var(--bg-tertiary)', border: '1px solid var(--border-color)' }}>
          <Search size={12} style={{ color: 'var(--text-muted)' }} />
          <input type="text" value={search} onChange={(e) => setSearch(e.target.value)}
            placeholder="搜索表名..." className="flex-1 bg-transparent outline-none text-xs"
            style={{ color: 'var(--text-primary)' }} />
        </div>
      </div>
      <div className="flex-1 overflow-y-auto text-xs">
        {connNodes.map((node, idx) => (
          <ConnTree key={node.info.id} node={node} idx={idx} toggleConn={toggleConn} search={search} />
        ))}
      </div>
    </div>
  )
}

function ConnTree({ node, idx, toggleConn, search }: { node: ConnNode; idx: number; toggleConn: (i: number) => void; search: string }) {
  return (
    <div>
      <div onClick={() => toggleConn(idx)}
        className="flex items-center px-2 py-1 cursor-pointer hover:opacity-80"
        style={{ color: 'var(--text-primary)' }}>
        {node.loading ? (
          <span className="w-4 h-4 flex items-center justify-center">
            <span className="animate-spin w-3 h-3 border-2 border-t-transparent rounded-full"
              style={{ borderColor: 'var(--accent)', borderTopColor: 'transparent' }} />
          </span>
        ) : node.expanded ? (
          <ChevronDown size={14} style={{ color: 'var(--text-muted)' }} />
        ) : (
          <ChevronRight size={14} style={{ color: 'var(--text-muted)' }} />
        )}
        <Server size={12} className="ml-1 mr-1.5" style={{ color: 'var(--success)' }} />
        <span className="truncate">{node.info.name}</span>
        <span className="ml-1 opacity-40">{node.info.type}</span>
      </div>
      {node.expanded && (
        <div className="ml-3">
          {node.databases.map((db) => (
            <DatabaseNode key={db} connId={node.info.id} database={db} search={search} />
          ))}
        </div>
      )}
    </div>
  )
}

function DatabaseNode({ connId, database, search }: { connId: string; database: string; search: string }) {
  const [expanded, setExpanded] = useState(false)
  const [tables, setTables] = useState<Table[]>([])
  const [loading, setLoading] = useState(false)
  const [loaded, setLoaded] = useState(false)
  const addTab = useTabStore((s) => s.addTab)

  const handleClick = async () => {
    if (expanded) { setExpanded(false); return }
    if (!loaded) {
      setLoading(true)
      try {
        const result = await api.listTables(connId, database)
        setTables(result)
        setLoaded(true)
      } catch { /* ignore */ }
      setLoading(false)
    }
    setExpanded(true)
  }

  const handleOpenQuery = (e: React.MouseEvent) => {
    e.stopPropagation()
    addTab({ type: 'query', title: `查询: ${database}`, connId, database })
  }

  const handleTableDoubleClick = (table: Table) => {
    addTab({ type: 'schema', title: `结构: ${table.name}`, connId, database: database, table: table.name })
    addTab({ type: 'data', title: `数据: ${table.name}`, connId, database: database, table: table.name })
  }

  const filteredTables = tables.filter((t) => {
    if (!search) return true
    return t.name.toLowerCase().includes(search.toLowerCase())
  })

  return (
    <div>
      <div onClick={handleClick}
        className="flex items-center px-2 py-0.5 cursor-pointer hover:opacity-80 group"
        style={{ color: 'var(--text-secondary)' }}>
        {loading ? (
          <span className="w-3 h-3 flex items-center justify-center">
            <span className="animate-spin w-2.5 h-2.5 border-2 border-t-transparent rounded-full"
              style={{ borderColor: 'var(--accent)', borderTopColor: 'transparent' }} />
          </span>
        ) : expanded ? (
          <ChevronDown size={12} style={{ color: 'var(--text-muted)' }} />
        ) : (
          <ChevronRight size={12} style={{ color: 'var(--text-muted)' }} />
        )}
        <Database size={11} className="ml-1 mr-1.5" style={{ color: 'var(--accent)' }} />
        <span className="truncate">{database}</span>
        <button onClick={handleOpenQuery}
          className="ml-auto opacity-0 group-hover:opacity-100 px-1 py-0.5 rounded text-xs hover:opacity-80"
          style={{ color: 'var(--accent)' }} title="在此库中打开查询">
          💻
        </button>
      </div>
      {expanded && (
        <div className="ml-4">
          {filteredTables.map((t) => (
            <div key={t.name}
              onDoubleClick={() => handleTableDoubleClick(t)}
              className="flex items-center px-2 py-0.5 cursor-pointer hover:opacity-80 rounded"
              style={{ color: 'var(--text-primary)' }}>
              {t.type === 'VIEW' ? (
                <Eye size={11} className="mr-1.5" style={{ color: 'var(--warning)' }} />
              ) : (
                <Table2 size={11} className="mr-1.5" style={{ color: 'var(--text-muted)' }} />
              )}
              <span className="truncate">{t.name}</span>
              {t.rowCount > 0 && (
                <span className="ml-auto opacity-40">{fmtNum(t.rowCount)}</span>
              )}
            </div>
          ))}
          {filteredTables.length === 0 && !loading && (
            <div className="px-2 py-1 opacity-40">无表</div>
          )}
        </div>
      )}
    </div>
  )
}

function fmtNum(n: number): string {
  if (n >= 1000000) return (n / 1000000).toFixed(1) + 'M'
  if (n >= 1000) return (n / 1000).toFixed(1) + 'K'
  return String(n)
}
