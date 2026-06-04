const BASE = '/api'

async function request<T>(url: string, options?: RequestInit): Promise<T> {
  const res = await fetch(BASE + url, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  })
  const data = await res.json()
  if (!res.ok) {
    throw new Error((data as { error?: string; detail?: string }).error || res.statusText)
  }
  return data as T
}

export async function GET<T>(url: string): Promise<T> {
  return request<T>(url)
}

export async function POST<T>(url: string, body?: unknown): Promise<T> {
  return request<T>(url, { method: 'POST', body: JSON.stringify(body) })
}

export async function PUT<T>(url: string, body?: unknown): Promise<T> {
  return request<T>(url, { method: 'PUT', body: JSON.stringify(body) })
}

export async function DELETE<T>(url: string): Promise<T> {
  return request<T>(url, { method: 'DELETE' })
}

export async function uploadFile<T>(url: string, formData: FormData): Promise<T> {
  const res = await fetch(BASE + url, { method: 'POST', body: formData })
  const data = await res.json()
  if (!res.ok) throw new Error((data as { error: string }).error || res.statusText)
  return data as T
}

export function downloadBlob(url: string, body: unknown, filename: string) {
  return fetch(BASE + url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  }).then((res) => {
    if (!res.ok) throw new Error('download failed')
    return res.blob()
  }).then((blob) => {
    const a = document.createElement('a')
    a.href = URL.createObjectURL(blob)
    a.download = filename
    a.click()
    URL.revokeObjectURL(a.href)
  })
}
