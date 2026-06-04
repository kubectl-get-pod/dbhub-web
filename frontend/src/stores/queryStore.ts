import { create } from 'zustand'
import type { QueryResult, QueryHistoryItem, QueryFavorite } from '../types'

interface QueryState {
  results: Record<string, QueryResult | null>
  history: QueryHistoryItem[]
  favorites: QueryFavorite[]
  running: Record<string, boolean>
  setResult: (tabId: string, result: QueryResult | null) => void
  setHistory: (items: QueryHistoryItem[]) => void
  addHistory: (item: QueryHistoryItem) => void
  setFavorites: (items: QueryFavorite[]) => void
  addFavorite: (item: QueryFavorite) => void
  removeFavorite: (id: string) => void
  setRunning: (tabId: string, v: boolean) => void
}

export const useQueryStore = create<QueryState>((set) => ({
  results: {},
  history: [],
  favorites: [],
  running: {},
  setResult: (tabId, result) => set((s) => ({ results: { ...s.results, [tabId]: result } })),
  setHistory: (history) => set({ history }),
  addHistory: (item) => set((s) => ({ history: [item, ...s.history].slice(0, 100) })),
  setFavorites: (favorites) => set({ favorites }),
  addFavorite: (item) => set((s) => ({ favorites: [...s.favorites, item] })),
  removeFavorite: (id) => set((s) => ({ favorites: s.favorites.filter((f) => f.id !== id) })),
  setRunning: (tabId, v) => set((s) => ({ running: { ...s.running, [tabId]: v } })),
}))
