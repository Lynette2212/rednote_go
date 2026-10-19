import { useCallback, useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { actionApi, noteApi, userApi } from '../api'
import { uploadFile } from '../api/client'
import { Avatar } from '../components/AppHeader'
import Masonry from '../components/Masonry'
import NoteCard, { formatCount } from '../components/NoteCard'
import { useAuth } from '../store/auth'
import type { Note, UserProfile } from '../types'

type Tab = 'notes' | 'favorites' | 'likes'

export default function Profile() {
  const { id } = useParams()
  const userId = Number(id)
  const { user: currentUser, setUser } = useAuth()

  const [profile, setProfile] = useState<UserProfile | null>(null)
  const [tab, setTab] = useState<Tab>('notes')
  const [notes, setNotes] = useState<Note[]>([])
  const [editing, setEditing] = useState(false)

  const isMe = currentUser?.id === userId

  const loadProfile = useCallback(() => {
    userApi.profile(userId).then(setProfile).catch(() => setProfile(null))
  }, [userId])

  useEffect(() => {
    loadProfile()
  }, [loadProfile])

  useEffect(() => {
    const load = async () => {
      let data
      if (tab === 'notes') data = await noteApi.userNotes(userId, 1)
      else if (tab === 'favorites') data = await noteApi.userFavorites(userId, 1)
      else data = await noteApi.userLikes(userId, 1)
      setNotes(data.results)
    }
    load().catch(() => setNotes([]))
  }, [userId, tab])

  const toggleFollow = async () => {
    if (!profile) return
    if (profile.following) {
      await actionApi.unfollow(userId)
      setProfile({ ...profile, following: false, fansCount: profile.fansCount - 1 })
    } else {
      await actionApi.follow(userId)
      setProfile({ ...profile, following: true, fansCount: profile.fansCount + 1 })
    }
  }

  if (!profile) return <div className="empty-tip">加载中…</div>
  const { user } = profile

  return (
    <div className="profile-page">
      <div className="profile-card">
        <Avatar url={user.avatar} name={user.nickname} size={80} />
        <div className="profile-info">
          <h2>{user.nickname}</h2>
          <div className="profile-username">小红薯号：{user.username}</div>
          {user.bio && <div className="profile-bio">{user.bio}</div>}
          <div className="profile-stats">
            <span>
              <b>{formatCount(profile.followingCount)}</b> 关注
            </span>
            <span>
              <b>{formatCount(profile.fansCount)}</b> 粉丝
            </span>
            <span>
              <b>{formatCount(profile.likedCount)}</b> 获赞
            </span>
            <span>
              <b>{formatCount(profile.noteCount)}</b> 笔记
            </span>
          </div>
        </div>
        <div className="profile-actions">
          {isMe ? (
            <button className="edit-btn" onClick={() => setEditing(true)}>
              编辑资料
            </button>
          ) : currentUser ? (
            <button className={profile.following ? 'follow-btn following' : 'follow-btn'} onClick={toggleFollow}>
              {profile.following ? '已关注' : '关注'}
            </button>
          ) : null}
        </div>
      </div>

      <div className="profile-tabs">
        <button className={tab === 'notes' ? 'tab active' : 'tab'} onClick={() => setTab('notes')}>
          笔记
        </button>
        <button className={tab === 'favorites' ? 'tab active' : 'tab'} onClick={() => setTab('favorites')}>
          收藏
        </button>
        <button className={tab === 'likes' ? 'tab active' : 'tab'} onClick={() => setTab('likes')}>
          赞过
        </button>
      </div>

      <Masonry>
        {notes.map((note) => (
          <NoteCard key={note.id} note={note} />
        ))}
      </Masonry>
      {notes.length === 0 && <div className="empty-tip">暂无内容</div>}

      {editing && isMe && (
        <EditProfileModal
          profile={profile}
          onClose={() => setEditing(false)}
          onSaved={(u) => {
            setUser(u)
            setEditing(false)
            loadProfile()
          }}
        />
      )}
    </div>
  )
}

function EditProfileModal({
  profile,
  onClose,
  onSaved,
}: {
  profile: UserProfile
  onClose: () => void
  onSaved: (u: import('../types').User) => void
}) {
  const [nickname, setNickname] = useState(profile.user.nickname)
  const [bio, setBio] = useState(profile.user.bio)
  const [avatar, setAvatar] = useState(profile.user.avatar)
  const [saving, setSaving] = useState(false)

  const changeAvatar = async (file: File) => {
    const url = await uploadFile(file)
    setAvatar(url)
  }

  const save = async () => {
    setSaving(true)
    try {
      const u = await userApi.update({ nickname, avatar, bio, gender: profile.user.gender })
      onSaved(u)
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="modal-mask" onClick={onClose}>
      <div className="modal-card" onClick={(e) => e.stopPropagation()}>
        <h3>编辑资料</h3>
        <label className="avatar-edit">
          <Avatar url={avatar} name={nickname} size={64} />
          <span>更换头像</span>
          <input
            type="file"
            accept="image/*"
            hidden
            onChange={(e) => e.target.files?.[0] && changeAvatar(e.target.files[0])}
          />
        </label>
        <input value={nickname} onChange={(e) => setNickname(e.target.value)} placeholder="昵称" />
        <textarea value={bio} onChange={(e) => setBio(e.target.value)} placeholder="简介" rows={3} />
        <div className="modal-actions">
          <button className="text-btn" onClick={onClose}>
            取消
          </button>
          <button className="submit-btn" onClick={save} disabled={saving || !nickname.trim()}>
            {saving ? '保存中…' : '保存'}
          </button>
        </div>
      </div>
    </div>
  )
}
