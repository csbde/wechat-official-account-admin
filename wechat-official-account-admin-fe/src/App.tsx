import React, { useState, useEffect } from 'react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { Toaster, toast } from 'sonner'
import { useAuthStore } from '@/stores/auth'
import { api } from '@/lib/api'
import { AppLayout } from '@/components/layout/AppLayout'
import { LoginPage } from '@/pages/login/LoginPage'
import { DashboardPage } from '@/pages/dashboard/DashboardPage'
import { AccountsPage } from '@/pages/accounts/AccountsPage'
import { WxGatePage } from '@/pages/wxgate/WxGatePage'
import { MenuDesignerPage } from '@/pages/menu/MenuDesignerPage'
import { MaterialsPage } from '@/pages/materials/MaterialsPage'
import { AutoReplyPage } from '@/pages/autoreply/AutoReplyPage'
import { FansPage } from '@/pages/fans/FansPage'
import { QrcodePage } from '@/pages/qrcode/QrcodePage'
import { LogsPage } from '@/pages/logs/LogsPage'
import { SystemPage } from '@/pages/system/SystemPage'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchOnWindowFocus: false,
      retry: 1,
    },
  },
})

export const App: React.FC = () => {
  const { token, setAuth } = useAuthStore()
  const [currentTab, setCurrentTab] = useState('dashboard')

  // 检测 URL Hash 中是否携带 OIDC 返回的 Token
  useEffect(() => {
    const hash = window.location.hash
    if (hash && hash.includes('token=')) {
      const params = new URLSearchParams(hash.replace('#', ''))
      const oidcToken = params.get('token')
      if (oidcToken) {
        localStorage.setItem('woaa_token', oidcToken)
        window.history.replaceState(null, '', window.location.pathname)
        // 拉取用户信息
        api
          .get('/api/auth/userinfo', {
            headers: { Authorization: `Bearer ${oidcToken}` },
          })
          .then((res: any) => {
            if (res.user) {
              setAuth(oidcToken, res.user, res.tenant)
              toast.success(`OIDC 登录成功！已自动绑定用户: ${res.user.nickname || res.user.username}`)
            }
          })
          .catch((e) => {
            console.error(e)
          })
      }
    }
  }, [setAuth])

  // 若未登录，显示登录页
  if (!token) {
    return (
      <QueryClientProvider client={queryClient}>
        <LoginPage />
        <Toaster richColors position="top-center" />
      </QueryClientProvider>
    )
  }

  // 渲染对应的视图组件
  const renderContent = () => {
    switch (currentTab) {
      case 'dashboard':
        return <DashboardPage onNavigate={setCurrentTab} />
      case 'accounts':
        return <AccountsPage />
      case 'wxgate':
        return <WxGatePage />
      case 'menu':
        return <MenuDesignerPage />
      case 'materials':
        return <MaterialsPage />
      case 'autoreply':
        return <AutoReplyPage />
      case 'fans':
        return <FansPage />
      case 'qrcode':
        return <QrcodePage />
      case 'logs':
        return <LogsPage />
      case 'system':
        return <SystemPage />
      default:
        return <DashboardPage onNavigate={setCurrentTab} />
    }
  }

  return (
    <QueryClientProvider client={queryClient}>
      <AppLayout currentTab={currentTab} onTabChange={setCurrentTab}>
        {renderContent()}
      </AppLayout>
      <Toaster richColors position="top-right" />
    </QueryClientProvider>
  )
}
