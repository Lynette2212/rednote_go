export interface Note {
  id: number
  userId: number
  title: string
  content: string
  images: string[]
  tags: string[]
  likeCount: number
  favoriteCount: number
  commentCount: number
  createTime: number
}

export interface UserProfile {
  id: number
  nickname: string
  avatar: string
  bio: string
}

export interface FeedResponse {
  code: number
  message: string
  data: {
    results: Note[]
    paging: {
      page: number
      limit: number
      total: number
    }
  }
}
