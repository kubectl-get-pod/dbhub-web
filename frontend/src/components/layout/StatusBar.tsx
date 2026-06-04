import { useTabStore } from '../../stores/tabStore'
import { useConnectionStore } from '../../stores/connectionStore'

export function StatusBar() {
  const active = useConnectionStore((s) => s.activeConnections)
  const tabs = useTabStore((s) => s.tabs)
  const activeTabId = useTabStore((s) => s.activeTabId)

  const activeTab = tabs.find((t) => t.id === activeTabId)
  const connected = Array.from(active.values())

  return (
    <div className="h-6 flex items-center px-3 text-xs flex-shrink-0 gap-4"
      style={{ backgroundColor: 'var(--bg-secondary)', color: 'var(--text-muted)', borderTop: '1px solid var(--border-color)' }}>
      <span>{connected.length > 0 ? `已连接 ${connected.length} 个数据库` : '未连接'}</span>
      {connected.map((c) => (
        <span key={c.id}>🟢 {c.name} ({c.type})</span>
      ))}
      <span className="ml-auto">{activeTab?.title || ''}</span>
    </div>
  )
}
