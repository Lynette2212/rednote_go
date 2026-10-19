export {}

declare global {
  interface Window {
    desktop?: {
      platform: string
      versions: { electron: string; chrome: string; node: string }
      player: {
        openDemo(): Promise<boolean>
        clips(): Promise<{ index: number; title: string }[]>
        openClip(index: number): Promise<boolean>
        play(): Promise<void>
        pause(): Promise<void>
        seek(ms: number): Promise<void>
        setVolume(volume: number): Promise<void>
        frame(): Promise<{
          buffer: Uint8Array
          width: number
          height: number
          pts: number
          seq: number
        } | null>
        state(): Promise<{
          playing: boolean
          ended: boolean
          positionMs: number
          durationMs: number
          error: string
        } | null>
      }
    }
  }
}
