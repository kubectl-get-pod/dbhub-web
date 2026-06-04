import { useEffect } from 'react'
import { Clock, Trash2, Loader } from 'lucide-react'
import { useQueryStore } from '../../stores/queryStore'
import { useTabStore } from '../../stores/tabStore'
import { useUIStore } from '../../stores/uiStore'
import * as api from '../../api/database'

export function QueryHistory() {
  const history = useQueryStore((s) => s.history)
  const setHistory = useQueryStore((s) => s.setHistory)
  const addTab = useTabStore((s) => s.addTab)
  const addToast = useUIStore((s) => s.addToast)

  useEffect(() => {
    api.listHistory().then(setHistory).catch(() => {})
  }, [])

  const handleLoad = (sql: string) => {
    addTab({ type: 'query', title: '历史查询', sql })
    addToast('已加载到新标签页', 'info')
  }

  const handleDelete = async (id: string) => {
    try {
      await api.deleteHistory(id)
      useQueryStore.getState().setHistory(useQueryStore.getState().history.filter((h) => h.id !== id))
    } catch { /* ignore */ }
  }

  const handleClearAll = async () => {
    for (const item of history) {
      await api.deleteHistory(item.id).catch(() => {})
    }
    setHistory([])
    addToast('已清空历史记录', 'info')
  }

  if (history.length === 0) return (
    <div className="p-3 text-xs text-center" style={{ color: 'var(--text-muted)' }}>
      暂无查询历史
    </div>
  )

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-2 py-1 text-xs"
        style={{ borderBottom: '1px solid var(--border-color)', color: 'var(--text-muted)' }}>
        <span className="flex items-center gap-1">
          <Clock size={12} /> 查询历史 ({history.length})
        </span>
        <button onClick={handleClearAll} className="hover:opacity-70">清空</button>
      </div>
      <div className="flex-1 overflow-y-auto">
        {history.map((item) => (
          <div key={item.id} className="group px-2 py-1.5 text-xs cursor-pointer hover:opacity-80"
            style={{ borderBottom: '1px solid var(--border-color)' }}
            onClick={() => handleLoad(item.sql)}>
            <div className="flex items-center justify-between">
              <span className="truncate flex-1 font-mono" style={{ color: 'var(--text-primary)' }}>
                {truncate(item.sql, 60)}
              </span>
              <button onClick={(e) => { e.stopPropagation(); handleDelete(item.id) }}
                className="opacity-0 group-hover:opacity-40 p-0.5 rounded hover:bg-red-500/20"
                style={{ color: 'var(--danger)' }}>
                <Trash2 size={11} />
              </button>
            </div>
            <div className="flex items-center gap-2 mt-0.5" style={{ color: 'var(--text-muted)' }}>
              <span>{item.connName}</span>
              <span>{item.duration}</span>
              <span>{item.createdAt}</span>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}

function truncate(s: string, max: number): string {
  const oneline = s.replace(/\s+/g, ' ').trim()
  return oneline.length > max ? oneline.slice(0, max) + '…' : oneline
}
