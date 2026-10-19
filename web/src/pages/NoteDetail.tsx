import { useCallback, useEffect, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { actionApi, commentApi, noteApi } from '../api'
import { ApiError, mediaUrl } from '../api/client'
import { Avatar } from '../components/AppHeader'
import { formatCount, formatTime, HeartIcon, StarIcon } from '../components/NoteCard'
import { useAuth } from '../store/auth'
import type { CommentItem, Note } from '../types'

export default function NoteDetail() {
  const { id } = useParams()
  const noteId = Number(id)
  const { user } = useAuth()
  const navigate = useNavigate()

  const [note, setNote] = useState<Note | null>(null)
  const [error, setError] = useState('')
  const [imgIndex, setImgIndex] = useState(0)
  const [comments, setComments] = useState<CommentItem[]>([])
  const [commentText, setCommentText] = useState('')
  const [replyTo, setReplyTo] = useState<CommentItem | null>(null)

  useEffect(() => {
    noteApi
      .detail(noteId)
      .then(setNote)
      .catch((e) => setError(e.message || '加载失败'))
    loadComments()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [noteId])

  const loadComments = useCallback(() => {
    commentApi.list(noteId, 1).then((data) => setComments(data.results)).catch(() => {})
  }, [noteId])

  const requireLogin = () => {
    if (!user) {
      navigate('/login')
      return false
    }
    return true
  }

  const toggleLike = async () => {
    if (!note || !requireLogin()) return
    const liked = !note.liked
    setNote({ ...note, liked, likeCount: note.likeCount + (liked ? 1 : -1) })
    try {
      if (liked) await actionApi.like(note.id)
      else await actionApi.unlike(note.id)
    } catch {
      setNote({ ...note, liked: !liked, likeCount: note.likeCount })
    }
  }

  const toggleFavorite = async () => {
    if (!note || !requireLogin()) return
    const favorited = !note.favorited
    setNote({ ...note, favorited, favoriteCount: note.favoriteCount + (favorited ? 1 : -1) })
    try {
      if (favorited) await actionApi.favorite(note.id)
      else await actionApi.unfavorite(note.id)
    } catch {
      setNote({ ...note, favorited: !favorited, favoriteCount: note.favoriteCount })
    }
  }

  const submitComment = async () => {
    if (!note || !requireLogin()) return
    const content = commentText.trim()
    if (!content) return
    try {
      await commentApi.create({
        noteId: note.id,
        parentId: replyTo ? (replyTo.parentId > 0 ? replyTo.parentId : replyTo.id) : undefined,
        replyToUserId: replyTo?.userId,
        content,
      })
      setCommentText('')
      setReplyTo(null)
      setNote({ ...note, commentCount: note.commentCount + 1 })
      loadComments()
    } catch (e) {
      alert(e instanceof ApiError ? e.message : '评论失败')
    }
  }

  const removeComment = async (comment: CommentItem) => {
    if (!note) return
    await commentApi.remove(comment.id)
    setNote({ ...note, commentCount: Math.max(0, note.commentCount - 1) })
    loadComments()
  }

  const removeNote = async () => {
    if (!note || !confirm('确定删除这篇笔记吗？')) return
    await noteApi.remove(note.id)
    navigate('/')
  }

  if (error) return <div className="empty-tip">{error}</div>
  if (!note) return <div className="empty-tip">加载中…</div>

  return (
    <div className="note-detail-wrap">
      <div className="note-detail">
        <div className="note-gallery">
          {note.mediaType === 'video' && note.videoUrl ? (
            <video
              className="note-video"
              src={mediaUrl(note.videoUrl)}
              poster={note.images[0] ? mediaUrl(note.images[0]) : undefined}
              controls
              playsInline
              preload="metadata"
            />
          ) : note.images.length > 0 ? (
            <>
              <img src={mediaUrl(note.images[imgIndex])} alt={note.title} />
              {note.images.length > 1 && (
                <>
                  {imgIndex > 0 && (
                    <button className="gallery-arrow left" onClick={() => setImgIndex(imgIndex - 1)}>
                      ‹
                    </button>
                  )}
                  {imgIndex < note.images.length - 1 && (
                    <button className="gallery-arrow right" onClick={() => setImgIndex(imgIndex + 1)}>
                      ›
                    </button>
                  )}
                  <div className="gallery-dots">
                    {note.images.map((_, i) => (
                      <span key={i} className={i === imgIndex ? 'dot active' : 'dot'} onClick={() => setImgIndex(i)} />
                    ))}
                  </div>
                </>
              )}
            </>
          ) : (
            <div className="note-gallery-empty">图文笔记</div>
          )}
        </div>

        <div className="note-side">
          <div className="note-author-row">
            <Link to={`/user/${note.author?.id}`} className="note-author">
              <Avatar url={note.author?.avatar} name={note.author?.nickname} size={40} />
              <span>{note.author?.nickname || '神秘用户'}</span>
            </Link>
            {user && user.id !== note.userId && <FollowButton userId={note.userId} />}
            {user && user.id === note.userId && (
              <button className="text-btn danger" onClick={removeNote}>
                删除
              </button>
            )}
          </div>

          <div className="note-body">
            <h1 className="note-title">{note.title}</h1>
            <p className="note-content">{note.content}</p>
            {note.tags.length > 0 && (
              <div className="note-tags">
                {note.tags.map((tag) => (
                  <Link key={tag} to={`/?q=${encodeURIComponent(tag)}`} className="note-tag">
                    #{tag}
                  </Link>
                ))}
              </div>
            )}
            <div className="note-time">{formatTime(note.createTime)}</div>
          </div>

          <div className="note-actions">
            <button className={note.liked ? 'action active-like' : 'action'} onClick={toggleLike}>
              <HeartIcon filled={note.liked} /> {formatCount(note.likeCount)}
            </button>
            <button className={note.favorited ? 'action active-fav' : 'action'} onClick={toggleFavorite}>
              <StarIcon filled={note.favorited} /> {formatCount(note.favoriteCount)}
            </button>
            <span className="action comment-count">
              <CommentIcon /> {formatCount(note.commentCount)}
            </span>
          </div>

          <div className="comment-input-row">
            {replyTo && (
              <div className="reply-tip">
                回复 {replyTo.user?.nickname}
                <button onClick={() => setReplyTo(null)}>×</button>
              </div>
            )}
            <div className="comment-input">
              <input
                value={commentText}
                onChange={(e) => setCommentText(e.target.value)}
                placeholder={user ? '说点什么…' : '登录后评论'}
                onKeyDown={(e) => e.key === 'Enter' && submitComment()}
              />
              <button className="send-btn" onClick={submitComment} disabled={!commentText.trim()}>
                发送
              </button>
            </div>
          </div>

          <div className="comment-list">
            {comments.map((c) => (
              <CommentView
                key={c.id}
                comment={c}
                onReply={setReplyTo}
                onDelete={removeComment}
                currentUserId={user?.id}
              />
            ))}
            {comments.length === 0 && <div className="empty-comments">还没有评论，来抢沙发</div>}
          </div>
        </div>
      </div>
    </div>
  )
}

function CommentView({
  comment,
  onReply,
  onDelete,
  currentUserId,
}: {
  comment: CommentItem
  onReply: (c: CommentItem) => void
  onDelete: (c: CommentItem) => void
  currentUserId?: number
}) {
  return (
    <div className="comment-item">
      <Avatar url={comment.user?.avatar} name={comment.user?.nickname} size={32} />
      <div className="comment-main">
        <div className="comment-nickname">{comment.user?.nickname || '神秘用户'}</div>
        <div className="comment-content">{comment.content}</div>
        <div className="comment-meta">
          <span>{formatTime(comment.createTime)}</span>
          <button className="text-btn" onClick={() => onReply(comment)}>
            回复
          </button>
          {currentUserId === comment.userId && (
            <button className="text-btn danger" onClick={() => onDelete(comment)}>
              删除
            </button>
          )}
        </div>
        {comment.children.map((child) => (
          <div className="comment-item child" key={child.id}>
            <Avatar url={child.user?.avatar} name={child.user?.nickname} size={24} />
            <div className="comment-main">
              <div className="comment-nickname">{child.user?.nickname || '神秘用户'}</div>
              <div className="comment-content">
                {child.replyTo && child.replyTo.id !== comment.userId && (
                  <span className="reply-to">回复 @{child.replyTo.nickname}：</span>
                )}
                {child.content}
              </div>
              <div className="comment-meta">
                <span>{formatTime(child.createTime)}</span>
                <button className="text-btn" onClick={() => onReply(child)}>
                  回复
                </button>
                {currentUserId === child.userId && (
                  <button className="text-btn danger" onClick={() => onDelete(child)}>
                    删除
                  </button>
                )}
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}

function FollowButton({ userId }: { userId: number }) {
  const [following, setFollowing] = useState<boolean | null>(null)

  useEffect(() => {
    // 简单起见进入页面时不拉取关注状态，由操作后切换
    setFollowing(null)
  }, [userId])

  const toggle = async () => {
    try {
      if (following) {
        await actionApi.unfollow(userId)
        setFollowing(false)
      } else {
        await actionApi.follow(userId)
        setFollowing(true)
      }
    } catch {
      /* 未登录等 */
    }
  }

  return (
    <button className={following ? 'follow-btn following' : 'follow-btn'} onClick={toggle}>
      {following ? '已关注' : '关注'}
    </button>
  )
}

function CommentIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
      <path d="M21 12a8 8 0 0 1-8 8H5l-2 2V12a8 8 0 0 1 8-8h2a8 8 0 0 1 8 8Z" />
    </svg>
  )
}
