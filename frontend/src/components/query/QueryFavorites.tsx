import { useState, useEffect } from 'react'
import { Star, Trash2, Play } from 'lucide-react'
import { useQueryStore } from '../../stores/queryStore'
import { useTabStore } from '../../stores/tabStore'
import { useUIStore } from '../../stores/uiStore'
import * as api from '../../api/database'

export function QueryFavorites() {
  const favorites = useQueryStore((s) => s.favorites)
  const setFavorites = useQueryStore((s) => s.setFavorites)
  const addTab = useTabStore((s) => s.addTab)
  const addToast = useUIStore((s) => s.addToast)

  useEffect(() => {
    api.listFavorites().then(setFavorites).catch(() => {})
  }, [])

  const handleLoad = (name: string, sql: string) => {
    addTab({ type: 'query', title: name, sql })
    addToast(`已加载: ${name}`, 'info')
  }

  const handleDelete = async (id: string) => {
    try {
      await api.deleteFavorite(id)
      useQueryStore.getState().removeFavorite(id)
      addToast('已删除', 'info')
    } catch { /* ignore */ }
  }

  if (favorites.length === 0) return null

  return (
    <div className="p-3" style={{ borderBottom: '1px solid var(--border-color)' }}>
      <div className="text-xs font-medium mb-2" style={{ color: 'var(--text-secondary)' }}>
        ⭐ 收藏查询 ({favorites.length})
      </div>
      {favorites.slice(0, 5).map((fav) => (
        <div key={fav.id} className="group flex items-center gap-1 py-0.5 text-xs"
          style={{ color: 'var(--text-primary)' }}>
          <Star size={10} style={{ color: 'var(--warning)' }} />
          <span className="flex-1 truncate cursor-pointer hover:underline"
            onClick={() => handleLoad(fav.name, fav.sql)}>{fav.name}</span>
          <button onClick={() => handleDelete(fav.id)}
            className="opacity-0 group-hover:opacity-50 p-0.5 rounded hover:bg-red-500/20">
            <Trash2 size={10} />
          </button>
        </div>
      ))}
    </div>
  )
}
