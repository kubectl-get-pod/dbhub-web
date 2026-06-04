import { useState, useEffect, useCallback } from 'react'
import { Pencil, Save, X, Plus, Trash2 } from 'lucide-react'
import { useUIStore } from '../../stores/uiStore'
import * as api from '../../api/database'
import { ConfirmDialog } from '../common/ConfirmDialog'
import { Modal } from '../common/Modal'
import type { Column, Index, ForeignKey, Tab } from '../../types'

interface Props { tab: Tab }

export function SchemaViewer({ tab }: Props) {
  const [columns, setColumns] = useState<Column[]>([])
  const [indexes, setIndexes] = useState<Index[]>([])
  const [fks, setFks] = useState<ForeignKey[]>([])
  const [ddl, setDdl] = useState('')
  const [loading, setLoading] = useState(true)
  const [editCol, setEditCol] = useState<string | null>(null)
  const [editComment, setEditComment] = useState('')
  const [showAddCol, setShowAddCol] = useState(false)
  const [showEditCol, setShowEditCol] = useState<Column | null>(null)
  const [confirmDropCol, setConfirmDropCol] = useState<Column | null>(null)
  const addToast = useUIStore((s) => s.addToast)

  const load = useCallback(async () => {
    if (!tab.connId || !tab.database || !tab.table) return
    setLoading(true)
    try {
      const [schema, ddlRes] = await Promise.all([
        api.getSchema(tab.connId, tab.database, tab.table),
        api.getDDL(tab.connId, tab.database, tab.table).catch(() => ({ ddl: '' })),
      ])
      setColumns(schema.columns)
      setIndexes(schema.indexes)
      setFks(schema.foreignKeys)
      setDdl(ddlRes.ddl)
    } catch { /* ignore */ }
    setLoading(false)
  }, [tab.connId, tab.database, tab.table])

  useEffect(() => { load() }, [load])

  const handleSaveComment = async (col: Column) => {
    if (!tab.connId || !tab.database || !tab.table) return
    try {
      await api.alterColumn(tab.connId, tab.database, tab.table, { oldName: col.name, comment: editComment })
      addToast('注释已更新', 'success')
      setEditCol(null)
      load()
    } catch (e) { addToast(`保存失败: ${(e as Error).message}`, 'error') }
  }

  const handleDropColumn = async () => {
    if (!confirmDropCol || !tab.connId || !tab.database || !tab.table) return
    try {
      await api.dropColumn(tab.connId, tab.database, tab.table, confirmDropCol.name)
      addToast(`列 ${confirmDropCol.name} 已删除`, 'success')
      setConfirmDropCol(null)
      load()
    } catch (e) { addToast(`删除失败: ${(e as Error).message}`, 'error') }
  }

  if (loading) return <div className="p-8 text-center text-sm" style={{ color: 'var(--text-muted)' }}>加载中...</div>

  return (
    <div className="p-4 overflow-auto h-full">
      <div className="flex items-center justify-between mb-3">
        <h3 className="text-sm font-semibold" style={{ color: 'var(--accent)' }}>{tab.table} — 表结构</h3>
        <button onClick={() => setShowAddCol(true)}
          className="flex items-center gap-1 px-2 py-1 rounded text-xs"
          style={{ backgroundColor: 'var(--accent)', color: '#fff' }}>
          <Plus size={12} /> 新增列
        </button>
      </div>

      {/* 列信息 */}
      <div className="mb-4">
        <h4 className="text-xs font-medium mb-2" style={{ color: 'var(--text-secondary)' }}>列</h4>
        <table className="w-full text-xs border-collapse" style={{ border: '1px solid var(--border-color)' }}>
          <thead>
            <tr style={{ backgroundColor: 'var(--bg-tertiary)' }}>
              <th className="px-2 py-1.5 text-left">列名</th>
              <th className="px-2 py-1.5 text-left">类型</th>
              <th className="px-2 py-1.5 text-center">可空</th>
              <th className="px-2 py-1.5 text-left">默认值</th>
              <th className="px-2 py-1.5 text-left">注释</th>
              <th className="px-2 py-1.5 text-center">主键</th>
              <th className="px-2 py-1.5 w-16">操作</th>
            </tr>
          </thead>
          <tbody>
            {columns.map((col) => (
              <tr key={col.name} style={{ borderTop: '1px solid var(--border-color)' }} className="group hover:opacity-80">
                <td className="px-2 py-1" style={{ color: 'var(--text-primary)', fontFamily: 'monospace' }}>{col.name}</td>
                <td className="px-2 py-1" style={{ color: 'var(--text-secondary)' }}>{col.dataType}</td>
                <td className="px-2 py-1 text-center" style={{ color: col.nullable ? 'var(--text-muted)' : 'var(--warning)' }}>{col.nullable ? 'YES' : 'NO'}</td>
                <td className="px-2 py-1 text-xs" style={{ color: 'var(--text-muted)' }}>{col.defaultVal || '-'}</td>
                <td className="px-2 py-1">
                  {editCol === col.name ? (
                    <div className="flex items-center gap-1">
                      <input autoFocus value={editComment} onChange={(e) => setEditComment(e.target.value)}
                        className="w-32 px-1 py-0.5 rounded text-xs outline-none"
                        style={{ backgroundColor: 'var(--bg-tertiary)', color: 'var(--text-primary)', border: '1px solid var(--accent)' }}
                        onKeyDown={(e) => { if (e.key === 'Enter') handleSaveComment(col); if (e.key === 'Escape') setEditCol(null) }} />
                      <button onClick={() => handleSaveComment(col)} className="p-0.5 rounded hover:opacity-70" style={{ color: 'var(--success)' }}><Save size={12} /></button>
                      <button onClick={() => setEditCol(null)} className="p-0.5 rounded hover:opacity-70" style={{ color: 'var(--danger)' }}><X size={12} /></button>
                    </div>
                  ) : (
                    <div className="flex items-center gap-1 group cursor-pointer"
                      onClick={() => { setEditCol(col.name); setEditComment(col.comment || '') }}>
                      <span style={{ color: col.comment ? 'var(--text-secondary)' : 'var(--text-muted)' }}>{col.comment || '点击添加注释'}</span>
                      <Pencil size={10} className="opacity-0 group-hover:opacity-50" />
                    </div>
                  )}
                </td>
                <td className="px-2 py-1 text-center">{col.primaryKey && <span style={{ color: 'var(--warning)' }}>🔑</span>}</td>
                <td className="px-2 py-1">
                  <div className="flex gap-0.5 opacity-0 group-hover:opacity-100">
                    <button onClick={() => setShowEditCol(col)} className="p-0.5 rounded hover:opacity-70" style={{ color: 'var(--text-muted)' }}><Pencil size={11} /></button>
                    {!col.primaryKey && (
                      <button onClick={() => setConfirmDropCol(col)} className="p-0.5 rounded hover:bg-red-500/20" style={{ color: 'var(--danger)' }}><Trash2 size={11} /></button>
                    )}
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* 索引 */}
      {indexes.length > 0 && (
        <div className="mb-4">
          <h4 className="text-xs font-medium mb-2" style={{ color: 'var(--text-secondary)' }}>索引</h4>
          <table className="w-full text-xs border-collapse" style={{ border: '1px solid var(--border-color)' }}>
            <thead><tr style={{ backgroundColor: 'var(--bg-tertiary)' }}><th className="px-2 py-1.5 text-left">索引名</th><th className="px-2 py-1.5 text-left">列</th><th className="px-2 py-1.5 text-center">唯一</th><th className="px-2 py-1.5 text-left">类型</th></tr></thead>
            <tbody>{indexes.map((idx) => (<tr key={idx.name} style={{ borderTop: '1px solid var(--border-color)' }}><td className="px-2 py-1" style={{ color: 'var(--text-primary)', fontFamily: 'monospace' }}>{idx.name}</td><td className="px-2 py-1" style={{ color: 'var(--text-secondary)' }}>{idx.columns.join(', ')}</td><td className="px-2 py-1 text-center">{idx.unique ? '✅' : ''}</td><td className="px-2 py-1" style={{ color: 'var(--text-muted)' }}>{idx.type}</td></tr>))}</tbody>
          </table>
        </div>
      )}

      {/* 外键 */}
      {fks.length > 0 && (
        <div className="mb-4">
          <h4 className="text-xs font-medium mb-2" style={{ color: 'var(--text-secondary)' }}>外键</h4>
          <table className="w-full text-xs border-collapse" style={{ border: '1px solid var(--border-color)' }}>
            <thead><tr style={{ backgroundColor: 'var(--bg-tertiary)' }}><th className="px-2 py-1.5 text-left">约束名</th><th className="px-2 py-1.5 text-left">列</th><th className="px-2 py-1.5 text-left">引用表</th><th className="px-2 py-1.5 text-left">引用列</th></tr></thead>
            <tbody>{fks.map((fk) => (<tr key={fk.name} style={{ borderTop: '1px solid var(--border-color)' }}><td className="px-2 py-1" style={{ color: 'var(--text-primary)', fontFamily: 'monospace' }}>{fk.name}</td><td className="px-2 py-1" style={{ color: 'var(--text-secondary)' }}>{fk.column}</td><td className="px-2 py-1" style={{ color: 'var(--accent)' }}>{fk.refTable}</td><td className="px-2 py-1" style={{ color: 'var(--text-secondary)' }}>{fk.refColumn}</td></tr>))}</tbody>
          </table>
        </div>
      )}

      {/* DDL */}
      {ddl && (
        <div>
          <h4 className="text-xs font-medium mb-2" style={{ color: 'var(--text-secondary)' }}>DDL</h4>
          <pre className="p-3 rounded text-xs overflow-auto" style={{ backgroundColor: 'var(--bg-tertiary)', color: 'var(--text-primary)', fontFamily: 'monospace', maxHeight: '200px' }}>{ddl}</pre>
        </div>
      )}

      {/* 新增列弹窗 */}
      {showAddCol && <ColumnForm modalTitle="新增列" tab={tab} onClose={() => setShowAddCol(false)} onSaved={load} />}

      {/* 编辑列弹窗 */}
      {showEditCol && (
        <ColumnForm modalTitle={`编辑列: ${showEditCol.name}`} tab={tab} initial={showEditCol}
          onClose={() => setShowEditCol(null)} onSaved={load} />
      )}

      {/* 删除列确认 */}
      <ConfirmDialog open={confirmDropCol !== null} title="确认删除列"
        message={`确定要删除列 "${confirmDropCol?.name}" 吗？此操作不可撤销，列中的数据将永久丢失。`}
        confirmText="删除列" danger
        onConfirm={handleDropColumn} onCancel={() => setConfirmDropCol(null)} />
    </div>
  )
}

function ColumnForm({ modalTitle, tab, initial, onClose, onSaved }: {
  modalTitle: string
  tab: Tab
  initial?: Column
  onClose: () => void
  onSaved: () => void
}) {
  const isEdit = !!initial
  const [name, setName] = useState(initial?.name || '')
  const [dataType, setDataType] = useState(initial?.dataType || 'VARCHAR(255)')
  const [nullable, setNullable] = useState(initial?.nullable ?? true)
  const [defaultVal, setDefaultVal] = useState(initial?.defaultVal || '')
  const [comment, setComment] = useState(initial?.comment || '')
  const [saving, setSaving] = useState(false)
  const addToast = useUIStore((s) => s.addToast)

  const handleSave = async () => {
    if (!tab.connId || !tab.database || !tab.table || !name.trim() || !dataType.trim()) return
    setSaving(true)
    try {
      if (isEdit && initial) {
        await api.alterColumn(tab.connId, tab.database, tab.table, {
          oldName: initial.name,
          newName: name !== initial.name ? name : undefined,
          newType: dataType !== initial.dataType ? dataType : undefined,
          comment: comment !== (initial.comment || '') ? comment : undefined,
          default: defaultVal !== (initial.defaultVal || '') ? defaultVal : undefined,
          nullable: nullable !== initial.nullable ? nullable : undefined,
        } as any)
        addToast('列已更新', 'success')
      } else {
        await api.addColumn(tab.connId, tab.database, tab.table, {
          name, type: dataType, nullable,
          default: defaultVal || undefined,
          comment: comment || undefined,
        } as any)
        addToast('列已新增', 'success')
      }
      onSaved()
      onClose()
    } catch (e) { addToast(`操作失败: ${(e as Error).message}`, 'error') }
    setSaving(false)
  }

  return (
    <Modal open onClose={onClose} title={modalTitle} width="420px">
      <div className="space-y-3 text-sm">
        {!isEdit || true ? (
          <div>
            <label className="text-xs" style={{ color: 'var(--text-muted)' }}>列名 *</label>
            <input value={name} onChange={(e) => setName(e.target.value)}
              className="w-full px-2 py-1.5 rounded mt-1 text-xs outline-none"
              style={{ backgroundColor: 'var(--bg-tertiary)', color: 'var(--text-primary)', border: '1px solid var(--border-color)' }} />
          </div>
        ) : null}
        <div>
          <label className="text-xs" style={{ color: 'var(--text-muted)' }}>数据类型 *</label>
          <input value={dataType} onChange={(e) => setDataType(e.target.value)}
            className="w-full px-2 py-1.5 rounded mt-1 text-xs outline-none"
            style={{ backgroundColor: 'var(--bg-tertiary)', color: 'var(--text-primary)', border: '1px solid var(--border-color)' }}
            placeholder="VARCHAR(255), INT, TEXT..." />
        </div>
        <div className="flex gap-4">
          <label className="flex items-center gap-1 text-xs cursor-pointer" style={{ color: 'var(--text-secondary)' }}>
            <input type="checkbox" checked={nullable} onChange={(e) => setNullable(e.target.checked)} /> 可为空
          </label>
        </div>
        <div>
          <label className="text-xs" style={{ color: 'var(--text-muted)' }}>默认值</label>
          <input value={defaultVal} onChange={(e) => setDefaultVal(e.target.value)}
            className="w-full px-2 py-1.5 rounded mt-1 text-xs outline-none"
            style={{ backgroundColor: 'var(--bg-tertiary)', color: 'var(--text-primary)', border: '1px solid var(--border-color)' }} />
        </div>
        <div>
          <label className="text-xs" style={{ color: 'var(--text-muted)' }}>注释</label>
          <input value={comment} onChange={(e) => setComment(e.target.value)}
            className="w-full px-2 py-1.5 rounded mt-1 text-xs outline-none"
            style={{ backgroundColor: 'var(--bg-tertiary)', color: 'var(--text-primary)', border: '1px solid var(--border-color)' }} />
        </div>
        <div className="flex gap-2 justify-end">
          <button onClick={onClose} className="px-3 py-1.5 rounded text-xs" style={{ backgroundColor: 'var(--bg-tertiary)', color: 'var(--text-secondary)' }}>取消</button>
          <button onClick={handleSave} disabled={saving}
            className="px-3 py-1.5 rounded text-xs" style={{ backgroundColor: 'var(--accent)', color: '#fff' }}>
            {saving ? '保存中...' : isEdit ? '保存修改' : '新增列'}
          </button>
        </div>
      </div>
    </Modal>
  )
}
