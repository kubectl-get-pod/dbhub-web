import { useEffect } from 'react'
import { useUIStore } from '../../stores/uiStore'

export function Toast() {
  const toasts = useUIStore((s) => s.toasts)
  const removeToast = useUIStore((s) => s.removeToast)

  return (
    <div className="fixed top-4 right-4 z-50 flex flex-col gap-2">
      {toasts.map((t) => (
        <ToastItem key={t.id} {...t} onRemove={() => removeToast(t.id)} />
      ))}
    </div>
  )
}

function ToastItem({ id, message, type, onRemove }: { id: string; message: string; type: string; onRemove: () => void }) {
  useEffect(() => {
    const timer = setTimeout(onRemove, 4000)
    return () => clearTimeout(timer)
  }, [id])

  const bg = type === 'error' ? 'var(--danger)' : type === 'success' ? 'var(--success)' : 'var(--accent)'
  return (
    <div className="px-4 py-2 rounded-lg text-sm text-white shadow-lg cursor-pointer animate-fade-in" style={{ backgroundColor: bg }} onClick={onRemove}>
      {message}
    </div>
  )
}
