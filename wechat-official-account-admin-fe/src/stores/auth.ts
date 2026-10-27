import { create } from 'zustand'

export interface User {
  id: number
  username: string
  nickname: string
  avatar?: string
  role: string
  status: number
  sso_provider?: string
}

interface AuthState {
  token: string | null
  user: User | null
  setAuth: (token: string, user: User) => void
  setUser: (user: User) => void
  logout: () => void
}

export const useAuthStore = create<AuthState>((set) => ({
  token: localStorage.getItem('woaa_token'),
  user: JSON.parse(localStorage.getItem('woaa_user') || 'null'),
  setAuth: (token, user) => {
    localStorage.setItem('woaa_token', token)
    localStorage.setItem('woaa_user', JSON.stringify(user))
    set({ token, user })
  },
  setUser: (user) => {
    localStorage.setItem('woaa_user', JSON.stringify(user))
    set({ user })
  },
  logout: () => {
    localStorage.removeItem('woaa_token')
    localStorage.removeItem('woaa_user')
    set({ token: null, user: null })
  },
}))
