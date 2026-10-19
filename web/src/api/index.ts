import { get, post } from './client'
import type { CommentItem, Note, PageResult, User, UserProfile } from '../types'

export const authApi = {
  register: (username: string, nickname: string, password: string) =>
    post<{ token: string; user: User }>('/api/register', { username, nickname, password }),
  login: (username: string, password: string) =>
    post<{ token: string; user: User }>('/api/login', { username, password }),
  me: () => get<User>('/api/user/info'),
}

export const userApi = {
  profile: (id: number) => get<UserProfile>(`/api/user/profile/${id}`),
  update: (data: { nickname: string; avatar: string; bio: string; gender: number }) =>
    post<User>('/api/user/update', data),
}

export interface NoteListParams {
  page: number
  limit?: number
  userId?: number
  q?: string
}

function qs(params: Record<string, string | number | undefined>): string {
  const search = new URLSearchParams()
  Object.entries(params).forEach(([k, v]) => {
    if (v !== undefined && v !== '') search.set(k, String(v))
  })
  const s = search.toString()
  return s ? `?${s}` : ''
}

export const noteApi = {
  feed: (page: number) => get<PageResult<Note>>(`/api/note/feed${qs({ page, limit: 20 })}`),
  followFeed: (page: number) => get<PageResult<Note>>(`/api/note/follow_feed${qs({ page, limit: 20 })}`),
  userNotes: (userId: number, page: number) => get<PageResult<Note>>(`/api/note/user_notes${qs({ userId, page, limit: 20 })}`),
  userLikes: (userId: number, page: number) => get<PageResult<Note>>(`/api/note/user_likes${qs({ userId, page, limit: 20 })}`),
  userFavorites: (userId: number, page: number) => get<PageResult<Note>>(`/api/note/user_favorites${qs({ userId, page, limit: 20 })}`),
  search: (q: string, page: number) => get<PageResult<Note>>(`/api/note/search${qs({ q, page, limit: 20 })}`),
  detail: (id: number) => get<Note>(`/api/note/${id}`),
  create: (data: { title: string; content: string; images: string[]; tags: string[]; videoUrl?: string }) =>
    post<Note>('/api/note/create', data),
  remove: (id: number) => post<void>(`/api/note/delete/${id}`),
}

export const commentApi = {
  list: (noteId: number, page: number) => get<PageResult<CommentItem>>(`/api/comment/list${qs({ noteId, page, limit: 10 })}`),
  create: (data: { noteId: number; parentId?: number; replyToUserId?: number; content: string }) =>
    post<CommentItem>('/api/comment/create', data),
  remove: (id: number) => post<void>(`/api/comment/delete/${id}`),
}

export const actionApi = {
  like: (noteId: number) => post<void>('/api/like/add', { noteId }),
  unlike: (noteId: number) => post<void>('/api/like/remove', { noteId }),
  favorite: (noteId: number) => post<void>('/api/favorite/add', { noteId }),
  unfavorite: (noteId: number) => post<void>('/api/favorite/remove', { noteId }),
  follow: (userId: number) => post<void>('/api/follow/add', { userId }),
  unfollow: (userId: number) => post<void>('/api/follow/remove', { userId }),
}
