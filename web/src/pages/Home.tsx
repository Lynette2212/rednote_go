import { useCallback, useEffect, useRef, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { noteApi } from '../api'
import Masonry from '../components/Masonry'
import NoteCard from '../components/NoteCard'
import { useAuth } from '../store/auth'
import type { Note } from '../types'

type Tab = 'recommend' | 'follow'

export default function Home() {
  const { user } = useAuth()
  const [searchParams, setSearchParams] = useSearchParams()
  const q = searchParams.get('q') || ''
  const tab = (searchParams.get('tab') as Tab) || 'recommend'

  const [notes, setNotes] = useState<Note[]>([])
  const [page, setPage] = useState(1)
  const [hasMore, setHasMore] = useState(true)
  const [loading, setLoading] = useState(false)
  const loaderRef = useRef<HTMLDivElement>(null)

  const loadPage = useCallback(
    async (p: number, replace: boolean) => {
      setLoading(true)
      try {
        let data
        if (q) data = await noteApi.search(q, p)
        else if (tab === 'follow') data = await noteApi.followFeed(p)
        else data = await noteApi.feed(p)
        setNotes((prev) => (replace ? data.results : [...prev, ...data.results]))
        setHasMore(data.results.length > 0 && notes.length + data.results.length < data.paging.total)
        setPage(p)
      } catch {
        if (tab === 'follow' && !user) setHasMore(false)
      } finally {
        setLoading(false)
      }
    },
    [q, tab, notes.length, user],
  )

  useEffect(() => {
    setNotes([])
    setHasMore(true)
    loadPage(1, true)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [q, tab])

  useEffect(() => {
    const el = loaderRef.current
    if (!el) return
    const observer = new IntersectionObserver((entries) => {
      if (entries[0].isIntersecting && hasMore && !loading) {
        loadPage(page + 1, false)
      }
    })
    observer.observe(el)
    return () => observer.disconnect()
  }, [hasMore, loading, page, loadPage])

  const switchTab = (t: Tab) => {
    setSearchParams(t === 'recommend' ? {} : { tab: t })
  }

  return (
    <div className="home">
      {!q && (
        <div className="home-tabs">
          <button className={tab === 'follow' ? 'tab active' : 'tab'} onClick={() => switchTab('follow')}>
            关注
          </button>
          <button className={tab === 'recommend' ? 'tab active' : 'tab'} onClick={() => switchTab('recommend')}>
            推荐
          </button>
        </div>
      )}
      {q && <div className="search-result-tip">「{q}」的搜索结果</div>}

      {tab === 'follow' && !user ? (
        <div className="empty-tip">登录后查看关注的人的笔记</div>
      ) : (
        <>
          <Masonry>
            {notes.map((note) => (
              <NoteCard key={note.id} note={note} />
            ))}
          </Masonry>
          <div ref={loaderRef} className="load-more">
            {loading ? '加载中…' : hasMore ? '' : notes.length > 0 ? '没有更多了' : '暂无笔记'}
          </div>
        </>
      )}
    </div>
  )
}
