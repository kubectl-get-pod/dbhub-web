import { useState } from 'react'
import { Modal } from '../common/Modal'
import { Spinner } from '../common/Spinner'
import { useConnectionStore } from '../../stores/connectionStore'
import { useUIStore } from '../../stores/uiStore'
import * as api from '../../api/connections'
import type { Connection, DBType } from '../../types'

interface Props {
  open: boolean
  onClose: () => void
  editConnection?: Connection | null
}

const DB_DEFAULTS: Record<DBType, { port: number; ssl: boolean; sid: boolean }> = {
  mysql:    { port: 3306, ssl: false, sid: false },
  postgres: { port: 5432, ssl: true,  sid: false },
  oracle:   { port: 1521, ssl: false, sid: true },
  mssql:    { port: 1433, ssl: false, sid: false },
}

export function ConnectionForm({ open, onClose, editConnection }: Props) {
  const addConnection = useConnectionStore((s) => s.addConnection)
  const updateConnection = useConnectionStore((s) => s.updateConnection)
  const addToast = useUIStore((s) => s.addToast)

  const [type, setType] = useState<DBType>(editConnection?.type || 'mysql')
  const [name, setName] = useState(editConnection?.name || '')
  const [group, setGroup] = useState(editConnection?.group || '')
  const [host, setHost] = useState(editConnection?.host || 'localhost')
  const [port, setPort] = useState(editConnection?.port || DB_DEFAULTS.mysql.port)
  const [user, setUser] = useState(editConnection?.user || 'root')
  const [password, setPassword] = useState(editConnection?.password || '')
  const [database, setDatabase] = useState(editConnection?.database || '')
  const [sslMode, setSSLMode] = useState(editConnection?.sslMode || 'disable')
  const [sid, setSID] = useState(editConnection?.sid || '')
  const [useSSH, setUseSSH] = useState(editConnection?.useSSH || false)
  const [sshHost, setSSHHost] = useState(editConnection?.sshHost || '')
  const [sshPort, setSSHPort] = useState(editConnection?.sshPort || 22)
  const [sshUser, setSSHUser] = useState(editConnection?.sshUser || '')
  const [sshPass, setSSHPass] = useState(editConnection?.sshPass || '')
  const [sshKey, setSSHKey] = useState(editConnection?.sshKey || '')
  const [testing, setTesting] = useState(false)
  const [saving, setSaving] = useState(false)

  const defaults = DB_DEFAULTS[type]

  const handleTypeChange = (t: DBType) => {
    setType(t)
    setPort(DB_DEFAULTS[t].port)
  }

  const buildConnection = (): Omit<Connection, 'id'> => ({
    name, group, type, host, port, user, password, database,
    sslMode: defaults.ssl ? sslMode : '',
    sid: defaults.sid ? sid : '',
    useSSH,
    sshHost, sshPort, sshUser, sshPass, sshKey,
  })

  const handleTest = async () => {
    setTesting(true)
    try {
      const result = await api.testConnection(buildConnection())
      addToast(`连接成功! 版本: ${result.version}`, 'success')
    } catch (e) {
      addToast(`连接失败: ${(e as Error).message}`, 'error')
    } finally {
      setTesting(false)
    }
  }

  const handleSave = async () => {
    if (!name || !host || !port || !user) {
      addToast('请填写所有必填项（标记 * 的字段）', 'error')
      return
    }
    setSaving(true)
    try {
      if (editConnection) {
        await api.updateConnection(editConnection.id, buildConnection())
        updateConnection(editConnection.id, { ...buildConnection(), id: editConnection.id } as Connection)
        addToast('连接已更新', 'success')
      } else {
        const conn = await api.createConnection(buildConnection())
        addConnection({ ...buildConnection(), id: conn.id } as Connection)
        addToast('连接已创建', 'success')
      }
      onClose()
    } catch (e) {
      addToast(`保存失败: ${(e as Error).message}`, 'error')
    } finally {
      setSaving(false)
    }
  }

  return (
    <Modal open={open} onClose={onClose} title={editConnection ? '编辑连接' : '新建连接'} width="520px">
      <div className="space-y-3 text-sm">
        {/* 数据库类型选择 */}
        <div className="flex gap-2">
          {(['mysql', 'postgres', 'oracle', 'mssql'] as DBType[]).map((t) => (
            <button key={t} onClick={() => handleTypeChange(t)}
              className="flex-1 py-1.5 rounded text-xs font-medium transition-colors"
              style={{
                backgroundColor: type === t ? 'var(--accent)' : 'var(--bg-tertiary)',
                color: type === t ? '#fff' : 'var(--text-secondary)',
              }}>
              {t === 'mysql' ? 'MySQL' : t === 'postgres' ? 'PostgreSQL' : t === 'oracle' ? 'Oracle' : 'SQL Server'}
            </button>
          ))}
        </div>

        {/* 基本信息 */}
        <Field label="连接名称" value={name} onChange={setName} placeholder="我的数据库" required />
        <Field label="分组" value={group} onChange={setGroup} placeholder="如：开发环境（可选）" optional />

        <div className="flex gap-2">
          <div className="flex-1"><Field label="主机" value={host} onChange={setHost} placeholder="localhost" required /></div>
          <div className="w-24"><Field label="端口" value={String(port)} onChange={(v) => setPort(Number(v) || 0)} required /></div>
        </div>

        <div className="flex gap-2">
          <div className="flex-1"><Field label="用户名" value={user} onChange={setUser} required /></div>
          <div className="flex-1"><Field label="密码" value={password} onChange={setPassword} type="password" required /></div>
        </div>

        <Field label="数据库" value={database} onChange={setDatabase} placeholder="连接成功后浏览所有数据库（可选）" optional />

        {/* PostgreSQL SSL */}
        {defaults.ssl && (
          <div>
            <label className="text-xs mb-1 block" style={{ color: 'var(--text-muted)' }}>SSL 模式（可选）</label>
            <select value={sslMode} onChange={(e) => setSSLMode(e.target.value)}
              className="w-full px-3 py-1.5 rounded text-xs outline-none"
              style={{ backgroundColor: 'var(--bg-tertiary)', color: 'var(--text-primary)', border: '1px solid var(--border-color)' }}>
              {['disable', 'allow', 'prefer', 'require', 'verify-ca', 'verify-full'].map((m) => (
                <option key={m} value={m}>{m}</option>
              ))}
            </select>
          </div>
        )}

        {/* Oracle SID */}
        {defaults.sid && <Field label="SID" value={sid} onChange={setSID} placeholder="留空使用 Service Name（可选）" optional />}

        {/* SSH 隧道 */}
        <div>
          <label className="flex items-center gap-2 text-xs cursor-pointer" style={{ color: 'var(--text-muted)' }}
            onClick={() => setUseSSH(!useSSH)}>
            <input type="checkbox" checked={useSSH} onChange={(e) => setUseSSH(e.target.checked)}
              className="accent-current" />
            SSH 隧道
          </label>
          {useSSH && (
            <div className="mt-2 p-3 rounded space-y-2" style={{ backgroundColor: 'var(--bg-secondary)', border: '1px solid var(--border-color)' }}>
              <div className="flex gap-2">
                <div className="flex-1"><Field label="SSH 主机" value={sshHost} onChange={setSSHHost} required /></div>
                <div className="w-24"><Field label="SSH 端口" value={String(sshPort)} onChange={(v) => setSSHPort(Number(v) || 22)} required /></div>
              </div>
              <div className="flex gap-2">
                <div className="flex-1"><Field label="SSH 用户" value={sshUser} onChange={setSSHUser} required /></div>
                <div className="flex-1"><Field label="SSH 密码" value={sshPass} onChange={setSSHPass} type="password" placeholder="密码或私钥二选一" optional /></div>
              </div>
              <Field label="SSH 私钥路径" value={sshKey} onChange={setSSHKey} placeholder="~/.ssh/id_rsa（可选）" optional />
            </div>
          )}
        </div>

        {/* 操作按钮 */}
        <div className="flex gap-2 pt-2">
          <button onClick={handleTest} disabled={testing}
            className="px-4 py-1.5 rounded text-xs flex items-center gap-2"
            style={{ backgroundColor: 'var(--bg-tertiary)', color: 'var(--text-primary)', border: '1px solid var(--border-color)' }}>
            {testing ? <Spinner size={14} /> : null}
            测试连接
          </button>
          <div className="flex-1" />
          <button onClick={onClose}
            className="px-4 py-1.5 rounded text-xs"
            style={{ backgroundColor: 'var(--bg-tertiary)', color: 'var(--text-secondary)' }}>
            取消
          </button>
          <button onClick={handleSave} disabled={saving}
            className="px-4 py-1.5 rounded text-xs flex items-center gap-2"
            style={{ backgroundColor: 'var(--accent)', color: '#fff' }}>
            {saving ? <Spinner size={14} /> : null}
            {editConnection ? '保存' : '创建'}
          </button>
        </div>
      </div>
    </Modal>
  )
}

function Field({ label, value, onChange, type = 'text', placeholder, required, optional }: {
  label: string; value: string; onChange: (v: string) => void; type?: string; placeholder?: string; required?: boolean; optional?: boolean
}) {
  return (
    <div>
      <label className="text-xs mb-1 flex items-center gap-1" style={{ color: 'var(--text-muted)' }}>
        {label}
        {required && <span style={{ color: 'var(--danger)' }}>*</span>}
        {optional && <span className="text-xs" style={{ color: 'var(--text-muted)', opacity: 0.6 }}>(可选)</span>}
      </label>
      <input type={type} value={value} onChange={(e) => onChange(e.target.value)} placeholder={placeholder}
        className="w-full px-3 py-1.5 rounded text-xs outline-none"
        style={{ backgroundColor: 'var(--bg-tertiary)', color: 'var(--text-primary)', border: '1px solid var(--border-color)' }} />
    </div>
  )
}
