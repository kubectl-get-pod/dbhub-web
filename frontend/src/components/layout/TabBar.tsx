import { X } from 'lucide-react'
import { useTabStore } from '../../stores/tabStore'

export function TabBar() {
  const tabs = useTabStore((s) => s.tabs)
  const activeTabId = useTabStore((s) => s.activeTabId)
  const setActive = useTabStore((s) => s.setActive)
  const closeTab = useTabStore((s) => s.closeTab)

  return (
    <div className="h-9 flex items-center flex-shrink-0 overflow-x-auto"
      style={{ backgroundColor: 'var(--bg-secondary)', borderBottom: '1px solid var(--border-color)' }}>
      {tabs.map((tab) => (
        <div
          key={tab.id}
          onClick={() => setActive(tab.id)}
          className="h-full flex items-center px-3 text-xs cursor-pointer select-none whitespace-nowrap border-r group"
          style={{
            backgroundColor: tab.id === activeTabId ? 'var(--bg-primary)' : 'transparent',
            color: tab.id === activeTabId ? 'var(--accent)' : 'var(--text-muted)',
            borderColor: 'var(--border-color)',
          }}
        >
          <span className="mr-1">
            {tab.type === 'schema' ? '📋' : tab.type === 'data' ? '📊' : tab.type === 'query' ? '💻' : tab.type === 'users' ? '👥' : '🏠'}
          </span>
          {tab.title}
          {tab.id !== 'welcome' && (
            <button
              onClick={(e) => { e.stopPropagation(); closeTab(tab.id) }}
              className="ml-2 opacity-0 group-hover:opacity-100 hover:text-red-400"
            >
              <X size={12} />
            </button>
          )}
        </div>
      ))}
    </div>
  )
}
