import { useState, useEffect, useCallback } from 'react'
import { ChevronLeft, ChevronRight, Plus, Trash2, Check, X } from 'lucide-react'
import { useUIStore } from '../../stores/uiStore'
import * as api from '../../api/database'
import type { Tab, QueryResult, Column } from '../../types'

interface Props {
  tab: Tab
}

const PAGE_SIZE = 50

export function DataGrid({ tab }: Props) {
  const [data, setData] = useState<QueryResult | null>(null)
  const [columns, setColumns] = useState<Column[]>([])
  const [page, setPage] = useState(0)
  const [rowCount, setRowCount] = useState(0)
  const [loading, setLoading] = useState(true)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [editingRow, setEditingRow] = useState<number | null>(null)
  const [editingValues, setEditingValues] = useState<Record<string, string>>({})
  const [showInsert, setShowInsert] = useState(false)
  const [insertValues, setInsertValues] = useState<Record<string, string>>({})
  const addToast = useUIStore((s) => s.addToast)

  const totalPages = Math.ceil(rowCount / PAGE_SIZE)

  const load = useCallback(async () => {
    if (!tab.connId || !tab.database || !tab.table) return
    setLoading(true)
    setLoadError(null)
    try {
      const [result, count, schema] = await Promise.all([
        api.listData(tab.connId, tab.database, tab.table, PAGE_SIZE, page * PAGE_SIZE),
        api.getRowCount(tab.connId, tab.database, tab.table),
        api.getSchema(tab.connId, tab.database, tab.table),
      ])
      setData(result)
      setRowCount(count.count)
      setColumns(schema.columns)
      setLoadError(null)
    } catch (e) {
      setLoadError((e as Error).message || '加载失败')
    }
    setLoading(false)
  }, [tab.connId, tab.database, tab.table, page])

  useEffect(() => { load() }, [load])

  const handleEdit = (rowIdx: number) => {
    if (!data) return
    const vals: Record<string, string> = {}
    data.columns.forEach((col, ci) => {
      vals[col] = String(data.rows[rowIdx][ci] ?? '')
    })
    setEditingValues(vals)
    setEditingRow(rowIdx)
  }

  const handleSaveEdit = async () => {
    if (!tab.connId || !tab.database || !tab.table || editingRow === null || !data) return
    const pkCols = columns.filter((c) => c.primaryKey).map((c) => c.name)
    const pk: Record<string, unknown> = {}
    if (pkCols.length > 0) {
      pkCols.forEach((c) => {
        const ci = data.columns.indexOf(c)
        pk[c] = data.rows[editingRow][ci]
      })
    }
    const values: Record<string, unknown> = {}
    Object.entries(editingValues).forEach(([k, v]) => {
      if (String(data.rows[editingRow][data.columns.indexOf(k)] ?? '') !== v) {
        values[k] = v
      }
    })
    if (Object.keys(values).length === 0) { setEditingRow(null); return }
    try {
      await api.updateRow(tab.connId, tab.database, tab.table, pk, values)
      addToast('更新成功', 'success')
      setEditingRow(null)
      load()
    } catch (e) {
      addToast(`更新失败: ${(e as Error).message}`, 'error')
    }
  }

  const handleDelete = async (rowIdx: number) => {
    if (!tab.connId || !tab.database || !tab.table || !data) return
    const pkCols = columns.filter((c) => c.primaryKey).map((c) => c.name)
    if (pkCols.length === 0) {
      addToast('该表无主键，无法删除行', 'error')
      return
    }
    const pk: Record<string, unknown> = {}
    pkCols.forEach((c) => {
      const ci = data.columns.indexOf(c)
      pk[c] = data.rows[rowIdx][ci]
    })
    try {
      await api.deleteRow(tab.connId, tab.database, tab.table, pk)
      addToast('删除成功', 'info')
      load()
    } catch (e) {
      addToast(`删除失败: ${(e as Error).message}`, 'error')
    }
  }

  const handleInsert = async () => {
    if (!tab.connId || !tab.database || !tab.table) return
    try {
      await api.insertRow(tab.connId, tab.database, tab.table, insertValues)
      addToast('插入成功', 'success')
      setShowInsert(false)
      setInsertValues({})
      load()
    } catch (e) {
      addToast(`插入失败: ${(e as Error).message}`, 'error')
    }
  }

  const handleCopyCell = (val: unknown) => {
    navigator.clipboard.writeText(String(val ?? ''))
    addToast('已复制', 'info')
  }

  if (loading) return <div className="p-8 text-center text-sm" style={{ color: 'var(--text-muted)' }}>加载中...</div>
  if (loadError) return (
    <div className="p-8 text-center">
      <p className="text-sm mb-2" style={{ color: 'var(--danger)' }}>加载失败</p>
      <p className="text-xs mb-3" style={{ color: 'var(--text-muted)' }}>{loadError}</p>
      <button onClick={() => load()} className="px-3 py-1 rounded text-xs" style={{ backgroundColor: 'var(--accent)', color: '#fff' }}>重试</button>
    </div>
  )
  if (!data || data.rows.length === 0) return (
    <div className="p-8 text-center text-sm" style={{ color: 'var(--text-muted)' }}>
      {rowCount === 0 ? '该表无数据' : '暂无数据'}
    </div>
  )

  return (
    <div className="flex flex-col h-full">
      {/* 工具栏 */}
      <div className="flex items-center gap-2 px-3 py-1.5 text-xs" style={{ borderBottom: '1px solid var(--border-color)' }}>
        <button onClick={() => setShowInsert(true)}
          className="flex items-center gap-1 px-2 py-1 rounded hover:opacity-80"
          style={{ backgroundColor: 'var(--accent)', color: '#fff' }}>
          <Plus size={12} /> 插入
        </button>
        <span className="ml-auto" style={{ color: 'var(--text-muted)' }}>
          共 {rowCount.toLocaleString()} 行
        </span>
      </div>

      {/* 数据表格 */}
      <div className="flex-1 overflow-auto">
        <table className="w-full text-xs border-collapse">
          <thead>
            <tr style={{ backgroundColor: 'var(--bg-tertiary)', position: 'sticky', top: 0 }}>
              <th className="px-2 py-1.5 text-left w-10">#</th>
              {data?.columns.map((col) => (
                <th key={col} className="px-2 py-1.5 text-left whitespace-nowrap" style={{ color: 'var(--text-secondary)' }}>
                  {col}
                </th>
              ))}
              <th className="px-2 py-1.5 w-16">操作</th>
            </tr>
          </thead>
          <tbody>
            {data?.rows.map((row, ri) => (
              <tr key={ri} style={{ borderTop: '1px solid var(--border-color)' }}
                onDoubleClick={() => handleEdit(ri)}
                className="hover:opacity-80">
                <td className="px-2 py-0.5" style={{ color: 'var(--text-muted)' }}>{page * PAGE_SIZE + ri + 1}</td>
                {row.map((val, ci) => (
                  <td key={ci} className="px-2 py-0.5 max-w-xs truncate cursor-pointer"
                    style={{ color: 'var(--text-primary)' }}
                    onClick={() => handleCopyCell(val)}
                    title="点击复制">
                    {editingRow === ri ? (
                      <input
                        value={editingValues[data.columns[ci]] ?? ''}
                        onChange={(e) => setEditingValues({ ...editingValues, [data.columns[ci]]: e.target.value })}
                        className="w-full px-1 py-0 rounded text-xs outline-none"
                        style={{ backgroundColor: 'var(--bg-tertiary)', color: 'var(--text-primary)', border: '1px solid var(--accent)' }}
                      />
                    ) : (
                      <span>{val === null ? <i style={{ color: 'var(--text-muted)' }}>NULL</i> : String(val)}</span>
                    )}
                  </td>
                ))}
                <td className="px-2 py-0.5">
                  <div className="flex items-center gap-1">
                    {editingRow === ri ? (
                      <>
                        <button onClick={handleSaveEdit} className="p-0.5 rounded hover:opacity-70" style={{ color: 'var(--success)' }}>
                          <Check size={12} />
                        </button>
                        <button onClick={() => setEditingRow(null)} className="p-0.5 rounded hover:opacity-70" style={{ color: 'var(--danger)' }}>
                          <X size={12} />
                        </button>
                      </>
                    ) : (
                      <button onClick={() => handleDelete(ri)} className="p-0.5 rounded hover:bg-red-500/20" style={{ color: 'var(--text-muted)' }}>
                        <Trash2 size={12} />
                      </button>
                    )}
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* 分页器 */}
      {totalPages > 1 && (
        <div className="flex items-center justify-center gap-2 py-2 text-xs border-t" style={{ borderColor: 'var(--border-color)' }}>
          <button onClick={() => setPage(Math.max(0, page - 1))} disabled={page === 0}
            className="p-1 rounded hover:opacity-70" style={{ color: page === 0 ? 'var(--text-muted)' : 'var(--text-primary)' }}>
            <ChevronLeft size={14} />
          </button>
          <span style={{ color: 'var(--text-secondary)' }}>{page + 1} / {totalPages}</span>
          <button onClick={() => setPage(Math.min(totalPages - 1, page + 1))} disabled={page >= totalPages - 1}
            className="p-1 rounded hover:opacity-70" style={{ color: page >= totalPages - 1 ? 'var(--text-muted)' : 'var(--text-primary)' }}>
            <ChevronRight size={14} />
          </button>
        </div>
      )}

      {/* 插入行弹窗 */}
      {showInsert && (
        <div className="fixed inset-0 z-50 flex items-center justify-center" style={{ backgroundColor: 'rgba(0,0,0,0.5)' }}
          onClick={() => setShowInsert(false)}>
          <div className="p-4 rounded w-96" style={{ backgroundColor: 'var(--bg-primary)', border: '1px solid var(--border-color)' }}
            onClick={(e) => e.stopPropagation()}>
            <h3 className="text-sm font-semibold mb-3" style={{ color: 'var(--accent)' }}>插入新行</h3>
            <div className="space-y-2 max-h-60 overflow-y-auto">
              {columns.map((col) => (
                <div key={col.name} className="flex items-center gap-2 text-xs">
                  <label className="w-24 truncate text-right" style={{ color: 'var(--text-secondary)' }}>{col.name}</label>
                  <input
                    value={insertValues[col.name] || ''}
                    onChange={(e) => setInsertValues({ ...insertValues, [col.name]: e.target.value })}
                    className="flex-1 px-2 py-1 rounded outline-none"
                    style={{ backgroundColor: 'var(--bg-tertiary)', color: 'var(--text-primary)', border: '1px solid var(--border-color)' }}
                    placeholder={col.nullable ? 'NULL' : '必填'}
                  />
                </div>
              ))}
            </div>
            <div className="flex gap-2 mt-3 justify-end">
              <button onClick={() => setShowInsert(false)}
                className="px-3 py-1 rounded text-xs" style={{ backgroundColor: 'var(--bg-tertiary)', color: 'var(--text-secondary)' }}>取消</button>
              <button onClick={handleInsert}
                className="px-3 py-1 rounded text-xs" style={{ backgroundColor: 'var(--accent)', color: '#fff' }}>插入</button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
