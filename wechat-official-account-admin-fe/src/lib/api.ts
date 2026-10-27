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
  const token = useAuthStore.getState().token
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
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
