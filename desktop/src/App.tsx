import { useCallback, useEffect, useRef, useState } from 'react'
import type { WheelEvent } from 'react'
import PlayerView from './PlayerView'
import type { FeedResponse, Note, UserProfile } from './types'

const API_BASE = import.meta.env.VITE_API_BASE ?? 'http://127.0.0.1:8082'

function assetUrl(path?: string) {
  if (!path)
    return ''
  return path.startsWith('http') ? path : `${API_BASE}${path}`
}

function Icon({ name }: { name: 'heart' | 'star' | 'comment' | 'share' }) {
  const paths = {
    heart: 'M12 21s-7.5-4.7-9.5-9A5.4 5.4 0 0 1 12 5.5 5.4 5.4 0 0 1 21.5 12c-2 4.3-9.5 9-9.5 9z',
    star: 'M12 3l2.9 5.9 6.5.9-4.7 4.6 1.1 6.5L12 17.8 6.2 20.9l1.1-6.5L2.6 9.8l6.5-.9L12 3z',
    comment: 'M21 12a8 8 0 0 1-8 8H5l-2 2V12a8 8 0 0 1 8-8h2a8 8 0 0 1 8 8z',
    share: 'M4 12v7h16v-7M12 3v12M8 7l4-4 4 4',
  }
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true">
      <path d={paths[name]} fill={name === 'heart' || name === 'star' ? 'none' : 'none'} />
    </svg>
  )
}

export default function App() {
  const [notes, setNotes] = useState<Note[]>([])
  const [index, setIndex] = useState(0)
  const [users, setUsers] = useState<Record<number, UserProfile>>({})
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [playerOpen, setPlayerOpen] = useState(false)
  const [playerClip, setPlayerClip] = useState(0)
  const wheelLock = useRef(0)

  useEffect(() => {
    let cancelled = false
    fetch(`${API_BASE}/api/note/feed?page=1&limit=20`)
      .then((response) => response.json())
      .then((payload: FeedResponse) => {
        if (cancelled)
          return
        if (payload.code !== 0)
          throw new Error(payload.message || '加载失败')
        setNotes(payload.data.results)
      })
      .catch((cause: Error) => {
        if (!cancelled)
          setError(`无法连接后端：${cause.message}`)
      })
      .finally(() => {
        if (!cancelled)
          setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [])

  const note = notes[index]

  useEffect(() => {
    if (!note)
      return
    const userId = note.userId
    if (users[userId])
      return
    let cancelled = false
    fetch(`${API_BASE}/api/user/profile/${userId}`)
      .then((response) => response.json())
      .then((payload: { code: number; data?: { user: UserProfile } }) => {
        if (cancelled || payload.code !== 0 || !payload.data?.user)
          return
        const profile = payload.data.user
        setUsers((current) => ({ ...current, [userId]: profile }))
      })
      .catch(() => undefined)
    return () => {
      cancelled = true
    }
  }, [note, users])

  const move = useCallback((direction: 1 | -1) => {
    setIndex((current) => Math.min(notes.length - 1, Math.max(0, current + direction)))
  }, [notes.length])

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (playerOpen)
        return
      if (event.key === 'ArrowDown' || event.key === 'PageDown')
        move(1)
      else if (event.key === 'ArrowUp' || event.key === 'PageUp')
        move(-1)
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [move, playerOpen])

  useEffect(() => {
    const openPlayer = (event: Event) => {
      const detail = (event as CustomEvent<{ clip?: number }>).detail
      setPlayerClip(detail?.clip ?? 0)
      setPlayerOpen(true)
    }
    window.addEventListener('rednote:open-player', openPlayer)
    return () => window.removeEventListener('rednote:open-player', openPlayer)
  }, [])

  const onWheel = (event: WheelEvent) => {
    if (playerOpen)
      return
    const now = Date.now()
    if (now - wheelLock.current < 520 || Math.abs(event.deltaY) < 12)
      return
    wheelLock.current = now
    move(event.deltaY > 0 ? 1 : -1)
  }

  const author = note ? users[note.userId] : undefined
  const avatar = assetUrl(author?.avatar)
  const cover = note ? assetUrl(note.images[0]) : ''

  return (
    <div className="app" onWheel={onWheel}>
      <div className="stage">
        {note && (
          <article className="card" key={note.id}>
            <div
              className="cover"
              onClick={() => setPlayerOpen(true)}
              style={cover ? { backgroundImage: `url(${cover})` } : undefined}
            />
            <div className="shade shade-top" />
            <div className="shade shade-bottom" />

            <header className="topbar">
              <div className="brand">小红薯</div>
              <nav className="tabs">
                <button className="tab" disabled>关注</button>
                <button className="tab active">推荐</button>
              </nav>
              <div className="counter">{index + 1} / {notes.length}</div>
            </header>

            <aside className="actions">
              <button className="action" title="点赞">
                <Icon name="heart" />
                <span>{note.likeCount}</span>
              </button>
              <button className="action" title="收藏">
                <Icon name="star" />
                <span>{note.favoriteCount}</span>
              </button>
              <button className="action" title="评论">
                <Icon name="comment" />
                <span>{note.commentCount}</span>
              </button>
              <button className="action" title="分享">
                <Icon name="share" />
                <span>分享</span>
              </button>
            </aside>

            <footer className="caption">
              <div className="author">
                <div className="avatar">
                  {avatar ? <img src={avatar} alt="" /> : (author?.nickname || '红').slice(0, 1)}
                </div>
                <span className="nickname">{author?.nickname || `用户 ${note.userId}`}</span>
              </div>
              <h1 className="title">{note.title}</h1>
              <p>{note.content}</p>
              <div className="tags">
                {note.tags.map((tag) => <span key={tag}>#{tag}</span>)}
              </div>
            </footer>

            <div className="hint">滚轮或 ↑ ↓ 切换</div>
          </article>
        )}

        {loading && <div className="state">正在加载信息流…</div>}
        {error && <div className="state error">{error}</div>}
      </div>

      {playerOpen && (
        <PlayerView
          title={note?.title ?? '播放演示'}
          initialClip={playerClip}
          onClose={() => setPlayerOpen(false)}
        />
      )}
    </div>
  )
}
