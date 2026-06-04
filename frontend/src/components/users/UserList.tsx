import { useState, useEffect } from 'react'
import { Plus, Trash2, Shield } from 'lucide-react'
import { Modal } from '../common/Modal'
import { Spinner } from '../common/Spinner'
import { ConfirmDialog } from '../common/ConfirmDialog'
import { useUIStore } from '../../stores/uiStore'
import * as api from '../../api/database'
import type { Tab, DatabaseUser, UserPrivilege } from '../../types'

interface Props {
  tab: Tab
}

export function UserList({ tab }: Props) {
  const [users, setUsers] = useState<DatabaseUser[]>([])
  const [loading, setLoading] = useState(true)
  const [createOpen, setCreateOpen] = useState(false)
  const [privOpenUser, setPrivOpenUser] = useState<string | null>(null)
  const [confirmDelete, setConfirmDelete] = useState<string | null>(null)
  const addToast = useUIStore((s) => s.addToast)

  const load = async () => {
    if (!tab.connId) return
    setLoading(true)
    try { setUsers(await api.listUsers(tab.connId)) } catch { /* ignore */ }
    setLoading(false)
  }

  useEffect(() => { load() }, [tab.connId])

  const handleDelete = (user: string) => {
    setConfirmDelete(user)
  }

  const doDelete = async () => {
    if (!tab.connId || !confirmDelete) return
    try {
      await api.deleteUser(tab.connId, confirmDelete)
      addToast(`用户 ${confirmDelete} 已删除`, 'info')
      setConfirmDelete(null)
      load()
    } catch (e) {
      addToast(`删除失败: ${(e as Error).message}`, 'error')
    }
  }

  if (loading) return <div className="p-8 text-center"><Spinner size={20} /></div>

  return (
    <div className="p-4 overflow-auto h-full">
      <div className="flex items-center justify-between mb-3">
        <h3 className="text-sm font-semibold" style={{ color: 'var(--accent)' }}>用户管理</h3>
        <button onClick={() => setCreateOpen(true)}
          className="flex items-center gap-1 px-2 py-1 rounded text-xs"
          style={{ backgroundColor: 'var(--accent)', color: '#fff' }}>
          <Plus size={12} /> 创建用户
        </button>
      </div>

      <table className="w-full text-xs border-collapse" style={{ border: '1px solid var(--border-color)' }}>
        <thead>
          <tr style={{ backgroundColor: 'var(--bg-tertiary)' }}>
            <th className="px-2 py-1.5 text-left">用户名</th>
            <th className="px-2 py-1.5 text-left">主机</th>
            <th className="px-2 py-1.5 text-left">角色</th>
            <th className="px-2 py-1.5 w-20">操作</th>
          </tr>
        </thead>
        <tbody>
          {users.map((u) => (
            <tr key={u.name} style={{ borderTop: '1px solid var(--border-color)' }}>
              <td className="px-2 py-1" style={{ color: 'var(--text-primary)', fontFamily: 'monospace' }}>{u.name}</td>
              <td className="px-2 py-1" style={{ color: 'var(--text-muted)' }}>{u.host || '-'}</td>
              <td className="px-2 py-1" style={{ color: 'var(--text-secondary)' }}>{(u.roles || []).join(', ') || '-'}</td>
              <td className="px-2 py-1">
                <div className="flex gap-1">
                  <button onClick={() => setPrivOpenUser(u.name)}
                    className="p-0.5 rounded hover:opacity-70" title="权限" style={{ color: 'var(--text-muted)' }}>
                    <Shield size={12} />
                  </button>
                  <button onClick={() => handleDelete(u.name)}
                    className="p-0.5 rounded hover:bg-red-500/20" style={{ color: 'var(--text-muted)' }}>
                    <Trash2 size={12} />
                  </button>
                </div>
              </td>
            </tr>
          ))}
        </tbody>
      </table>

      {createOpen && <CreateUserDialog tab={tab} onClose={() => setCreateOpen(false)} onCreated={load} />}
      {privOpenUser && <PrivilegeEditor tab={tab} user={privOpenUser} onClose={() => setPrivOpenUser(null)} />}

      <ConfirmDialog
        open={confirmDelete !== null}
        title="确认删除用户"
        message={`确定要删除数据库用户 "${confirmDelete}" 吗？此操作不可撤销。`}
        confirmText="删除"
        danger
        onConfirm={doDelete}
        onCancel={() => setConfirmDelete(null)}
      />
    </div>
  )
}

function CreateUserDialog({ tab, onClose, onCreated }: { tab: Tab; onClose: () => void; onCreated: () => void }) {
  const [user, setUser] = useState('')
  const [password, setPassword] = useState('')
  const addToast = useUIStore((s) => s.addToast)

  const handleCreate = async () => {
    if (!tab.connId || !user || !password) return
    try {
      await api.createUser(tab.connId, user, password)
      addToast(`用户 ${user} 已创建`, 'success')
      onCreated()
      onClose()
    } catch (e) {
      addToast(`创建失败: ${(e as Error).message}`, 'error')
    }
  }

  return (
    <Modal open onClose={onClose} title="创建用户" width="360px">
      <div className="space-y-3 text-sm">
        <div>
          <label className="text-xs" style={{ color: 'var(--text-muted)' }}>用户名</label>
          <input value={user} onChange={(e) => setUser(e.target.value)}
            className="w-full px-2 py-1.5 rounded mt-1 text-xs outline-none"
            style={{ backgroundColor: 'var(--bg-tertiary)', color: 'var(--text-primary)', border: '1px solid var(--border-color)' }} />
        </div>
        <div>
          <label className="text-xs" style={{ color: 'var(--text-muted)' }}>密码</label>
          <input type="password" value={password} onChange={(e) => setPassword(e.target.value)}
            className="w-full px-2 py-1.5 rounded mt-1 text-xs outline-none"
            style={{ backgroundColor: 'var(--bg-tertiary)', color: 'var(--text-primary)', border: '1px solid var(--border-color)' }} />
        </div>
        <div className="flex gap-2 justify-end">
          <button onClick={onClose}
            className="px-3 py-1.5 rounded text-xs" style={{ backgroundColor: 'var(--bg-tertiary)', color: 'var(--text-secondary)' }}>取消</button>
          <button onClick={handleCreate}
            className="px-3 py-1.5 rounded text-xs" style={{ backgroundColor: 'var(--accent)', color: '#fff' }}>创建</button>
        </div>
      </div>
    </Modal>
  )
}

function PrivilegeEditor({ tab, user, onClose }: { tab: Tab; user: string; onClose: () => void }) {
  const [privs, setPrivs] = useState<UserPrivilege[]>([])
  const [loading, setLoading] = useState(true)
  const [grantOpen, setGrantOpen] = useState(false)
  const [grantDB, setGrantDB] = useState('')
  const [grantTable, setGrantTable] = useState('*')
  const [grantPrivs, setGrantPrivs] = useState<string[]>(['SELECT'])
  const addToast = useUIStore((s) => s.addToast)

  useEffect(() => {
    if (!tab.connId) return
    api.getPrivileges(tab.connId, user).then(setPrivs).catch(() => {}).finally(() => setLoading(false))
  }, [tab.connId, user])

  const allPrivOptions = ['SELECT', 'INSERT', 'UPDATE', 'DELETE', 'CREATE', 'DROP', 'GRANT', 'ALTER']

  const handleGrant = async () => {
    if (!tab.connId || !grantDB) return
    try {
      await api.grantPrivilege(tab.connId, user, grantDB, grantTable, grantPrivs)
      addToast('权限已授予', 'success')
      setGrantOpen(false)
      const updated = await api.getPrivileges(tab.connId, user)
      setPrivs(updated)
    } catch (e) {
      addToast(`授予失败: ${(e as Error).message}`, 'error')
    }
  }

  if (loading) return <Modal open onClose={onClose} title={`权限: ${user}`}><Spinner /></Modal>

  return (
    <Modal open onClose={onClose} title={`权限: ${user}`} width="500px">
      <div className="space-y-3 text-xs">
        {privs.map((p, i) => (
          <div key={i} className="p-2 rounded" style={{ backgroundColor: 'var(--bg-tertiary)' }}>
            <span style={{ color: 'var(--accent)' }}>{p.database}.{p.table}</span>
            <span className="ml-2" style={{ color: 'var(--text-secondary)' }}>{p.privileges.join(', ')}</span>
          </div>
        ))}
        {privs.length === 0 && <p style={{ color: 'var(--text-muted)' }}>暂无权限</p>}
      </div>
      <div className="flex gap-2 mt-3">
        {!grantOpen ? (
          <button onClick={() => setGrantOpen(true)}
            className="px-2 py-1 rounded text-xs" style={{ backgroundColor: 'var(--accent)', color: '#fff' }}>授予权限</button>
        ) : (
          <div className="flex-1 space-y-2">
            <div className="flex gap-2">
              <div className="flex-1">
                <input value={grantDB} onChange={(e) => setGrantDB(e.target.value)} placeholder="数据库"
                  className="w-full px-2 py-1 rounded text-xs outline-none"
                  style={{ backgroundColor: 'var(--bg-tertiary)', color: 'var(--text-primary)', border: '1px solid var(--border-color)' }} />
              </div>
              <div className="flex-1">
                <input value={grantTable} onChange={(e) => setGrantTable(e.target.value)} placeholder="表名 (* = 全部)"
                  className="w-full px-2 py-1 rounded text-xs outline-none"
                  style={{ backgroundColor: 'var(--bg-tertiary)', color: 'var(--text-primary)', border: '1px solid var(--border-color)' }} />
              </div>
            </div>
            <div className="flex flex-wrap gap-1">
              {allPrivOptions.map((p) => (
                <label key={p} className="flex items-center gap-0.5 text-xs cursor-pointer" style={{ color: 'var(--text-secondary)' }}>
                  <input type="checkbox" checked={grantPrivs.includes(p)}
                    onChange={(e) => setGrantPrivs(e.target.checked ? [...grantPrivs, p] : grantPrivs.filter((x) => x !== p))}
                    className="accent-current" /> {p}
                </label>
              ))}
            </div>
            <div className="flex gap-2">
              <button onClick={() => setGrantOpen(false)}
                className="px-2 py-1 rounded text-xs" style={{ backgroundColor: 'var(--bg-tertiary)', color: 'var(--text-secondary)' }}>取消</button>
              <button onClick={handleGrant}
                className="px-2 py-1 rounded text-xs" style={{ backgroundColor: 'var(--accent)', color: '#fff' }}>确认授予</button>
            </div>
          </div>
        )}
      </div>
    </Modal>
  )
}
