import { create } from 'zustand'

export interface WxAccount {
  id: number
  tenant_id: number
  name: string
  appid: string
  appsecret: string
  token: string
  encoding_aes_key: string
  original_id?: string
  avatar_url?: string
  account_type?: string // direct, authorized
  component_appid?: string
  enabled: boolean
  is_default: boolean
  description?: string
  callback_url?: string
}

interface AppState {
  currentApp: WxAccount | null
  setCurrentApp: (app: WxAccount | null) => void
  theme: 'light' | 'dark'
  toggleTheme: () => void
}

export const useAppStore = create<AppState>((set, get) => ({
  currentApp: JSON.parse(localStorage.getItem('woaa_active_app') || 'null'),
  setCurrentApp: (app) => {
    localStorage.setItem('woaa_active_app', JSON.stringify(app))
    set({ currentApp: app })
  },
  theme: (localStorage.getItem('woaa_theme') as 'light' | 'dark') || 'light',
  toggleTheme: () => {
    const next = get().theme === 'light' ? 'dark' : 'light'
    localStorage.setItem('woaa_theme', next)
    if (next === 'dark') {
      document.documentElement.classList.add('dark')
    } else {
      document.documentElement.classList.remove('dark')
    }
    set({ theme: next })
  },
}))
