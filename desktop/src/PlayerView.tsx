import { useEffect, useRef, useState } from 'react'

interface PlayerViewProps {
  title: string
  onClose: () => void
}

interface PlayerState {
  playing: boolean
  ended: boolean
  positionMs: number
  durationMs: number
  error: string
}

const initial: PlayerState = {
  playing: false,
  ended: false,
  positionMs: 0,
  durationMs: 0,
  error: '',
}

function formatTime(ms: number) {
  const total = Math.max(0, Math.floor(ms / 1000))
  const minutes = Math.floor(total / 60)
  const seconds = total % 60
  return `${minutes}:${seconds < 10 ? '0' : ''}${seconds}`
}

export default function PlayerView({ title, onClose }: PlayerViewProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const [state, setState] = useState<PlayerState>(initial)
  const [dragging, setDragging] = useState(false)
  const [dragValue, setDragValue] = useState(0)
  const [volume, setVolume] = useState(70)

  useEffect(() => {
    const api = window.desktop?.player
    if (!api) {
      setState((current) => ({ ...current, error: '原生播放器未加载（请先构建 native 模块）' }))
      return
    }

    let stopped = false
    let busy = false
    let raf = 0
    let timer = 0

    const drawLoop = async () => {
      if (stopped)
        return
      if (!busy) {
        busy = true
        try {
          const frame = await api.frame()
          const canvas = canvasRef.current
          if (frame && canvas) {
            if (canvas.width !== frame.width || canvas.height !== frame.height) {
              canvas.width = frame.width
              canvas.height = frame.height
            }
            const context = canvas.getContext('2d')
            if (context) {
              const pixels = new Uint8ClampedArray(frame.buffer)
              context.putImageData(new ImageData(pixels, frame.width, frame.height), 0, 0)
            }
          }
        } catch {
          // 单帧读取失败不打断播放
        } finally {
          busy = false
        }
      }
      raf = requestAnimationFrame(drawLoop)
    }

    const start = async () => {
      const opened = await api.openDemo()
      if (stopped)
        return
      if (!opened) {
        const current = await api.state()
        setState((value) => ({ ...value, error: current?.error || '打开视频失败' }))
        return
      }
      await api.setVolume(volume)
      await api.play()
      drawLoop()
      timer = window.setInterval(async () => {
        const current = await api.state()
        if (!stopped && current)
          setState(current)
      }, 250)
    }

    void start()

    return () => {
      stopped = true
      cancelAnimationFrame(raf)
      clearInterval(timer)
      void api.pause()
    }
  }, [])

  const togglePlay = async () => {
    const api = window.desktop?.player
    if (!api)
      return
    if (state.playing)
      await api.pause()
    else
      await api.play()
  }

  const commitSeek = async () => {
    setDragging(false)
    await window.desktop?.player.seek(dragValue)
  }

  const changeVolume = async (value: number) => {
    setVolume(value)
    await window.desktop?.player.setVolume(value)
  }

  const duration = state.durationMs || 1
  const shownPosition = dragging ? dragValue : state.positionMs

  return (
    <div className="player">
      <div className="player-top">
        <button className="back" onClick={onClose}>返回</button>
        <span className="player-title">{title}</span>
        <span className="player-badge">FFmpeg 8.1 · 软解</span>
      </div>

      <div className="player-stage">
        <canvas ref={canvasRef} className="player-canvas" />
      </div>

      {state.error && <div className="player-error">{state.error}</div>}

      <div className="player-controls">
        <input
          className="range"
          type="range"
          min={0}
          max={duration}
          value={shownPosition}
          onPointerDown={() => {
            setDragging(true)
            setDragValue(shownPosition)
          }}
          onChange={(event) => setDragValue(Number(event.target.value))}
          onPointerUp={() => void commitSeek()}
        />

        <div className="player-row">
          <button className="play-button" onClick={() => void togglePlay()}>
            {state.playing ? '暂停' : state.ended ? '重播' : '播放'}
          </button>
          <span className="time">{formatTime(shownPosition)} / {formatTime(state.durationMs)}</span>
          <span className="spacer" />
          <span className="volume-label">音量</span>
          <input
            className="range volume"
            type="range"
            min={0}
            max={100}
            value={volume}
            onChange={(event) => void changeVolume(Number(event.target.value))}
          />
        </div>
      </div>
    </div>
  )
}
