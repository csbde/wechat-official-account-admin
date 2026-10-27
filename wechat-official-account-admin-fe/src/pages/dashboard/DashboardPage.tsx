import React from 'react'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import {
  Smartphone,
  AppWindow,
  Users,
  Image as ImageIcon,
  Activity,
  ArrowUpRight,
  ShieldCheck,
  Zap,
  RefreshCw,
  CheckCircle2,
  AlertTriangle,
  XCircle,
  HelpCircle,
  Wifi,
} from 'lucide-react'
import { useQuery } from '@tanstack/react-query'
import { api } from '@/lib/api'
import { useAppStore } from '@/stores/app'

interface DashboardProps {
  onNavigate: (tab: string) => void
}

export const DashboardPage: React.FC<DashboardProps> = ({ onNavigate }) => {
  const { currentApp } = useAppStore()

  // 获取服务连通性与连接健康监控
  const { data: healthData, refetch: refetchHealth, isFetching: checkingHealth } = useQuery<any>({
    queryKey: ['system-health'],
    queryFn: () => api.get('/api/system/health'),
    refetchInterval: 30000,
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

  const components = healthData?.components || []
  const overallHealthy = healthData?.overall_status === 'healthy'

  return (
    <div className="space-y-6">
      {/* 顶部欢迎卡片 */}
      <div className="relative overflow-hidden rounded-2xl bg-gradient-to-r from-emerald-600 via-emerald-700 to-teal-800 p-8 text-white shadow-lg shadow-emerald-900/10">
        <div className="relative z-10 max-w-2xl space-y-2">
          <div className="inline-flex items-center gap-1.5 px-3 py-1 rounded-full bg-white/15 backdrop-blur-md text-xs font-semibold uppercase tracking-wider text-emerald-100">
            <Zap className="h-3.5 w-3.5 text-amber-300" />
            <span>Go 1.26+ & 多租户数据隔离</span>
          </div>
          <h1 className="text-3xl font-extrabold tracking-tight sm:text-4xl">
            微信公众号综合运营与网关平台
          </h1>
          <p className="text-emerald-100/90 text-sm sm:text-base leading-relaxed">
            内置统一高并发凭证中心与授权网关 (WXGate)，支持微信开放平台第三方免密代管托管、OIDC 邮箱自动绑定与 ChatGPT 大模型智能回复。
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
              onClick={() => onNavigate('miniprogram')}
              variant="outline"
              className="border-white/40 text-white bg-white/10 hover:bg-white/20 font-medium"
            >
              <AppWindow className="h-4 w-4 mr-1.5 text-amber-300" />
              微信小程序快速注册
            </Button>
            <Button
              onClick={() => onNavigate('wxgate')}
              variant="outline"
              className="border-white/30 text-white hover:bg-white/10"
            >
              微信网关与凭证中心
              <ArrowUpRight className="h-4 w-4 ml-1" />
            </Button>
          </div>
        </div>
      </div>

      {/* 核心指标统计 */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">当前租户公众号</CardTitle>
            <Smartphone className="h-4 w-4 text-emerald-600 dark:text-emerald-400" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{accounts?.length || 0}</div>
            <p className="text-xs text-muted-foreground mt-1 flex items-center gap-1">
              当前激活: <span className="font-semibold text-foreground truncate max-w-[120px]">{currentApp ? currentApp.name : '未选择'}</span>
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">当前号关注者</CardTitle>
            <Users className="h-4 w-4 text-blue-600 dark:text-blue-400" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{fansData?.pagination?.total ?? 0}</div>
            <p className="text-xs text-muted-foreground mt-1">支持后台异步队列批量同步</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">多媒体素材总数</CardTitle>
            <ImageIcon className="h-4 w-4 text-purple-600 dark:text-purple-400" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{materialsData?.pagination?.total ?? 0}</div>
            <p className="text-xs text-muted-foreground mt-1">永久/临时素材与图片直连</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">综合服务连通性</CardTitle>
            <Activity className="h-4 w-4 text-emerald-600" />
          </CardHeader>
          <CardContent>
            <div className="flex items-center gap-2">
              <span className="text-2xl font-bold capitalize">{overallHealthy ? '良好' : '部分受限'}</span>
              <Badge variant={overallHealthy ? 'success' : 'destructive'} className="text-[10px]">
                {overallHealthy ? 'ONLINE' : 'DEGRADED'}
              </Badge>
            </div>
            <p className="text-xs text-muted-foreground mt-1 font-mono">
              出口IP: {publicIP?.ip || '127.0.0.1'}
            </p>
          </CardContent>
        </Card>
      </div>

      {/* 快捷配置指引与服务连通监控 */}
      <div className="grid gap-6 lg:grid-cols-12 items-start">
        {/* 左侧：当前公众号接入参数 */}
        <div className="lg:col-span-5">
          <Card>
            <CardHeader>
              <CardTitle className="text-base flex items-center gap-2">
                <ShieldCheck className="h-5 w-5 text-emerald-600" />
                <span>微信公众号接入参数</span>
              </CardTitle>
              <CardDescription className="text-xs">
                {currentApp?.account_type === 'authorized'
                  ? '该公众号由微信开放平台第三方一键授权代管'
                  : '直连模式需配置至微信公众后台 (mp.weixin.qq.com)'}
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-3 text-sm">
              {currentApp ? (
                <>
                  <div className="p-3 rounded-lg bg-muted/60 space-y-1">
                    <div className="text-xs text-muted-foreground">服务器 URL (微信后台填写):</div>
                    <div className="font-mono text-xs font-semibold select-all break-all text-primary">
                      {(typeof window !== "undefined" && window.location.protocol === "https:" && (currentApp.callback_url || "").startsWith("http:")) ? currentApp.callback_url.replace(/^http:/, "https:") : (currentApp.callback_url || `/wxmp/${currentApp.appid}/handler`)}
                    </div>
                  </div>
                  <div className="grid grid-cols-2 gap-3">
                    <div className="p-3 rounded-lg bg-muted/60 space-y-1">
                      <div className="text-xs text-muted-foreground">令牌 (Token):</div>
                      <div className="font-mono text-xs font-semibold select-all">{currentApp.token || '未配置'}</div>
                    </div>
                    <div className="p-3 rounded-lg bg-muted/60 space-y-1">
                      <div className="text-xs text-muted-foreground">接入模式:</div>
                      <div className="text-xs font-medium">
                        {currentApp.account_type === 'authorized' ? '第三方平台托管' : '直接凭据接入'}
                      </div>
                    </div>
                  </div>
                </>
              ) : (
                <div className="p-6 text-center text-muted-foreground text-sm">
                  请先在顶部下拉选择一个公众号以查看配置参数
                </div>
              )}
            </CardContent>
          </Card>
        </div>

        {/* 右侧：后端技术栈真实连通性与连接健康监控 (重点改进！) */}
        <div className="lg:col-span-7">
          <Card>
            <CardHeader className="pb-3 flex flex-row items-center justify-between">
              <div>
                <CardTitle className="text-base flex items-center gap-2">
                  <Wifi className="h-5 w-5 text-emerald-600" />
                  <span>服务连通性与连接健康监控</span>
                </CardTitle>
                <CardDescription className="text-xs">
                  实时检测数据库、缓存、后台任务队列及微信官方 API 真实握手连接状态
                </CardDescription>
              </div>

              <Button
                variant="outline"
                size="sm"
                onClick={() => refetchHealth()}
                disabled={checkingHealth}
                className="h-8 text-xs gap-1"
                title="立即发起连通性自检"
              >
                <RefreshCw className={`h-3.5 w-3.5 ${checkingHealth ? 'animate-spin' : ''}`} />
                <span>重新检测</span>
              </Button>
            </CardHeader>

            <CardContent className="space-y-3 pt-1">
              {components.map((item: any, idx: number) => {
                const isConn = item.status === 'connected'
                const isNotConfig = item.status === 'not_configured'
                return (
                  <div
                    key={idx}
                    className="p-3 rounded-xl border bg-card/60 flex flex-col sm:flex-row sm:items-center justify-between gap-2.5 transition-colors hover:bg-muted/40"
                  >
                    <div className="space-y-0.5">
                      <div className="flex items-center gap-2">
                        <span className="font-semibold text-sm text-foreground">{item.name}</span>
                        {isConn ? (
                          <Badge variant="success" className="text-[10px] gap-0.5 py-0">
                            <CheckCircle2 className="h-3 w-3" />
                            <span>正常已连接</span>
                          </Badge>
                        ) : isNotConfig ? (
                          <Badge variant="secondary" className="text-[10px] gap-0.5 py-0 text-muted-foreground">
                            <HelpCircle className="h-3 w-3" />
                            <span>未配置 (可选)</span>
                          </Badge>
                        ) : (
                          <Badge variant="destructive" className="text-[10px] gap-0.5 py-0">
                            <XCircle className="h-3 w-3" />
                            <span>连接异常</span>
                          </Badge>
                        )}

                        {item.latency_ms > 0 && (
                          <span className="text-[11px] font-mono text-muted-foreground">
                            ({item.latency_ms} ms)
                          </span>
                        )}
                      </div>

                      <div className="text-xs text-muted-foreground font-mono truncate max-w-md">
                        {item.target}
                      </div>
                    </div>

                    <div className="text-right sm:self-center">
                      <span className="text-xs text-muted-foreground font-medium">
                        {item.details}
                      </span>
                    </div>
                  </div>
                )
              })}

              {components.length === 0 && (
                <div className="py-6 text-center text-xs text-muted-foreground">正在探测服务连通性...</div>
              )}
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  )
}
