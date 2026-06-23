import { useEffect, useRef, useCallback, useState } from 'react'
import { EditorState } from '@codemirror/state'
import { EditorView, keymap, lineNumbers, highlightActiveLine } from '@codemirror/view'
import { defaultKeymap, history, historyKeymap } from '@codemirror/commands'
import { sql, MySQL, PostgreSQL, StandardSQL } from '@codemirror/lang-sql'
import { autocompletion, CompletionContext, CompletionResult, acceptCompletion } from '@codemirror/autocomplete'
import { oneDark } from '@codemirror/theme-one-dark'
import { format } from 'sql-formatter'
import { Play, Zap, FileText, Star, ChevronDown, Clock } from 'lucide-react'
import { useQueryStore } from '../../stores/queryStore'
import { useTabStore } from '../../stores/tabStore'
import { useUIStore } from '../../stores/uiStore'
import * as api from '../../api/database'
import type { AutoCompleteData, Tab } from '../../types'
import { QueryHistory } from './QueryHistory'

interface Props {
  tab: Tab
}

export function QueryEditor({ tab }: Props) {
  const editorRef = useRef<HTMLDivElement>(null)
  const viewRef = useRef<EditorView | null>(null)
  const setResult = useQueryStore((s) => s.setResult)
  const setRunning = useQueryStore((s) => s.setRunning)
  const setTabSql = useTabStore((s) => s.setTabSql)
  const addToast = useUIStore((s) => s.addToast)
  const addHistory = useQueryStore((s) => s.addHistory)
  const running = useQueryStore((s) => s.running[tab.id])

  // Build auto-complete data source
  const buildCompletionSource = useCallback(async (db: string) => {
    if (!tab.connId) return []
    try {
      const data = await api.getAutoComplete(tab.connId, db)
      return dataToCompletions(data)
    } catch {
      return []
    }
  }, [tab.connId])

  useEffect(() => {
    if (!editorRef.current) return

    const completionSource = async (ctx: CompletionContext): Promise<CompletionResult | null> => {
      const match = ctx.matchBefore(/\w*/)
      const data = await buildCompletionSource(tab.database || '')
      return { from: match ? match.from : ctx.pos, options: data }
    }

    const onExecuteAll = () => {
      const text = viewRef.current?.state.doc.toString() || ''
      if (text.trim()) executeSQL(text)
    }

    const onExecuteSelected = () => {
      const selection = viewRef.current?.state.selection.main
      if (!selection || selection.empty) {
        onExecuteAll()
        return
      }
      const text = viewRef.current?.state.doc.sliceString(selection.from, selection.to) || ''
      if (text.trim()) executeSQL(text)
    }

    const state = EditorState.create({
      doc: tab.sql || '',
      extensions: [
        lineNumbers(),
        highlightActiveLine(),
        history(),
        keymap.of([...defaultKeymap, ...historyKeymap]),
        oneDark,
        sql({ dialect: sqlDialect(tab.database) }),
        autocompletion({ override: [completionSource] }),
        EditorView.updateListener.of((update) => {
          if (update.docChanged && viewRef.current) {
            setTabSql(tab.id, viewRef.current.state.doc.toString())
          }
        }),
        keymap.of([
          { key: 'Ctrl-Enter', run: () => { onExecuteSelected(); return true } },
          { key: 'Tab', run: acceptCompletion },
        ]),
      ],
    })

    const view = new EditorView({
      state,
      parent: editorRef.current,
    })
    viewRef.current = view

    return () => view.destroy()
  }, [tab.id])

  const executeSQL = async (sqlText: string) => {
    if (!tab.connId || !sqlText.trim()) return
    setRunning(tab.id, true)
    try {
      const result = await api.executeQuery(tab.connId, sqlText, 0, 0, tab.database, tab.table)
      setResult(tab.id, result)
      addToast(`执行成功, ${result.rowCount} 行, ${result.duration}`, 'success')
    } catch (e) {
      addToast(`执行失败: ${(e as Error).message}`, 'error')
    } finally {
      setRunning(tab.id, false)
    }
  }

  const handleFormat = () => {
    if (!viewRef.current) return
    const text = viewRef.current.state.doc.toString()
    try {
      const formatted = format(text, { language: formatterDialect(tab.database) })
      viewRef.current.dispatch({
        changes: { from: 0, to: viewRef.current.state.doc.length, insert: formatted },
      })
    } catch {
      addToast('格式化失败：SQL 语法错误', 'error')
    }
  }

  const handleSaveFavorite = async () => {
    if (!viewRef.current) return
    const sql = viewRef.current.state.doc.toString().trim()
    if (!sql) return
    try {
      const name = prompt('收藏名称:', '我的查询')
      if (!name) return
      await api.createFavorite(name, sql, tab.database || '')
      addToast('已保存到收藏', 'success')
    } catch (e) {
      addToast(`保存失败: ${(e as Error).message}`, 'error')
    }
  }

  const handleExportCSV = async () => {
    if (!viewRef.current || !tab.connId) return
    const sql = viewRef.current.state.doc.toString().trim()
    if (!sql) return
    try {
      await api.exportCSV(tab.connId, sql, 'query_result.csv', tab.database, tab.table)
      addToast('CSV 导出成功', 'success')
    } catch {
      addToast('导出失败', 'error')
    }
  }

  const handleExportExcel = async () => {
    if (!viewRef.current || !tab.connId) return
    const sql = viewRef.current.state.doc.toString().trim()
    if (!sql) return
    try {
      await api.exportExcel(tab.connId, sql, 'query_result.xlsx', tab.database, tab.table)
      addToast('Excel 导出成功', 'success')
    } catch {
      addToast('导出失败', 'error')
    }
  }

  const [exportOpen, setExportOpen] = useState(false)
  const [showHistory, setShowHistory] = useState(false)

  const result = useQueryStore((s) => s.results[tab.id])

  return (
    <div className="flex flex-col h-full">
      {/* 工具栏 */}
      <div className="flex items-center gap-1 px-2 py-1" style={{ borderBottom: '1px solid var(--border-color)' }}>
        {tab.database && (
          <span className="px-2 py-0.5 rounded text-xs font-mono" style={{ backgroundColor: 'var(--bg-tertiary)', color: 'var(--accent)' }}>
            {tab.database}
          </span>
        )}
        <button onClick={() => executeSQL(viewRef.current?.state.doc.toString() || '')}
          disabled={running}
          className="flex items-center gap-1 px-2 py-1 rounded text-xs font-medium"
          style={{ backgroundColor: 'var(--accent)', color: '#fff' }}>
          <Play size={12} /> 执行 (Ctrl+Enter)
        </button>
        <button onClick={handleFormat}
          className="flex items-center gap-1 px-2 py-1 rounded text-xs"
          style={{ backgroundColor: 'var(--bg-tertiary)', color: 'var(--text-secondary)' }}>
          <Zap size={12} /> 格式化
        </button>
        <button onClick={handleSaveFavorite}
          className="flex items-center gap-1 px-2 py-1 rounded text-xs"
          style={{ backgroundColor: 'var(--bg-tertiary)', color: 'var(--text-secondary)' }}>
          <Star size={12} /> 收藏
        </button>
        <div className="relative">
          <button onClick={() => setExportOpen(!exportOpen)}
            className="flex items-center gap-1 px-2 py-1 rounded text-xs"
            style={{ backgroundColor: 'var(--bg-tertiary)', color: 'var(--text-secondary)' }}>
            <FileText size={12} /> 导出 <ChevronDown size={10} />
          </button>
          {exportOpen && (
            <div className="absolute top-full left-0 mt-1 rounded shadow-lg z-50 py-1 min-w-[120px]"
              style={{ backgroundColor: 'var(--bg-primary)', border: '1px solid var(--border-color)' }}>
              <button onClick={() => { handleExportCSV(); setExportOpen(false) }}
                className="w-full text-left px-3 py-1.5 text-xs hover:opacity-80"
                style={{ color: 'var(--text-primary)' }}>
                📄 CSV 导出
              </button>
              <button onClick={() => { handleExportExcel(); setExportOpen(false) }}
                className="w-full text-left px-3 py-1.5 text-xs hover:opacity-80"
                style={{ color: 'var(--text-primary)' }}>
                📊 Excel 导出 (.xlsx)
              </button>
            </div>
          )}
        </div>
        <div className="flex-1" />
        <button onClick={() => setShowHistory(!showHistory)}
          className="flex items-center gap-1 px-2 py-1 rounded text-xs"
          style={{ backgroundColor: showHistory ? 'var(--accent)' : 'var(--bg-tertiary)', color: showHistory ? '#fff' : 'var(--text-secondary)' }}>
          <Clock size={12} /> 历史
        </button>
      </div>

      {/* 编辑器 + 历史面板 */}
      <div className="flex-1 flex overflow-hidden" style={{ minHeight: '40%', maxHeight: result ? '50%' : '100%' }}>
        <div className="flex-1 overflow-hidden" ref={editorRef} />
        {showHistory && (
          <div className="w-[240px] flex-shrink-0 overflow-y-auto" style={{ borderLeft: '1px solid var(--border-color)', backgroundColor: 'var(--bg-secondary)' }}>
            <QueryHistory />
          </div>
        )}
      </div>

      {/* 结果区 */}
      {result && (
        <div className="border-t" style={{ borderColor: 'var(--border-color)', maxHeight: '50%', overflow: 'auto' }}>
          <div className="flex items-center px-3 py-1 text-xs" style={{ borderBottom: '1px solid var(--border-color)', backgroundColor: 'var(--bg-secondary)' }}>
            <span style={{ color: 'var(--text-secondary)' }}>
              {result.rowCount} 行 | {result.duration}
            </span>
          </div>
          <table className="w-full text-xs border-collapse">
            <thead>
              <tr style={{ backgroundColor: 'var(--bg-tertiary)' }}>
                {result.columns.map((col) => (
                  <th key={col} className="px-2 py-1 text-left" style={{ color: 'var(--text-secondary)' }}>{col}</th>
                ))}
              </tr>
              {result.comments && Object.values(result.comments).some(c => c) && (
                <tr style={{ backgroundColor: 'var(--bg-tertiary)' }}>
                  {result.columns.map((col) => (
                    <th key={col} className="px-2 py-0.5 text-left font-normal italic" style={{ color: 'var(--text-muted)', fontSize: '10px' }}>
                      {result.comments![col] || ''}
                    </th>
                  ))}
                </tr>
              )}
            </thead>
            <tbody>
              {result.rows.slice(0, 200).map((row, ri) => (
                <tr key={ri} style={{ borderTop: '1px solid var(--border-color)' }}>
                  {row.map((val, ci) => (
                    <td key={ci} className="px-2 py-0.5 max-w-xs truncate" style={{ color: 'var(--text-primary)' }}>
                      {val === null ? <i style={{ color: 'var(--text-muted)' }}>NULL</i> : String(val)}
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}

function sqlDialect(database?: string) {
  switch (database) {
    case 'postgres': return PostgreSQL
    case 'mysql': return MySQL
    default: return StandardSQL
  }
}

function formatterDialect(database?: string): 'mysql' | 'postgresql' | 'sql' {
  switch (database) {
    case 'postgres': return 'postgresql'
    case 'mysql': return 'mysql'
    default: return 'sql'
  }
}

function dataToCompletions(data: AutoCompleteData) {
  const items = [
    ...data.keywords.map((k) => ({ label: k, type: 'keyword' as const })),
    ...data.functions.map((f) => ({ label: f, type: 'function' as const })),
    ...data.tables.map((t) => ({ label: t.name, type: 'type' as const, detail: 'TABLE' })),
    ...data.columns.map((c) => ({
      label: c.column,
      type: 'property' as const,
      detail: `${c.type} (${c.table})`,
    })),
  ]
  return items
}
