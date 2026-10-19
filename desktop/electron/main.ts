import { app, BrowserWindow, ipcMain } from 'electron'
import fs from 'node:fs'
import path from 'node:path'

const devServerUrl = process.env.VITE_DEV_SERVER_URL
const shotArg = process.argv.find((arg) => arg.startsWith('--screenshot='))
const shotPath = shotArg ? shotArg.slice('--screenshot='.length) : undefined
const delayArg = process.argv.find((arg) => arg.startsWith('--shot-delay='))
const playerMode = process.argv.includes('--player')
const clipArg = process.argv.find((arg) => arg.startsWith('--clip='))
const initialClip = clipArg ? Number(clipArg.split('=')[1]) || 0 : 0

interface NativeFrame {
  buffer: Uint8Array
  width: number
  height: number
  pts: number
  seq: number
}

interface NativeState {
  playing: boolean
  ended: boolean
  positionMs: number
  durationMs: number
  error: string
}

interface NativePlayer {
  open(source: string): boolean
  play(): void
  pause(): void
  seek(ms: number): void
  setVolume(volume: number): void
  getFrame(): NativeFrame | null
  getState(): NativeState
}

let player: NativePlayer | null = null

const demoClips = [
  { file: 'anime_sakura.mp4', title: '樱夜' },
  { file: 'anime_sunset.mp4', title: '海边黄昏' },
  { file: 'anime_neon.mp4', title: '霓虹雨夜' },
]

function clipPath(index: number) {
  return path.join(app.getAppPath(), 'assets', 'videos', demoClips[index].file)
}

function initNativePlayer() {
  try {
    const nativeDir = path.join(app.getAppPath(), 'native')
    process.env.PATH = `${path.join(nativeDir, 'third_party', 'ffmpeg', 'bin')};${process.env.PATH ?? ''}`
    const addon = require(path.join(nativeDir, 'build', 'Release', 'rednote_player.node')) as {
      Player: new () => NativePlayer
    }
    player = new addon.Player()

    ipcMain.handle('player:open-demo', () => {
      if (!player)
        return false
      return player.open(clipPath(0))
    })
    ipcMain.handle('player:clips', () => demoClips.map((clip, index) => ({ index, title: clip.title })))
    ipcMain.handle('player:open-clip', (_event, index: number) => {
      if (!player || index < 0 || index >= demoClips.length)
        return false
      return player.open(clipPath(index))
    })
    ipcMain.handle('player:play', () => player?.play())
    ipcMain.handle('player:pause', () => player?.pause())
    ipcMain.handle('player:seek', (_event, ms: number) => player?.seek(ms))
    ipcMain.handle('player:volume', (_event, volume: number) => player?.setVolume(volume))
    ipcMain.handle('player:frame', () => player?.getFrame() ?? null)
    ipcMain.handle('player:state', () => player?.getState() ?? null)
  } catch (error) {
    console.error('native player load failed', error)
  }
}

function createWindow() {
  const window = new BrowserWindow({
    width: 480,
    height: 900,
    minWidth: 390,
    minHeight: 700,
    title: '小红薯 · 桌面端',
    backgroundColor: '#0f0f12',
    autoHideMenuBar: true,
    webPreferences: {
      preload: path.join(__dirname, 'preload.js'),
      contextIsolation: true,
      nodeIntegration: false,
      sandbox: true,
      webSecurity: true,
      backgroundThrottling: false,
    },
  })

  if (shotPath) {
    window.setAlwaysOnTop(true)
    window.show()
    window.focus()
  }

  if (devServerUrl) {
    void window.loadURL(devServerUrl)
  } else {
    void window.loadFile(path.join(__dirname, '..', 'dist', 'index.html'))
  }
  window.webContents.on('console-message', (...args: unknown[]) => {
    console.log('renderer console:', ...args.slice(1))
  })

  if (shotPath) {
    const delay = delayArg ? Number(delayArg.split('=')[1]) : playerMode ? 7000 : 5000
    window.webContents.once('did-finish-load', () => {
      if (playerMode) {
        setTimeout(() => {
          void window.webContents.executeJavaScript(
            `window.dispatchEvent(new CustomEvent('rednote:open-player', { detail: { clip: ${initialClip} } }))`,
          )
        }, 1200)
      }
      setTimeout(async () => {
        try {
          const bodyText = await window.webContents.executeJavaScript('document.body.innerText.slice(0, 300)')
          console.log('renderer body:', bodyText)
          const image = await window.webContents.capturePage()
          fs.writeFileSync(shotPath, image.toPNG())
        } catch (error) {
          console.error('capture failed', error)
        }
        app.quit()
      }, delay)
    })
  }
}

app.whenReady().then(() => {
  initNativePlayer()
  createWindow()
  app.on('activate', () => {
    if (BrowserWindow.getAllWindows().length === 0)
      createWindow()
  })
})

app.on('window-all-closed', () => {
  if (process.platform !== 'darwin')
    app.quit()
})
