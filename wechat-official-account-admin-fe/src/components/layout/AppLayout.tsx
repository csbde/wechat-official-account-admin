import React, { useEffect, useState } from 'react'
import {
  LayoutDashboard,
  Smartphone,
  AppWindow,
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
  Zap,
  BellRing,
  Building2,
  Menu,
  X,
} from 'lucide-react'
import { useAuthStore } from '@/stores/auth'
import { useAppStore, WxAccount } from '@/stores/app'
import { api } from '@/lib/api'
import { useQuery, useQueryClient } from '@tanstack/react-query'
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
import { useIsMobile } from '@/hooks/useIsMobile'

interface AppLayoutProps {
  currentTab: string
  onTabChange: (tab: string) => void
  children: React.ReactNode
}

export const AppLayout: React.FC<AppLayoutProps> = ({ currentTab, onTabChange, children }) => {
  const queryClient = useQueryClient()
  const { user, tenant, logout, activeTenantId, setActiveTenantId } = useAuthStore()
  const { currentApp, setCurrentApp, theme, toggleTheme } = useAppStore()
  const isSuperAdmin = user?.role === 'superadmin'
  const isMobile = useIsMobile()
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false)

  // 租户列表 (仅超级管理员需要，用于跨租户代运营视角切换)
  const { data: tenants } = useQuery<any[]>({
    queryKey: ['system-tenants'],
    queryFn: () => api.get('/api/system/tenants'),
    enabled: isSuperAdmin,
    retry: 0,
  })
  const viewTenant = activeTenantId ? tenants?.find((t) => t.id === activeTenantId) : null

  const switchTenantView = (id: number | null) => {
    if ((id ?? null) === (activeTenantId ?? null)) return
    setActiveTenantId(id)
    // 视角变更后上下文完全不同：清空已选公众号并刷新所有查询
    setCurrentApp(null)
    queryClient.invalidateQueries()
  }

  // 获取所有公众号列表
  const { data: accounts } = useQuery<WxAccount[]>({
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

  // 当屏幕尺寸发生改变切换回大屏时，自动重置手机菜单状态
  useEffect(() => {
    if (!isMobile) {
      setMobileMenuOpen(false)
    }
  }, [isMobile])

  const navItems = [
    { id: 'dashboard', label: '工作台概览', icon: LayoutDashboard },
    { id: 'accounts', label: '公众号与开放平台', icon: Smartphone },
    { id: 'miniprogram', label: '微信小程序', icon: AppWindow },
    { id: 'wxgate', label: '微信网关 (WXGate)', icon: Zap },
    { id: 'menu', label: '自定义菜单', icon: MenuSquare, requiresApp: true },
    { id: 'materials', label: '素材多媒体', icon: ImageIcon, requiresApp: true },
    { id: 'autoreply', label: '智能自动回复', icon: MessageSquareReply, requiresApp: true },
    { id: 'fans', label: '粉丝与标签', icon: Users, requiresApp: true },
    { id: 'qrcode', label: '带参二维码', icon: QrCode, requiresApp: true },
    { id: 'notify', label: '群发与模板消息', icon: BellRing, requiresApp: true },
    { id: 'logs', label: '请求与追踪日志', icon: FileText },
    { id: 'system', label: '系统与权限', icon: Settings, adminOnly: true },
  ].filter(item => !item.adminOnly || user?.role === 'admin' || user?.role === 'superadmin')

  const handleNavClick = (id: string) => {
    onTabChange(id)
    if (isMobile) {
      setMobileMenuOpen(false)
    }
  }

  return (
    <div className="flex min-h-screen bg-background text-foreground overflow-x-hidden">
      {/* 移动端遮罩层 */}
      {isMobile && mobileMenuOpen && (
        <div
          className="fixed inset-0 z-40 bg-black/60 backdrop-blur-sm transition-opacity duration-200 md:hidden"
          onClick={() => setMobileMenuOpen(false)}
          aria-hidden="true"
        />
      )}

      {/* 侧边栏：大屏固定左侧，小屏作为抽屉 */}
      <aside
        className={`
          fixed inset-y-0 left-0 z-50 w-72 max-w-[80vw] md:w-64 border-r border-border/70 bg-card/95 backdrop-blur-md flex flex-col shrink-0 select-none transition-transform duration-200 ease-in-out md:static md:translate-x-0
          ${mobileMenuOpen ? 'translate-x-0 shadow-2xl' : '-translate-x-full'}
        `}
      >
        {/* 平台 Logo 与名称 */}
        <div className="h-16 flex items-center justify-between px-5 md:px-6 border-b border-border/60 shrink-0">
          <div className="flex items-center gap-3">
            <div className="h-9 w-9 rounded-xl bg-gradient-to-tr from-emerald-600 to-green-500 flex items-center justify-center text-white shadow-md shadow-emerald-500/20 shrink-0">
              <Radio className="h-5 w-5" />
            </div>
            <div>
              <div className="font-bold text-base tracking-tight flex items-center gap-1.5">
                <span>WOAA Admin</span>
                <span className="text-[10px] font-semibold uppercase px-1.5 py-0.5 rounded bg-primary/10 text-primary">v2.0</span>
              </div>
              <div className="text-xs text-muted-foreground truncate max-w-[140px]">微信综合运营与网关平台</div>
            </div>
          </div>

          {/* 移动端关闭抽屉按钮 */}
          <Button
            variant="ghost"
            size="icon"
            onClick={() => setMobileMenuOpen(false)}
            className="md:hidden h-8 w-8 rounded-lg text-muted-foreground hover:text-foreground"
          >
            <X className="h-4 w-4" />
          </Button>
        </div>

        {/* 导航菜单 */}
        <nav className="flex-1 p-3 space-y-1 overflow-y-auto touch-scroll">
          {navItems.map((item) => {
            const Icon = item.icon
            const isActive = currentTab === item.id
            const disabled = item.requiresApp && !currentApp

            return (
              <button
                key={item.id}
                disabled={disabled}
                onClick={() => handleNavClick(item.id)}
                className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-all duration-150 min-h-[42px] ${
                  isActive
                    ? 'bg-primary text-primary-foreground shadow-sm shadow-primary/20 font-semibold'
                    : disabled
                    ? 'opacity-40 cursor-not-allowed text-muted-foreground'
                    : 'text-muted-foreground hover:bg-muted/80 hover:text-foreground'
                }`}
              >
                <Icon className={`h-4 w-4 shrink-0 ${isActive ? 'text-primary-foreground' : 'text-muted-foreground'}`} />
                <span className="flex-1 text-left truncate">{item.label}</span>
                {item.requiresApp && !currentApp && (
                  <span className="text-[10px] bg-muted px-1.5 py-0.5 rounded text-muted-foreground shrink-0">未选账号</span>
                )}
              </button>
            )
          })}
        </nav>

        {/* 底部系统状态 */}
        <div className="p-4 pb-safe border-t border-border/60 text-xs text-muted-foreground flex items-center justify-between shrink-0">
          <div className="flex items-center gap-2">
            <span className="h-2 w-2 rounded-full bg-emerald-500 animate-pulse"></span>
            <span>API & 网关运行中</span>
          </div>
          <span className="text-[11px] font-mono text-muted-foreground/80">Go + WXGate</span>
        </div>
      </aside>

      {/* 右侧主工作区 */}
      <div className="flex-1 flex flex-col min-w-0 overflow-x-hidden">
        {/* 顶部 Header */}
        <header className="h-16 border-b border-border/60 bg-card/80 backdrop-blur-md px-3 sm:px-6 flex items-center justify-between sticky top-0 z-30 shrink-0">
          <div className="flex items-center gap-2 sm:gap-3 min-w-0">
            {/* 移动端汉堡菜单触发按钮 */}
            <Button
              variant="ghost"
              size="icon"
              onClick={() => setMobileMenuOpen(true)}
              className="md:hidden h-9 w-9 rounded-lg shrink-0 -ml-1 text-foreground"
              aria-label="打开侧边导航菜单"
            >
              <Menu className="h-5 w-5" />
            </Button>

            {/* 超级管理员：租户视角切换器 */}
            {isSuperAdmin && (tenants?.length ?? 0) > 0 && (
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button
                    variant="outline"
                    className={`h-9 sm:h-10 px-2 sm:px-3 gap-1.5 sm:gap-2 text-left border shrink-0 ${
                      viewTenant
                        ? 'border-amber-500/50 bg-amber-500/10 hover:bg-amber-500/15'
                        : 'border-border bg-background'
                    }`}
                  >
                    <Building2
                      className={`h-4 w-4 shrink-0 ${viewTenant ? 'text-amber-600' : 'text-primary'}`}
                    />
                    <div className="flex flex-col text-left hidden sm:block">
                      <span className="text-[10px] text-muted-foreground leading-none">
                        {viewTenant ? '租户视角' : isSuperAdmin ? '平台全局视角' : '我的租户'}
                      </span>
                      <span className="text-xs font-semibold leading-tight mt-0.5 max-w-[100px] md:max-w-[120px] truncate">
                        {viewTenant
                          ? viewTenant.name
                          : isSuperAdmin
                            ? '全部租户'
                            : tenant?.name || '默认组织'}
                      </span>
                    </div>
                    <ChevronDown className="h-3.5 w-3.5 opacity-60 ml-0.5" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="start" className="w-72">
                  <DropdownMenuLabel className="text-xs text-muted-foreground">
                    切换租户视角 (数据完全隔离)
                  </DropdownMenuLabel>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem
                    onClick={() => switchTenantView(null)}
                    className={`cursor-pointer py-2 gap-2 ${!activeTenantId ? 'bg-primary/10 font-semibold text-primary' : ''}`}
                  >
                    <ShieldAlert className="h-4 w-4 shrink-0 opacity-70" />
                    <div className="flex flex-col truncate pr-2">
                      <span className="truncate text-sm">平台全局视角 (跨租户可见)</span>
                      <span className="text-[10px] text-muted-foreground">
                        仅超级管理员可用；新建数据仍落入你自己的租户
                      </span>
                    </div>
                  </DropdownMenuItem>
                  <DropdownMenuSeparator />
                  {tenants!.filter((t) => t.is_home).map((t) => (
                    <DropdownMenuItem
                      key={`home-${t.id}`}
                      onClick={() => switchTenantView(t.id)}
                      className={`cursor-pointer py-2 ${activeTenantId === t.id ? 'bg-primary/10 font-semibold text-primary' : ''}`}
                    >
                      <div className="flex flex-col truncate pr-2">
                        <span className="truncate text-sm">{t.name} (本租户 · 严格锁定)</span>
                        <span className="text-[10px] text-muted-foreground font-mono">
                          {t.code} · {t.account_count} 个账号 / {t.user_count} 个用户
                        </span>
                      </div>
                    </DropdownMenuItem>
                  ))}
                  {tenants!
                    .filter((t) => !t.is_home)
                    .map((t) => (
                      <DropdownMenuItem
                        key={t.id}
                        disabled={t.status !== 1}
                        onClick={() => switchTenantView(t.id)}
                        className={`cursor-pointer py-2 ${
                          activeTenantId === t.id ? 'bg-amber-500/15 font-semibold text-amber-700' : ''
                        }`}
                      >
                        <div className="flex flex-col truncate pr-2">
                          <span className="truncate text-sm flex items-center gap-1.5">
                            {t.name}
                            {t.status !== 1 && (
                              <Badge variant="destructive" className="text-[9px]">
                                停用
                              </Badge>
                            )}
                          </span>
                          <span className="text-[10px] text-muted-foreground font-mono">
                            {t.code} · {t.account_count} 个账号 / {t.user_count} 个用户
                          </span>
                        </div>
                      </DropdownMenuItem>
                    ))}
                  <DropdownMenuSeparator />
                  <DropdownMenuItem
                    onClick={() => handleNavClick('system')}
                    className="cursor-pointer gap-2 text-primary font-medium"
                  >
                    <Building2 className="h-4 w-4" />
                    <span>管理租户与组织</span>
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            )}

            {/* 当前选中的公众号 Switcher */}
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="outline" className="h-9 sm:h-10 px-2 sm:px-3.5 border-dashed border-primary/40 bg-primary/5 hover:bg-primary/10 gap-1.5 sm:gap-2.5 text-left shrink min-w-0">
                  <div className="h-6 w-6 rounded-lg bg-emerald-500/20 text-emerald-600 dark:text-emerald-400 flex items-center justify-center font-bold text-xs shrink-0">
                    {currentApp ? currentApp.name.slice(0, 1) : '?'}
                  </div>
                  <div className="flex flex-col text-left min-w-0">
                    <span className="text-xs font-semibold max-w-[85px] sm:max-w-[140px] md:max-w-[160px] truncate leading-tight">
                      {currentApp ? currentApp.name : '未选择'}
                    </span>
                    <span className="text-[10px] text-muted-foreground font-mono leading-none hidden sm:block truncate max-w-[140px]">
                      {currentApp ? currentApp.appid : '请点击选择'}
                    </span>
                  </div>
                  <ChevronDown className="h-3.5 w-3.5 opacity-60 shrink-0" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="start" className="w-64 sm:w-72">
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
                        {(acc as any).tenant_name && (acc as any).tenant_id !== user?.tenant_id && (
                          <span className="text-[10px] text-amber-600 dark:text-amber-400 truncate">
                            归属：{(acc as any).tenant_name}
                          </span>
                        )}
                      </div>
                      {acc.is_default && <Badge variant="outline" className="text-[10px] shrink-0">默认</Badge>}
                    </DropdownMenuItem>
                  ))
                ) : (
                  <div className="p-3 text-xs text-center text-muted-foreground">暂未配置公众号</div>
                )}
                <DropdownMenuSeparator />
                <DropdownMenuItem onClick={() => handleNavClick('accounts')} className="cursor-pointer gap-2 text-primary font-medium">
                  <Plus className="h-4 w-4" />
                  <span>管理 / 添加新公众号</span>
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>

            {isSuperAdmin && !activeTenantId && (
              <div className="hidden xl:flex items-center gap-1.5 text-xs text-sky-700 dark:text-sky-300 bg-sky-500/10 px-3 py-1.5 rounded-lg border border-sky-500/25 shrink-0">
                <ShieldAlert className="h-4 w-4 shrink-0" />
                <span>平台全局视角：可查看所有租户的公众号与数据</span>
              </div>
            )}

            {viewTenant && (
              <div className="hidden lg:flex items-center gap-1.5 text-xs text-amber-700 dark:text-amber-300 bg-amber-500/10 px-3 py-1.5 rounded-lg border border-amber-500/25 shrink-0">
                <ShieldAlert className="h-4 w-4 shrink-0" />
                <span>
                  以租户「{viewTenant.name}」视角管理
                </span>
                <button
                  onClick={() => switchTenantView(null)}
                  className="underline font-semibold hover:text-amber-900 ml-1"
                >
                  退出视角
                </button>
              </div>
            )}

            {!currentApp && (
              <div className="hidden md:flex items-center gap-1.5 text-xs text-amber-600 dark:text-amber-400 bg-amber-500/10 px-2.5 py-1 rounded-lg border border-amber-500/20 shrink-0">
                <ShieldAlert className="h-3.5 w-3.5 shrink-0" />
                <span className="truncate max-w-[200px]">请先选择公众号</span>
              </div>
            )}
          </div>

          {/* 顶部右侧：主题切换与用户菜单 */}
          <div className="flex items-center gap-1.5 sm:gap-2.5 shrink-0">
            {/* 主题切换 */}
            <Button variant="ghost" size="icon" onClick={toggleTheme} className="rounded-lg h-9 w-9">
              {theme === 'dark' ? <Sun className="h-4 w-4 text-amber-400" /> : <Moon className="h-4 w-4 text-slate-700" />}
            </Button>

            {/* 用户菜单 */}
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="ghost" className="h-9 px-2 sm:px-2.5 gap-1.5 sm:gap-2 rounded-lg">
                  <div className="h-7 w-7 rounded-full bg-primary/20 text-primary flex items-center justify-center font-bold text-xs uppercase shrink-0">
                    {user?.nickname ? user.nickname.slice(0, 1) : user?.username?.slice(0, 1) || 'U'}
                  </div>
                  <div className="text-left hidden sm:block">
                    <div className="text-xs font-medium leading-none">{user?.nickname || user?.username}</div>
                    <div className="text-[10px] text-muted-foreground capitalize leading-tight mt-0.5">{user?.role} · {tenant?.name || '默认组织'}</div>
                  </div>
                  <ChevronDown className="h-3 w-3 opacity-50 shrink-0" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-48">
                <DropdownMenuLabel>
                  <div className="font-semibold text-sm">{user?.nickname || user?.username}</div>
                  <div className="text-xs text-muted-foreground font-normal">角色: {user?.role}</div>
                </DropdownMenuLabel>
                <DropdownMenuSeparator />
                <DropdownMenuItem onClick={() => handleNavClick('system')} className="cursor-pointer gap-2">
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
        <main className="flex-1 p-3 sm:p-4 md:p-6 pb-safe overflow-y-auto touch-scroll">
          {children}
        </main>
      </div>
    </div>
  )
}
