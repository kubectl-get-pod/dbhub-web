import { create } from 'zustand'

type Theme = 'dark' | 'light'

interface Toast {
  id: string
  message: string
  type: 'success' | 'error' | 'info'
}

interface UIState {
  theme: Theme
  sidebarWidth: number
  toasts: Toast[]
  toggleTheme: () => void
  addToast: (message: string, type?: Toast['type']) => void
  removeToast: (id: string) => void
}

export const useUIStore = create<UIState>((set) => ({
  theme: (localStorage.getItem('dbhub-theme') as Theme) || 'dark',
  sidebarWidth: 280,
  toasts: [],
  toggleTheme: () => set((s) => {
    const next = s.theme === 'dark' ? 'light' : 'dark'
    localStorage.setItem('dbhub-theme', next)
    return { theme: next }
  }),
  addToast: (message, type = 'info') => set((s) => ({
    toasts: [...s.toasts, { id: Date.now().toString(), message, type }],
  })),
  removeToast: (id) => set((s) => ({
    toasts: s.toasts.filter((t) => t.id !== id),
  })),
}))
