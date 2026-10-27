import React, { useEffect } from 'react'
import {
  LayoutDashboard,
  Smartphone,
  MenuSquare,
  Image as ImageIcon,
  MessageSquareReply,
  Users,
  QrCode,
  FileText,
  Settings,
  LogOut,
  Sun,
  Moon,
  ChevronDown,
  Plus,
  Radio,
  ShieldAlert,
} from 'lucide-react'
import { useAuthStore } from '@/stores/auth'
import { useAppStore, WxAccount } from '@/stores/app'
import { api } from '@/lib/api'
import { useQuery } from '@tanstack/react-query'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'

interface AppLayoutProps {
  currentTab: string
  onTabChange: (tab: string) => void
  children: React.ReactNode
}

export const AppLayout: React.FC<AppLayoutProps> = ({ currentTab, onTabChange, children }) => {
  const { user, logout } = useAuthStore()
  const { currentApp, setCurrentApp, theme, toggleTheme } = useAppStore()

  // 获取所有公众号列表
  const { data: accounts, refetch: refetchAccounts } = useQuery<WxAccount[]>({
    queryKey: ['accounts'],
    queryFn: () => api.get('/api/accounts'),
  })

  // 如果没有当前公众号且有公众号列表，自动选择默认公众号
  useEffect(() => {
    if (!currentApp && accounts && accounts.length > 0) {
      const def = accounts.find((a) => a.is_default) || accounts[0]
      setCurrentApp(def)
    }
  }, [accounts, currentApp, setCurrentApp])

  const navItems = [
    { id: 'dashboard', label: '工作台概览', icon: LayoutDashboard },
    { id: 'accounts', label: '公众号管理', icon: Smartphone },
    { id: 'menu', label: '自定义菜单', icon: MenuSquare, requiresApp: true },
    { id: 'materials', label: '素材多媒体', icon: ImageIcon, requiresApp: true },
    { id: 'autoreply', label: '智能自动回复', icon: MessageSquareReply, requiresApp: true },
    { id: 'fans', label: '粉丝与标签', icon: Users, requiresApp: true },
    { id: 'qrcode', label: '带参二维码', icon: QrCode, requiresApp: true },
    { id: 'logs', label: '请求与追踪日志', icon: FileText },
    { id: 'system', label: '系统与权限', icon: Settings },
  ]

  return (
    <div className="flex min-h-screen bg-background text-foreground">
      {/* 左侧侧边栏 */}
      <aside className="w-64 border-r border-border/70 bg-card/60 backdrop-blur-md flex flex-col shrink-0 select-none">
        {/* 平台 Logo 与名称 */}
        <div className="h-16 flex items-center px-6 gap-3 border-b border-border/60">
          <div className="h-9 w-9 rounded-xl bg-gradient-to-tr from-emerald-600 to-green-500 flex items-center justify-center text-white shadow-md shadow-emerald-500/20">
            <Radio className="h-5 w-5" />
          </div>
          <div>
            <div className="font-bold text-base tracking-tight flex items-center gap-1.5">
              <span>WOAA Admin</span>
              <span className="text-[10px] font-semibold uppercase px-1.5 py-0.5 rounded bg-primary/10 text-primary">v2.0</span>
            </div>
            <div className="text-xs text-muted-foreground">微信公众号管理平台</div>
          </div>
        </div>

        {/* 导航菜单 */}
        <nav className="flex-1 p-3 space-y-1 overflow-y-auto">
          {navItems.map((item) => {
            const Icon = item.icon
            const isActive = currentTab === item.id
            const disabled = item.requiresApp && !currentApp

            return (
              <button
                key={item.id}
                disabled={disabled}
                onClick={() => onTabChange(item.id)}
                className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-all duration-150 ${
                  isActive
                    ? 'bg-primary text-primary-foreground shadow-sm shadow-primary/20 font-semibold'
                    : disabled
                    ? 'opacity-40 cursor-not-allowed text-muted-foreground'
                    : 'text-muted-foreground hover:bg-muted/80 hover:text-foreground'
                }`}
              >
                <Icon className={`h-4 w-4 ${isActive ? 'text-primary-foreground' : 'text-muted-foreground'}`} />
                <span className="flex-1 text-left">{item.label}</span>
                {item.requiresApp && !currentApp && (
                  <span className="text-[10px] bg-muted px-1.5 py-0.5 rounded text-muted-foreground">未选账号</span>
                )}
              </button>
            )
          })}
        </nav>

        {/* 底部系统状态 */}
        <div className="p-4 border-t border-border/60 text-xs text-muted-foreground flex items-center justify-between">
          <div className="flex items-center gap-2">
            <span className="h-2 w-2 rounded-full bg-emerald-500 animate-pulse"></span>
            <span>API 服务正常</span>
          </div>
          <span className="text-[11px] font-mono text-muted-foreground/80">Go + React</span>
        </div>
      </aside>

      {/* 右侧主工作区 */}
      <div className="flex-1 flex flex-col min-w-0">
        {/* 顶部 Header */}
        <header className="h-16 border-b border-border/60 bg-card/60 backdrop-blur-md px-6 flex items-center justify-between sticky top-0 z-30">
          {/* 当前选中的公众号 Switcher */}
          <div className="flex items-center gap-3">
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="outline" className="h-10 px-3.5 border-dashed border-primary/40 bg-primary/5 hover:bg-primary/10 gap-2.5 text-left">
                  <div className="h-6 w-6 rounded-lg bg-emerald-500/20 text-emerald-600 dark:text-emerald-400 flex items-center justify-center font-bold text-xs">
                    {currentApp ? currentApp.name.slice(0, 1) : '?'}
                  </div>
                  <div className="flex flex-col text-left">
                    <span className="text-xs font-semibold max-w-[160px] truncate leading-tight">
                      {currentApp ? currentApp.name : '未选择公众号'}
                    </span>
                    <span className="text-[10px] text-muted-foreground font-mono leading-none">
                      {currentApp ? currentApp.appid : '请点击下拉选择'}
                    </span>
                  </div>
                  <ChevronDown className="h-3.5 w-3.5 opacity-60 ml-1" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="start" className="w-64">
                <DropdownMenuLabel className="text-xs text-muted-foreground">切换公众号账号</DropdownMenuLabel>
                <DropdownMenuSeparator />
                {accounts && accounts.length > 0 ? (
                  accounts.map((acc) => (
                    <DropdownMenuItem
                      key={acc.id}
                      onClick={() => setCurrentApp(acc)}
                      className={`flex items-center justify-between cursor-pointer py-2 ${
                        currentApp?.id === acc.id ? 'bg-primary/10 font-semibold text-primary' : ''
                      }`}
                    >
                      <div className="flex flex-col truncate pr-2">
                        <span className="truncate text-sm">{acc.name}</span>
                        <span className="text-[10px] text-muted-foreground font-mono">{acc.appid}</span>
                      </div>
                      {acc.is_default && <Badge variant="outline" className="text-[10px] shrink-0">默认</Badge>}
                    </DropdownMenuItem>
                  ))
                ) : (
                  <div className="p-3 text-xs text-center text-muted-foreground">暂未配置公众号</div>
                )}
                <DropdownMenuSeparator />
                <DropdownMenuItem onClick={() => onTabChange('accounts')} className="cursor-pointer gap-2 text-primary font-medium">
                  <Plus className="h-4 w-4" />
                  <span>管理 / 添加新公众号</span>
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>

            {!currentApp && (
              <div className="hidden md:flex items-center gap-1.5 text-xs text-amber-600 dark:text-amber-400 bg-amber-500/10 px-3 py-1.5 rounded-lg border border-amber-500/20">
                <ShieldAlert className="h-4 w-4" />
                <span>请先选择或创建一个微信公众号以解锁菜单、素材与粉丝等功能</span>
              </div>
            )}
          </div>

          {/* 顶部右侧：主题切换与用户菜单 */}
          <div className="flex items-center gap-3">
            {/* 主题切换 */}
            <Button variant="ghost" size="icon" onClick={toggleTheme} className="rounded-lg h-9 w-9">
              {theme === 'dark' ? <Sun className="h-4 w-4 text-amber-400" /> : <Moon className="h-4 w-4 text-slate-700" />}
            </Button>

            {/* 用户菜单 */}
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="ghost" className="h-9 px-2.5 gap-2 rounded-lg">
                  <div className="h-7 w-7 rounded-full bg-primary/20 text-primary flex items-center justify-center font-bold text-xs uppercase">
                    {user?.nickname ? user.nickname.slice(0, 1) : user?.username?.slice(0, 1) || 'U'}
                  </div>
                  <div className="text-left hidden sm:block">
                    <div className="text-xs font-medium leading-none">{user?.nickname || user?.username}</div>
                    <div className="text-[10px] text-muted-foreground capitalize leading-tight mt-0.5">{user?.role}</div>
                  </div>
                  <ChevronDown className="h-3 w-3 opacity-50" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-48">
                <DropdownMenuLabel>
                  <div className="font-semibold text-sm">{user?.nickname || user?.username}</div>
                  <div className="text-xs text-muted-foreground font-normal">角色: {user?.role}</div>
                </DropdownMenuLabel>
                <DropdownMenuSeparator />
                <DropdownMenuItem onClick={() => onTabChange('system')} className="cursor-pointer gap-2">
                  <Settings className="h-4 w-4" />
                  <span>系统设置</span>
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem onClick={logout} className="cursor-pointer gap-2 text-destructive focus:text-destructive">
                  <LogOut className="h-4 w-4" />
                  <span>退出登录</span>
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </header>

        {/* 核心工作区主页面 */}
        <main className="flex-1 p-6 overflow-y-auto">
          {children}
        </main>
      </div>
    </div>
  )
}
