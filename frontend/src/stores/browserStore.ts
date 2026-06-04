import { create } from 'zustand'

interface BrowserState {
  expandedNodes: Set<string>
  toggleNode: (key: string) => void
  collapseAll: () => void
}

export const useBrowserStore = create<BrowserState>((set) => ({
  expandedNodes: new Set(),
  toggleNode: (key) => set((s) => {
    const next = new Set(s.expandedNodes)
    if (next.has(key)) next.delete(key)
    else next.add(key)
    return { expandedNodes: next }
  }),
  collapseAll: () => set({ expandedNodes: new Set() }),
}))
