import { useState, useRef } from 'react'
import { Upload } from 'lucide-react'
import { Modal } from '../common/Modal'
import { Spinner } from '../common/Spinner'
import { useUIStore } from '../../stores/uiStore'
import type { Tab } from '../../types'

interface Props {
  open: boolean
  onClose: () => void
  tab: Tab
}

export function ImportDialog({ open, onClose, tab }: Props) {
  const [file, setFile] = useState<File | null>(null)
  const [importing, setImporting] = useState(false)
  const [result, setResult] = useState<{ inserted: number; failed: number } | null>(null)
  const addToast = useUIStore((s) => s.addToast)
  const fileInputRef = useRef<HTMLInputElement>(null)

  const handleImport = async () => {
    if (!file || !tab.connId || !tab.database || !tab.table) return
    setImporting(true)
    try {
      const formData = new FormData()
      formData.append('file', file)
      formData.append('connId', tab.connId)
      formData.append('database', tab.database)
      formData.append('table', tab.table)

      const res = await fetch('/api/import/csv', { method: 'POST', body: formData })
      const data = await res.json()
      if (!res.ok) throw new Error(data.error || 'import failed')
      setResult(data)
      addToast(`导入完成: ${data.inserted} 成功, ${data.failed} 失败`, 'success')
    } catch (e) {
      addToast(`导入失败: ${(e as Error).message}`, 'error')
    } finally {
      setImporting(false)
    }
  }

  return (
    <Modal open={open} onClose={onClose} title="CSV 导入" width="420px">
      <div className="space-y-3 text-sm">
        <p className="text-xs" style={{ color: 'var(--text-muted)' }}>
          目标表: <b>{tab.table}</b> ({tab.database})
        </p>

        <div className="text-center p-6 rounded border-2 border-dashed cursor-pointer"
          style={{ borderColor: 'var(--border-color)', backgroundColor: 'var(--bg-secondary)' }}
          onClick={() => fileInputRef.current?.click()}>
          <Upload size={24} className="mx-auto mb-2" style={{ color: 'var(--text-muted)' }} />
          <p style={{ color: 'var(--text-secondary)' }}>
            {file ? file.name : '点击选择 CSV 文件'}
          </p>
          <p className="text-xs mt-1" style={{ color: 'var(--text-muted)' }}>
            首行为列名，与目标表列名自动匹配
          </p>
          <input ref={fileInputRef} type="file" accept=".csv" className="hidden"
            onChange={(e) => setFile(e.target.files?.[0] || null)} />
        </div>

        {result && (
          <div className="p-2 rounded text-xs" style={{ backgroundColor: 'var(--bg-tertiary)' }}>
            ✅ 成功: {result.inserted} 行<br />
            {result.failed > 0 && <span style={{ color: 'var(--danger)' }}>❌ 失败: {result.failed} 行</span>}
          </div>
        )}

        <div className="flex gap-2 justify-end">
          <button onClick={onClose}
            className="px-3 py-1.5 rounded text-xs" style={{ backgroundColor: 'var(--bg-tertiary)', color: 'var(--text-secondary)' }}>取消</button>
          <button onClick={handleImport} disabled={!file || importing}
            className="px-3 py-1.5 rounded text-xs flex items-center gap-1"
            style={{ backgroundColor: file ? 'var(--accent)' : 'var(--bg-tertiary)', color: file ? '#fff' : 'var(--text-muted)' }}>
            {importing ? <Spinner size={14} /> : null}
            开始导入
          </button>
        </div>
      </div>
    </Modal>
  )
}
