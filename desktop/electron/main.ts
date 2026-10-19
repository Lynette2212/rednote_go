import { app, BrowserWindow, ipcMain } from 'electron'
import fs from 'node:fs'
import path from 'node:path'

const devServerUrl = process.env.VITE_DEV_SERVER_URL
const shotArg = process.argv.find((arg) => arg.startsWith('--screenshot='))
const shotPath = shotArg ? shotArg.slice('--screenshot='.length) : undefined
const delayArg = process.argv.find((arg) => arg.startsWith('--shot-delay='))
const playerMode = process.argv.includes('--player')

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
      return player.open(path.join(app.getAppPath(), 'assets', 'test.mp4'))
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
    },
  })

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
            "window.dispatchEvent(new Event('rednote:open-player'))",
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
