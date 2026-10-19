import { useEffect, useRef, useState, type ReactNode } from 'react'

// 简易瀑布流：把子项依次分配到当前最矮的列，保证阅读顺序为从左到右
export default function Masonry({ children, columns = 5 }: { children: ReactNode[]; columns?: number }) {
  const [cols, setCols] = useState(columns)
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const el = ref.current
    if (!el) return
    const compute = () => {
      const w = el.clientWidth
      if (w < 560) setCols(2)
      else if (w < 880) setCols(3)
      else if (w < 1200) setCols(4)
      else setCols(5)
    }
    compute()
    const observer = new ResizeObserver(compute)
    observer.observe(el)
    return () => observer.disconnect()
  }, [])

  const buckets: ReactNode[][] = Array.from({ length: cols }, () => [])
  const heights = new Array(cols).fill(0)
  children.forEach((child) => {
    let minIdx = 0
    for (let i = 1; i < cols; i++) {
      if (heights[i] < heights[minIdx]) minIdx = i
    }
    buckets[minIdx].push(child)
    heights[minIdx] += 1
  })

  return (
    <div className="masonry" ref={ref}>
      {buckets.map((bucket, i) => (
        <div className="masonry-col" key={i}>
          {bucket}
        </div>
      ))}
    </div>
  )
}
