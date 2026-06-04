import { create } from 'zustand'
import type { Tab } from '../types'

interface TabState {
  tabs: Tab[]
  activeTabId: string
  addTab: (tab: Omit<Tab, 'id'>) => string
  closeTab: (id: string) => void
  setActive: (id: string) => void
  setTabSql: (id: string, sql: string) => void
}

let tabCounter = 0

export const useTabStore = create<TabState>((set) => ({
  tabs: [{ id: 'welcome', type: 'welcome', title: '欢迎' }],
  activeTabId: 'welcome',
  addTab: (tab) => {
    const id = 'tab_' + (++tabCounter)
    const newTab: Tab = { ...tab, id }
    set((s) => ({ tabs: [...s.tabs, newTab], activeTabId: id }))
    return id
  },
  closeTab: (id) => set((s) => {
    const idx = s.tabs.findIndex((t) => t.id === id)
    const tabs = s.tabs.filter((t) => t.id !== id)
    let activeTabId = s.activeTabId
    if (activeTabId === id) {
      activeTabId = tabs[Math.min(idx, tabs.length - 1)]?.id || 'welcome'
    }
    return { tabs, activeTabId }
  }),
  setActive: (id) => set({ activeTabId: id }),
  setTabSql: (id, sql) => set((s) => ({
    tabs: s.tabs.map((t) => (t.id === id ? { ...t, sql } : t)),
  })),
}))
