import { Modal } from './Modal'

interface Props {
  open: boolean
  title: string
  message: string
  confirmText?: string
  cancelText?: string
  danger?: boolean
  onConfirm: () => void
  onCancel: () => void
}

export function ConfirmDialog({
  open, title, message, confirmText = '确认', cancelText = '取消',
  danger = false, onConfirm, onCancel,
}: Props) {
  if (!open) return null

  return (
    <Modal open={open} onClose={onCancel} title={title} width="380px">
      <div className="text-sm" style={{ color: 'var(--text-secondary)' }}>
        <p className="mb-4 whitespace-pre-wrap">{message}</p>
        <div className="flex gap-2 justify-end">
          <button onClick={onCancel}
            className="px-3 py-1.5 rounded text-xs"
            style={{ backgroundColor: 'var(--bg-tertiary)', color: 'var(--text-secondary)' }}>
            {cancelText}
          </button>
          <button onClick={onConfirm}
            className="px-3 py-1.5 rounded text-xs font-medium"
            style={{ backgroundColor: danger ? 'var(--danger)' : 'var(--accent)', color: '#fff' }}>
            {confirmText}
          </button>
        </div>
      </div>
    </Modal>
  )
}
