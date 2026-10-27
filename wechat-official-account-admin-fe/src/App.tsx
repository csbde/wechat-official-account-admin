import React, { useState } from 'react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { Toaster } from 'sonner'
import { useAuthStore } from '@/stores/auth'
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
  const { token } = useAuthStore()
  const [currentTab, setCurrentTab] = useState('dashboard')

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
