import React from 'react'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import {
  Smartphone,
  Users,
  Image as ImageIcon,
  MessageSquareReply,
  Activity,
  ArrowUpRight,
  ShieldCheck,
  Zap,
} from 'lucide-react'
import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api'
import { useAppStore } from '@/stores/app'

interface DashboardProps {
  onNavigate: (tab: string) => void
}

export const DashboardPage: React.FC<DashboardProps> = ({ onNavigate }) => {
  const { currentApp } = useAppStore()

  // 获取系统健康与基础统计
  const { data: health } = useQuery<any>({
    queryKey: ['system-health'],
    queryFn: () => api.get('/api/system/health'),
  })

  const { data: publicIP } = useQuery<any>({
    queryKey: ['system-ip'],
    queryFn: () => api.get('/api/system/public-ip'),
  })

  const { data: accounts } = useQuery<any[]>({
    queryKey: ['accounts'],
    queryFn: () => api.get('/api/accounts'),
  })

  const { data: fansData } = useQuery<any>({
    queryKey: ['fans-count', currentApp?.appid],
    queryFn: () => api.get('/api/fans?page=1&page_size=1'),
    enabled: !!currentApp,
  })

  const { data: materialsData } = useQuery<any>({
    queryKey: ['materials-count', currentApp?.appid],
    queryFn: () => api.get('/api/materials?page=1&page_size=1'),
    enabled: !!currentApp,
  })

  return (
    <div className="space-y-6">
      {/* 顶部欢迎卡片 */}
      <div className="relative overflow-hidden rounded-2xl bg-gradient-to-r from-emerald-600 via-emerald-700 to-teal-800 p-8 text-white shadow-lg shadow-emerald-900/10">
        <div className="relative z-10 max-w-2xl space-y-2">
          <div className="inline-flex items-center gap-1.5 px-3 py-1 rounded-full bg-white/15 backdrop-blur-md text-xs font-semibold uppercase tracking-wider text-emerald-100">
            <Zap className="h-3.5 w-3.5 text-amber-300" />
            <span>全新重构架构已就绪</span>
          </div>
          <h1 className="text-3xl font-extrabold tracking-tight sm:text-4xl">
            微信公众号综合管理平台
          </h1>
          <p className="text-emerald-100/90 text-sm sm:text-base leading-relaxed">
            基于 Go 1.23 + Gin + GORM + PostgreSQL + Redis + Asynq 与 React 18 + TypeScript + Vite + Tailwind CSS + shadcn/ui 全栈重构，提供极致响应与可靠的多公众号运营能力。
          </p>
          <div className="pt-2 flex flex-wrap gap-3">
            <Button
              onClick={() => onNavigate('accounts')}
              variant="secondary"
              className="font-semibold shadow-md bg-white text-emerald-800 hover:bg-emerald-50"
            >
              <Smartphone className="h-4 w-4 mr-1.5" />
              管理微信公众号
            </Button>
            <Button
              onClick={() => onNavigate('menu')}
              variant="outline"
              className="border-white/30 text-white hover:bg-white/10"
            >
              设计微信自定义菜单
              <ArrowUpRight className="h-4 w-4 ml-1" />
            </Button>
          </div>
        </div>
      </div>

      {/* 核心指标统计 */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">配置公众号</CardTitle>
            <Smartphone className="h-4 w-4 text-emerald-600 dark:text-emerald-400" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{accounts?.length || 0}</div>
            <p className="text-xs text-muted-foreground mt-1 flex items-center gap-1">
              当前活跃: <span className="font-semibold text-foreground">{currentApp ? currentApp.name : '未选择'}</span>
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">关注者总数</CardTitle>
            <Users className="h-4 w-4 text-blue-600 dark:text-blue-400" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{fansData?.pagination?.total ?? 0}</div>
            <p className="text-xs text-muted-foreground mt-1">
              支持异步后台队列批量同步
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">多媒体素材</CardTitle>
            <ImageIcon className="h-4 w-4 text-purple-600 dark:text-purple-400" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{materialsData?.pagination?.total ?? 0}</div>
            <p className="text-xs text-muted-foreground mt-1">永久素材与图片缓存</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">系统健康状态</CardTitle>
            <Activity className="h-4 w-4 text-emerald-600" />
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-2">
              <span className="text-2xl font-bold capitalize">{health?.status || 'Active'}</span>
              <Badge variant="success" className="text-[10px]">ALL UP</Badge>
            </div>
            <p className="text-xs text-muted-foreground mt-1 font-mono">
              出口IP: {publicIP?.ip || '127.0.0.1'}
            </p>
          </CardContent>
        </Card>
      </div>

      {/* 快捷配置指引与服务连通 */}
      <div className="grid gap-6 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle className="text-base flex items-center gap-2">
              <ShieldCheck className="h-5 w-5 text-emerald-600" />
              <span>微信公众号接入参数</span>
            </CardTitle>
            <CardDescription>配置至微信公众平台后台 (mp.weixin.qq.com) -&gt; 基本设置</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3 text-sm">
            {currentApp ? (
              <>
                <div className="p-3 rounded-lg bg-muted/60 space-y-1">
                  <div className="text-xs text-muted-foreground">服务器地址 (URL):</div>
                  <div className="font-mono text-xs font-semibold select-all break-all text-primary">
                    {currentApp.callback_url || `http://<your-host>/wxmp/${currentApp.appid}/handler`}
                  </div>
                </div>
                <div className="grid grid-cols-2 gap-3">
                  <div className="p-3 rounded-lg bg-muted/60 space-y-1">
                    <div className="text-xs text-muted-foreground">令牌 (Token):</div>
                    <div className="font-mono text-xs font-semibold select-all">{currentApp.token || '未配置'}</div>
                  </div>
                  <div className="p-3 rounded-lg bg-muted/60 space-y-1">
                    <div className="text-xs text-muted-foreground">消息加解密方式:</div>
                    <div className="text-xs font-medium">兼容 / 明文模式</div>
                  </div>
                </div>
              </>
            ) : (
              <div className="p-4 text-center text-muted-foreground text-sm">请先选择一个公众号查看微信回调配置</div>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="text-base">后端技术栈监控与集成</CardTitle>
            <CardDescription>各组件运行指标与状态实时监测</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="flex items-center justify-between text-sm py-1.5 border-b">
              <span className="text-muted-foreground">PostgreSQL (GORM) 主数据库</span>
              <Badge variant={health?.database === 'up' ? 'success' : 'destructive'}>
                {health?.database || 'up'}
              </Badge>
            </div>
            <div className="flex items-center justify-between text-sm py-1.5 border-b">
              <span className="text-muted-foreground">Redis 高速缓存服务</span>
              <Badge variant={health?.redis === 'up' ? 'success' : 'destructive'}>
                {health?.redis || 'up'}
              </Badge>
            </div>
            <div className="flex items-center justify-between text-sm py-1.5 border-b">
              <span className="text-muted-foreground">Asynq 分布式后台任务队列</span>
              <Badge variant={health?.asynq === 'up' ? 'success' : 'destructive'}>
                {health?.asynq || 'up'}
              </Badge>
            </div>
            <div className="flex items-center justify-between text-sm py-1.5">
              <span className="text-muted-foreground">OpenTelemetry 链路追踪与 OpenAPI</span>
              <Badge variant="outline" className="font-mono text-[10px]">Active</Badge>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
