import { contextBridge, ipcRenderer } from 'electron'

contextBridge.exposeInMainWorld('desktop', {
  platform: process.platform,
  versions: {
    electron: process.versions.electron,
    chrome: process.versions.chrome,
    node: process.versions.node,
  },
  player: {
    openDemo: () => ipcRenderer.invoke('player:open-demo'),
    clips: () => ipcRenderer.invoke('player:clips'),
    openClip: (index: number) => ipcRenderer.invoke('player:open-clip', index),
    play: () => ipcRenderer.invoke('player:play'),
    pause: () => ipcRenderer.invoke('player:pause'),
    seek: (ms: number) => ipcRenderer.invoke('player:seek', ms),
    setVolume: (volume: number) => ipcRenderer.invoke('player:volume', volume),
    frame: () => ipcRenderer.invoke('player:frame'),
    state: () => ipcRenderer.invoke('player:state'),
  },
})
