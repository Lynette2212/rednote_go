import { useRef, useState, type ChangeEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { noteApi } from '../api'
import { ApiError, mediaUrl, uploadFile, uploadVideo } from '../api/client'
import { useAuth } from '../store/auth'

export default function Publish() {
  const { user } = useAuth()
  const navigate = useNavigate()
  const fileInputRef = useRef<HTMLInputElement>(null)
  const videoInputRef = useRef<HTMLInputElement>(null)

  const [images, setImages] = useState<string[]>([])
  const [videoUrl, setVideoUrl] = useState('')
  const [videoName, setVideoName] = useState('')
  const [uploading, setUploading] = useState(false)
  const [videoUploading, setVideoUploading] = useState(false)
  const [title, setTitle] = useState('')
  const [content, setContent] = useState('')
  const [tagInput, setTagInput] = useState('')
  const [tags, setTags] = useState<string[]>([])
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')

  if (!user) {
    navigate('/login')
    return null
  }

  const onSelectFiles = async (e: ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(e.target.files || [])
    if (files.length === 0) return
    setUploading(true)
    setError('')
    try {
      for (const file of files.slice(0, 9 - images.length)) {
        const url = await uploadFile(file)
        setImages((prev) => [...prev, url])
      }
    } catch (err) {
      setError(err instanceof ApiError ? err.message : '上传失败')
    } finally {
      setUploading(false)
      e.target.value = ''
    }
  }

  const onSelectVideo = async (e: ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    if (file.size > 200 * 1024 * 1024) {
      setError('视频不能超过 200MB')
      e.target.value = ''
      return
    }
    setVideoUploading(true)
    setError('')
    try {
      const url = await uploadVideo(file)
      setVideoUrl(url)
      setVideoName(file.name)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : '视频上传失败')
    } finally {
      setVideoUploading(false)
      e.target.value = ''
    }
  }

  const addTag = () => {
    const t = tagInput.trim().replace(/^#/, '')
    if (t && !tags.includes(t) && tags.length < 10) {
      setTags([...tags, t])
    }
    setTagInput('')
  }

  const submit = async () => {
    if (!videoUrl && images.length === 0) {
      setError('请至少上传一张图片或一个视频')
      return
    }
    if (!title.trim()) {
      setError('请填写标题')
      return
    }
    setSubmitting(true)
    setError('')
    try {
      const note = await noteApi.create({
        title: title.trim(),
        content: content.trim(),
        images,
        tags,
        videoUrl: videoUrl || undefined,
      })
      navigate(`/note/${note.id}`)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : '发布失败')
      setSubmitting(false)
    }
  }

  return (
    <div className="publish-page">
      <h2>发布笔记</h2>
      <div className="publish-form">
        <div className="video-upload">
          {videoUrl ? (
            <div className="video-preview">
              <video src={mediaUrl(videoUrl)} controls preload="metadata" />
              <div className="video-meta">
                <span>{videoName || '视频'}</span>
                <button
                  onClick={() => {
                    setVideoUrl('')
                    setVideoName('')
                  }}
                >
                  移除视频
                </button>
              </div>
            </div>
          ) : (
            <button
              className="upload-add video-add"
              onClick={() => videoInputRef.current?.click()}
              disabled={videoUploading}
            >
              {videoUploading ? '视频上传中…' : '+ 添加视频（mp4 / webm，≤200MB）'}
            </button>
          )}
          <input
            ref={videoInputRef}
            type="file"
            accept="video/mp4,video/webm"
            hidden
            onChange={onSelectVideo}
          />
        </div>

        <div className="upload-label">{videoUrl ? '封面 / 配图（可选）' : '图片'}</div>
        <div className="upload-area">
          {images.map((url, i) => (
            <div className="upload-preview" key={url}>
              <img src={mediaUrl(url)} alt="" />
              <button className="remove-img" onClick={() => setImages(images.filter((_, idx) => idx !== i))}>
                ×
              </button>
            </div>
          ))}
          {images.length < 9 && (
            <button className="upload-add" onClick={() => fileInputRef.current?.click()} disabled={uploading}>
              {uploading ? '上传中…' : '+ 添加图片'}
            </button>
          )}
          <input
            ref={fileInputRef}
            type="file"
            accept="image/*"
            multiple
            hidden
            onChange={onSelectFiles}
          />
        </div>

        <input
          className="title-input"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          placeholder="填写标题，会有更多赞哦～"
          maxLength={64}
        />
        <textarea
          className="content-input"
          value={content}
          onChange={(e) => setContent(e.target.value)}
          placeholder="分享此刻的想法…"
          rows={6}
        />
        <div className="tags-row">
          {tags.map((tag) => (
            <span className="tag-chip" key={tag}>
              #{tag}
              <button onClick={() => setTags(tags.filter((t) => t !== tag))}>×</button>
            </span>
          ))}
          <input
            value={tagInput}
            onChange={(e) => setTagInput(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && (e.preventDefault(), addTag())}
            placeholder="添加话题标签，回车确认"
          />
        </div>

        {error && <div className="form-error">{error}</div>}
        <button className="submit-btn" onClick={submit} disabled={submitting}>
          {submitting ? '发布中…' : '发布'}
        </button>
      </div>
    </div>
  )
}
