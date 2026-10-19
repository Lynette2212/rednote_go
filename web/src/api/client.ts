const TOKEN_KEY = 'rednote_token'
const MEDIA_BASE = import.meta.env.VITE_API_BASE ?? ''

export function mediaUrl(path?: string): string {
  if (!path) return ''
  if (/^https?:\/\//.test(path)) return path
  // 演示视频与封面随 Web 一起发布
  if (path.startsWith('/videos/') || path.startsWith('/covers/')) return path
  return `${MEDIA_BASE}${path}`
}

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY)
}

export function setToken(token: string) {
  localStorage.setItem(TOKEN_KEY, token)
}

export function clearToken() {
  localStorage.removeItem(TOKEN_KEY)
}

export class ApiError extends Error {
  code: number
  constructor(code: number, message: string) {
    super(message)
    this.code = code
  }
}

async function request<T>(url: string, options: RequestInit = {}): Promise<T> {
  const headers: Record<string, string> = {
    ...(options.headers as Record<string, string>),
  }
  const token = getToken()
  if (token) headers['Authorization'] = `Bearer ${token}`
  if (options.body && !(options.body instanceof FormData)) {
    headers['Content-Type'] = 'application/json'
  }

  const resp = await fetch(url, { ...options, headers })
  const json = await resp.json()
  if (!json.success) {
    throw new ApiError(json.code ?? 500, json.message || '请求失败')
  }
  return json.data as T
}

export function get<T>(url: string): Promise<T> {
  return request<T>(url)
}

export function post<T>(url: string, body?: unknown): Promise<T> {
  return request<T>(url, { method: 'POST', body: body === undefined ? undefined : JSON.stringify(body) })
}

export async function uploadFile(file: File): Promise<string> {
  const form = new FormData()
  form.append('file', file)
  const data = await request<{ url: string }>('/api/upload', { method: 'POST', body: form })
  return data.url
}

export async function uploadVideo(file: File): Promise<string> {
  const form = new FormData()
  form.append('file', file)
  const data = await request<{ url: string }>('/api/upload/video', { method: 'POST', body: form })
  return data.url
}
