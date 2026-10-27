import { create } from 'zustand'

export interface Tenant {
  id: number
  name: string
  code: string
}

export interface User {
  id: number
  tenant_id: number
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
  tenant: Tenant | null
  // 超级管理员的“租户视角”：null 表示本租户；非空表示代管的其他租户 ID
  activeTenantId: number | null
  setAuth: (token: string, user: User, tenant?: Tenant) => void
  setUser: (user: User) => void
  setTenant: (tenant: Tenant) => void
  setActiveTenantId: (id: number | null) => void
  logout: () => void
}

export const useAuthStore = create<AuthState>((set) => ({
  token: localStorage.getItem('woaa_token'),
  user: JSON.parse(localStorage.getItem('woaa_user') || 'null'),
  tenant: JSON.parse(localStorage.getItem('woaa_tenant') || 'null'),
  activeTenantId: (() => {
    const raw = localStorage.getItem('woaa_active_tenant')
    return raw ? Number(raw) : null
  })(),
  setAuth: (token, user, tenant) => {
    localStorage.setItem('woaa_token', token)
    localStorage.setItem('woaa_user', JSON.stringify(user))
    if (tenant) {
      localStorage.setItem('woaa_tenant', JSON.stringify(tenant))
    }
    set({ token, user, tenant: tenant || null, activeTenantId: null })
  },
  setUser: (user) => {
    localStorage.setItem('woaa_user', JSON.stringify(user))
    set({ user })
  },
  setTenant: (tenant) => {
    localStorage.setItem('woaa_tenant', JSON.stringify(tenant))
    set({ tenant })
  },
  setActiveTenantId: (id) => {
    if (id && id > 0) {
      localStorage.setItem('woaa_active_tenant', String(id))
    } else {
      localStorage.removeItem('woaa_active_tenant')
    }
    set({ activeTenantId: id && id > 0 ? id : null })
  },
  logout: () => {
    localStorage.removeItem('woaa_token')
    localStorage.removeItem('woaa_user')
    localStorage.removeItem('woaa_tenant')
    localStorage.removeItem('woaa_active_tenant')
    set({ token: null, user: null, tenant: null, activeTenantId: null })
  },
}))
