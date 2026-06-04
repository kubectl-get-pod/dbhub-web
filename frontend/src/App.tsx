import { useState } from 'react'
import { useUIStore } from './stores/uiStore'
import { useTabStore } from './stores/tabStore'
import { useConnectionStore } from './stores/connectionStore'
import { StatusBar } from './components/layout/StatusBar'
import { TabBar } from './components/layout/TabBar'
import { Toast } from './components/common/Toast'
import { ConnectionList } from './components/connection/ConnectionList'
import { ConnectionForm } from './components/connection/ConnectionForm'
import { TreeBrowser } from './components/browser/TreeBrowser'
import { SchemaViewer } from './components/schema/SchemaViewer'
import { DataGrid } from './components/data/DataGrid'
import { QueryEditor } from './components/query/QueryEditor'
import { QueryFavorites } from './components/query/QueryFavorites'
import { UserList } from './components/users/UserList'

function App() {
  return (
    <ThemeProvider>
      <div className="flex h-screen w-screen"
        style={{ backgroundColor: 'var(--bg-primary)', color: 'var(--text-primary)' }}>
        <Sidebar />
        <MainArea />
        <ThemeToggle />
      </div>
      <Toast />
    </ThemeProvider>
  )
}

function ThemeProvider({ children }: { children: React.ReactNode }) {
  const theme = useUIStore((s) => s.theme)
  return <div className={theme}>{children}</div>
}

function Sidebar() {
  const [formOpen, setFormOpen] = useState(false)
  const hasActive = useConnectionStore((s) => s.activeConnections.size > 0)
  const addTab = useTabStore((s) => s.addTab)

  const handleOpenUsers = () => {
    const conns = Array.from(useConnectionStore.getState().activeConnections.values())
    if (conns.length > 0) {
      addTab({ type: 'users', title: '用户管理', connId: conns[0].id })
    }
  }

  return (
    <aside className="flex-shrink-0 flex flex-col" style={{ width: '280px', backgroundColor: 'var(--bg-secondary)', borderRight: '1px solid var(--border-color)' }}>
      <div className="p-3 flex items-center justify-between">
        <div>
          <div className="text-sm font-bold" style={{ color: 'var(--accent)' }}>dbhub-web</div>
          <div className="text-xs" style={{ color: 'var(--text-muted)' }}>数据库管理工具</div>
        </div>
        <div className="flex gap-1">
          {hasActive && (
            <button onClick={handleOpenUsers}
              className="w-7 h-7 rounded flex items-center justify-center hover:opacity-80 transition-opacity"
              style={{ backgroundColor: 'var(--bg-tertiary)', color: 'var(--text-secondary)' }} title="用户管理">
              👥
            </button>
          )}
          <button onClick={() => setFormOpen(true)}
            className="w-7 h-7 rounded flex items-center justify-center hover:opacity-80 transition-opacity"
            style={{ backgroundColor: 'var(--accent)', color: '#fff' }} title="新建连接">
            +
          </button>
        </div>
      </div>
      <div className="flex-1 overflow-y-auto flex flex-col">
        {/* 保存的连接列表（始终显示） */}
        <div style={hasActive ? { maxHeight: '35%' } : { flex: 1 }}>
          <ConnectionList />
        </div>
        {/* 活动连接的数据库树（有连接时显示） */}
        {hasActive && (
          <div className="flex-1 border-t" style={{ borderColor: 'var(--border-color)' }}>
            <TreeBrowser />
          </div>
        )}
      </div>
      <QueryFavorites />
      <ConnectionForm open={formOpen} onClose={() => setFormOpen(false)} />
    </aside>
  )
}

function MainArea() {
  const tabs = useTabStore((s) => s.tabs)
  const activeTabId = useTabStore((s) => s.activeTabId)
  const activeTab = tabs.find((t) => t.id === activeTabId)

  const renderTab = () => {
    if (!activeTab || activeTab.type === 'welcome') return <WelcomeScreen />
    if (activeTab.type === 'schema') return <SchemaViewer tab={activeTab} />
    if (activeTab.type === 'data') return <DataGrid tab={activeTab} />
    if (activeTab.type === 'query') return <QueryEditor tab={activeTab} />
    if (activeTab.type === 'users') return <UserList tab={activeTab} />
    return (
      <div className="flex-1 h-full flex items-center justify-center" style={{ color: 'var(--text-muted)' }}>
        <div className="text-center">
          <p className="text-sm">功能开发中...</p>
        </div>
      </div>
    )
  }

  return (
    <main className="flex-1 flex flex-col overflow-hidden">
      <TabBar />
      <div className="flex-1 overflow-auto">
        {renderTab()}
      </div>
      <StatusBar />
    </main>
  )
}

function WelcomeScreen() {
  return (
    <div className="flex-1 h-full flex items-center justify-center">
      <div className="text-center">
        <h1 className="text-2xl font-bold mb-2" style={{ color: 'var(--accent)' }}>dbhub-web</h1>
        <p style={{ color: 'var(--text-secondary)' }}>轻量级、零依赖、跨平台数据库管理工具</p>
        <p className="mt-4 text-sm" style={{ color: 'var(--text-muted)' }}>点击左侧 + 新建连接开始使用</p>
      </div>
    </div>
  )
}

function ThemeToggle() {
  const { theme, toggleTheme } = useUIStore()
  return (
    <button onClick={toggleTheme}
      className="fixed bottom-8 right-8 w-10 h-10 rounded-full flex items-center justify-center text-lg shadow-lg hover:scale-110 transition-transform z-50"
      style={{ backgroundColor: 'var(--bg-tertiary)', border: '1px solid var(--border-color)' }}
      title={theme === 'dark' ? '切换到亮色主题' : '切换到暗色主题'}>
      {theme === 'dark' ? '☀️' : '🌙'}
    </button>
  )
}

export default App
