export interface User {
  id: number
  username: string
  nickname: string
  avatar: string
  bio: string
  gender: number
  createTime: number
}

export interface Author {
  id: number
  nickname: string
  avatar: string
}

export interface Note {
  id: number
  userId: number
  title: string
  content: string
  mediaType: 'image' | 'video'
  videoUrl: string
  images: string[]
  tags: string[]
  likeCount: number
  favoriteCount: number
  commentCount: number
  createTime: number
  author: Author | null
  liked: boolean
  favorited: boolean
}

export interface CommentItem {
  id: number
  noteId: number
  userId: number
  parentId: number
  content: string
  createTime: number
  user: Author | null
  replyTo: Author | null
  children: CommentItem[]
}

export interface Paging {
  page: number
  limit: number
  total: number
}

export interface PageResult<T> {
  results: T[]
  paging: Paging
}

export interface UserProfile {
  user: User
  noteCount: number
  followingCount: number
  fansCount: number
  likedCount: number
  following: boolean
}
