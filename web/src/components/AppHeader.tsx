import { useState, type FormEvent } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useAuth } from '../store/auth'

export default function AppHeader() {
  const { user, logout } = useAuth()
  const navigate = useNavigate()
  const [keyword, setKeyword] = useState('')
  const [menuOpen, setMenuOpen] = useState(false)

  const onSearch = (e: FormEvent) => {
    e.preventDefault()
    const q = keyword.trim()
    if (q) navigate(`/?q=${encodeURIComponent(q)}`)
  }

  return (
    <header className="app-header">
      <div className="header-inner">
        <Link to="/" className="logo">
          小红薯
        </Link>
        <form className="search-box" onSubmit={onSearch}>
          <input
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
            placeholder="搜索笔记"
          />
          <button type="submit" className="search-btn" aria-label="搜索">
            <SearchIcon />
          </button>
        </form>
        <nav className="header-actions">
          {user ? (
            <>
              <Link to="/publish" className="publish-btn">
                发布笔记
              </Link>
              <div className="user-menu-wrap">
                <button className="avatar-btn" onClick={() => setMenuOpen((v) => !v)}>
                  <Avatar url={user.avatar} name={user.nickname} size={36} />
                </button>
                {menuOpen && (
                  <div className="user-menu" onMouseLeave={() => setMenuOpen(false)}>
                    <Link to={`/user/${user.id}`} onClick={() => setMenuOpen(false)}>
                      个人主页
                    </Link>
                    <button
                      onClick={() => {
                        logout()
                        setMenuOpen(false)
                        navigate('/')
                      }}
                    >
                      退出登录
                    </button>
                  </div>
                )}
              </div>
            </>
          ) : (
            <Link to="/login" className="login-btn">
              登录
            </Link>
          )}
        </nav>
      </div>
    </header>
  )
}

export function Avatar({ url, name, size = 32 }: { url?: string; name?: string; size?: number }) {
  if (url) {
    return <img className="avatar" src={url} alt={name} style={{ width: size, height: size }} />
  }
  return (
    <span className="avatar avatar-placeholder" style={{ width: size, height: size, fontSize: size * 0.45 }}>
      {(name || '红')[0]}
    </span>
  )
}

function SearchIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
      <circle cx="11" cy="11" r="7" />
      <path d="m20 20-3.5-3.5" />
    </svg>
  )
}
