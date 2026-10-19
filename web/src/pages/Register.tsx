import { useState, type FormEvent } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { ApiError } from '../api/client'
import { useAuth } from '../store/auth'

export default function Register() {
  const { register } = useAuth()
  const navigate = useNavigate()
  const [username, setUsername] = useState('')
  const [nickname, setNickname] = useState('')
  const [password, setPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    if (password !== confirm) {
      setError('两次输入的密码不一致')
      return
    }
    setLoading(true)
    setError('')
    try {
      await register(username, nickname, password)
      navigate('/')
    } catch (err) {
      setError(err instanceof ApiError ? err.message : '注册失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="auth-page">
      <form className="auth-card" onSubmit={submit}>
        <h2 className="auth-title">注册小红薯</h2>
        <input value={username} onChange={(e) => setUsername(e.target.value)} placeholder="用户名（3-32 位）" autoFocus />
        <input value={nickname} onChange={(e) => setNickname(e.target.value)} placeholder="昵称（选填）" />
        <input
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          placeholder="密码（至少 6 位）"
        />
        <input
          type="password"
          value={confirm}
          onChange={(e) => setConfirm(e.target.value)}
          placeholder="确认密码"
        />
        {error && <div className="form-error">{error}</div>}
        <button className="submit-btn" type="submit" disabled={loading || !username || !password}>
          {loading ? '注册中…' : '注册'}
        </button>
        <div className="auth-switch">
          已有账号？<Link to="/login">去登录</Link>
        </div>
      </form>
    </div>
  )
}
