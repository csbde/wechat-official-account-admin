import axios from 'axios'
import { toast } from 'sonner'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'

export interface ApiResponse<T = any> {
  code: number
  message: string
  data: T
}

export interface PageResult<T = any> {
  list: T[]
  pagination: {
    page: number
    page_size: number
    total: number
  }
}

export const api = axios.create({
  baseURL: '',
  timeout: 15000,
})

// 请求拦截器
api.interceptors.request.use((config) => {
  const { token, user, activeTenantId } = useAuthStore.getState()
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }

  // 超级管理员代管其他租户时携带视角标识，后端会二次校验该租户存在且已启用
  if (activeTenantId && user && activeTenantId !== user.tenant_id) {
    config.headers['X-Tenant-ID'] = String(activeTenantId)
  }

  const currentApp = useAppStore.getState().currentApp
  if (currentApp?.appid) {
    config.headers['X-App-ID'] = currentApp.appid
  }

  return config
})

// 响应拦截器
api.interceptors.response.use(
  (response) => {
    const res = response.data
    // 如果返回的是二进制或非标准格式
    if (response.config.responseType === 'blob' || !res.code && res.code !== 0) {
      return res
    }

    // 后端在租户视角不可用时会回写 X-Tenant-Fallback，前端据此自动纠正，避免卡在失效视角
    const fallback = response.headers?.['x-tenant-fallback']
    if (fallback) {
      const id = Number(fallback)
      const st = useAuthStore.getState()
      if (st.activeTenantId && id > 0 && st.activeTenantId !== id) {
        st.setActiveTenantId(id === st.user?.tenant_id ? null : id)
        toast.error(`目标租户视角不可用，已自动切换回${id === st.user?.tenant_id ? '本租户' : '租户 #' + id}`)
      }
    }

    if (res.code !== 0) {
      toast.error(res.message || '请求处理失败')
      return Promise.reject(new Error(res.message || 'Error'))
    }
    return res.data
  },
  (error) => {
    if (error.response?.status === 401) {
      useAuthStore.getState().logout()
      toast.error('登录状态已失效，请重新登录')
    } else {
      const msg = error.response?.data?.message || error.message || '网络请求错误'
      toast.error(msg)
    }
    return Promise.reject(error)
  }
)
