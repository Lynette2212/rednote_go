import { Link } from 'react-router-dom'
import type { Note } from '../types'
import { Avatar } from './AppHeader'

export default function NoteCard({ note }: { note: Note }) {
  return (
    <Link to={`/note/${note.id}`} className="note-card">
      <div className="note-cover">
        {note.images[0] ? (
          <img src={note.images[0]} alt={note.title} loading="lazy" />
        ) : (
          <div className="note-cover-empty">{note.title}</div>
        )}
      </div>
      <div className="note-card-body">
        <h3 className="note-card-title">{note.title}</h3>
        <div className="note-card-footer">
          <span className="note-card-author">
            <Avatar url={note.author?.avatar} name={note.author?.nickname} size={20} />
            <span className="note-card-nickname">{note.author?.nickname || '神秘用户'}</span>
          </span>
          <span className={`note-card-likes${note.liked ? ' liked' : ''}`}>
            <HeartIcon filled={note.liked} size={14} />
            {formatCount(note.likeCount)}
          </span>
        </div>
      </div>
    </Link>
  )
}

export function HeartIcon({ filled, size = 18 }: { filled?: boolean; size?: number }) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill={filled ? '#ff2442' : 'none'}
      stroke={filled ? '#ff2442' : 'currentColor'}
      strokeWidth="2"
    >
      <path d="M19.5 12.6 12 20l-7.5-7.4A5 5 0 1 1 12 6.1a5 5 0 1 1 7.5 6.5Z" />
    </svg>
  )
}

export function StarIcon({ filled, size = 18 }: { filled?: boolean; size?: number }) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill={filled ? '#ffb400' : 'none'}
      stroke={filled ? '#ffb400' : 'currentColor'}
      strokeWidth="2"
    >
      <path d="m12 2 3.1 6.3 6.9 1-5 4.9 1.2 6.8L12 17.8 5.8 21l1.2-6.8-5-4.9 6.9-1L12 2Z" />
    </svg>
  )
}

export function formatCount(n: number): string {
  if (n >= 10000) return (n / 10000).toFixed(1).replace(/\.0$/, '') + 'w'
  if (n >= 1000) return (n / 1000).toFixed(1).replace(/\.0$/, '') + 'k'
  return String(n)
}

export function formatTime(ts: number): string {
  if (!ts) return ''
  const diff = Date.now() - ts
  const minute = 60 * 1000
  const hour = 60 * minute
  const day = 24 * hour
  if (diff < minute) return '刚刚'
  if (diff < hour) return Math.floor(diff / minute) + ' 分钟前'
  if (diff < day) return Math.floor(diff / hour) + ' 小时前'
  if (diff < 7 * day) return Math.floor(diff / day) + ' 天前'
  const d = new Date(ts)
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}
