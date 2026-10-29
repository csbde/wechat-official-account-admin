import React, { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'
import { useAppStore, WxAccount } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { Button } from '@/components/ui/button'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog'
import {
  Plus,
  RefreshCw,
  Smartphone,
  Info,
  Copy,
  Trash2,
  Edit,
  Check,
  Star,
  ExternalLink,
  ShieldCheck,
  QrCode,
  Layers,
  Settings,
  Building2,
  CheckCircle,
  HelpCircle,
  Filter,
} from 'lucide-react'
import { AuthorizerDetailView } from './AuthorizerDetail'
import { TenantScopeBanner } from '@/components/common/TenantScopeBanner'
import { toast } from 'sonner'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import * as z from 'zod'

const accountSchema = z.object({
  name: z.string().min(2, '公众号名称至少2个字符'),
  appid: z.string().min(6, 'AppID 不能为空'),
  appsecret: z.string().optional(),
  token: z.string().optional(),
  encoding_aes_key: z.string().optional(),
  description: z.string().optional(),
  app_type: z.enum(['service', 'subscription', 'miniprogram']).default('service'),
  verify_type: z.number().default(-1),
})

type AccountFormData = z.infer<typeof accountSchema>

export const AccountsPage: React.FC = () => {
  // 协议自适应保障：如果当前站点是 https，自动规范输出为 https 协议
  const formatSecureURL = (rawUrl?: string) => {
    if (!rawUrl) return ""
    if (typeof window !== "undefined" && window.location.protocol === "https:" && rawUrl.startsWith("http:")) {
      return rawUrl.replace(/^http:/, "https:")
    }
    return rawUrl
  }

  const queryClient = useQueryClient()
  const { currentApp, setCurrentApp } = useAppStore()
  const { user, tenant: ownTenant, activeTenantId } = useAuthStore()
  const isSuperAdmin = user?.role === 'superadmin'
  const [transferTarget, setTransferTarget] = useState<any>(null)
  const [transferTenantId, setTransferTenantId] = useState<number>(0)

  const { data: tenantOptions } = useQuery<any[]>({
    queryKey: ['system-tenant-options'],
    queryFn: () => api.get('/api/system/tenants/options'),
    enabled: isSuperAdmin,
    retry: 0,
  })
  const { data: allTenants } = useQuery<any[]>({
    queryKey: ['system-tenants'],
    queryFn: () => api.get('/api/system/tenants'),
    enabled: isSuperAdmin,
    retry: 0,
  })

  const transferMutation = useMutation({
    mutationFn: ({ id, tenant_id }: any) => api.post(`/api/accounts/${id}/transfer-tenant`, { tenant_id }),
    onSuccess: (res: any) => {
      toast.success(res?.message || '迁移完成')
      setTransferTarget(null)
      queryClient.invalidateQueries({ queryKey: ['accounts'] })
      queryClient.invalidateQueries({ queryKey: ['system-tenants'] })
    },
  })

  const [filterType, setFilterType] = useState<string>('all')
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editingId, setEditingId] = useState<number | null>(null)
  const [openComponentModal, setOpenComponentModal] = useState(false)
  const [detailModalOpen, setDetailModalOpen] = useState(false)
  const [currentDetail, setCurrentDetail] = useState<any>(null)
  const [loadingDetail, setLoadingDetail] = useState(false)
  const [detailError, setDetailError] = useState('')
  const [detailTarget, setDetailTarget] = useState<any>(null)

  // 获取公众号列表
  const { data: accounts, isLoading } = useQuery<WxAccount[]>({
    queryKey: ['accounts'],
    queryFn: () => api.get('/api/accounts'),
  })

  // 获取微信开放平台第三方平台配置
  const { data: openComp } = useQuery<any>({
    queryKey: ['open-component'],
    queryFn: () => api.get('/api/open/component'),
  })

  const form = useForm<AccountFormData>({
    resolver: zodResolver(accountSchema),
    defaultValues: {
      name: '',
      appid: '',
      appsecret: '',
      token: 'wechat_token_2025',
      encoding_aes_key: '',
      description: '',
      app_type: 'service',
      verify_type: -1,
    },
  })

  // 第三方平台配置表单状态
  const [compName, setCompName] = useState('')
  const [compAppId, setCompAppId] = useState('')
  const [compSecret, setCompSecret] = useState('')
  const [compToken, setCompToken] = useState('')
  const [compAesKey, setCompAesKey] = useState('')
  const [manualTicket, setManualTicket] = useState('')
  const [requestingTicket, setRequestingTicket] = useState(false)

  // 打开第三方平台配置时回填
  const handleOpenCompSettings = () => {
    if (openComp) {
      setCompName(openComp.name || '')
      setCompAppId(openComp.component_appid || '')
      setCompSecret(openComp.component_appsecret || '')
      setCompToken(openComp.token || '')
      setCompAesKey(openComp.encoding_aes_key || '')
    }
    setOpenComponentModal(true)
  }

  // 一键请求微信立即推送 Ticket
  const handleRequestPushTicket = async () => {
    try {
      setRequestingTicket(true)
      const res: any = await api.post('/api/open/request-ticket')
      toast.success(res.message || '已成功向微信服务器发送请求，请等待约 10 秒后刷新查看')
      setTimeout(() => queryClient.invalidateQueries({ queryKey: ['open-component'] }), 6000)
    } catch (e: any) {
      toast.error('请求失败: ' + (e.message || '请检查网络或第三方凭证是否有效'))
    } finally {
      setRequestingTicket(false)
    }
  }

  // 手动保存 Ticket
  const handleSaveManualTicket = async () => {
    if (!manualTicket.trim()) {
      toast.error('请输入 Ticket 字符串')
      return
    }
    try {
      const res: any = await api.post('/api/open/manual-ticket', { ticket: manualTicket.trim() })
      toast.success(res.message || 'Ticket 更新成功！')
      queryClient.invalidateQueries({ queryKey: ['open-component'] })
      setManualTicket('')
    } catch (e: any) {
      toast.error('保存失败: ' + (e.message || '网络错误'))
    }
  }

  // 获取微信官方权威档案
  const fetchAuthorizerDetail = async (acc: any) => {
    setLoadingDetail(true)
    setDetailError('')
    try {
      // 注意: api 响应拦截器已把 {code,message,data} 解包，这里拿到的就是档案数据本身
      const res: any = await api.get(`/api/accounts/${acc.id}/authorizer-info`)
      setCurrentDetail(res)
    } catch (e: any) {
      setDetailError(e?.message || '获取微信官方档案失败，请检查该账号凭据与网络连通性')
      setCurrentDetail(null)
    } finally {
      setLoadingDetail(false)
    }
  }

  const handleViewDetail = (acc: any) => {
    setDetailTarget(acc)
    setCurrentDetail(null)
    setDetailModalOpen(true)
    fetchAuthorizerDetail(acc)
  }

  // 保存第三方平台凭据
  const saveCompMutation = useMutation({
    mutationFn: (values: any) => api.post('/api/open/component', values),
    onSuccess: () => {
      toast.success('微信开放平台第三方平台凭证保存成功')
      queryClient.invalidateQueries({ queryKey: ['open-component'] })
      setOpenComponentModal(false)
    },
  })

  // 发起微信第三方扫码授权
  const handleStartWeChatAuth = async () => {
    try {
      const res: any = await api.get('/api/open/auth-url')
      if (res.auth_url) {
        window.open(res.auth_url, '_blank')
        toast.info('已在弹窗中打开微信官方授权页，请公众号管理员微信扫码确认授权')
      }
    } catch (e: any) {
      toast.error(e.message || '请先配置微信开放平台第三方凭据')
    }
  }

  // 创建/更新公众号
  const saveMutation = useMutation({
    mutationFn: (values: AccountFormData) => {
      if (editingId) {
        return api.put(`/api/accounts/${editingId}`, values) as unknown as Promise<WxAccount>
      }
      return api.post('/api/accounts', values) as unknown as Promise<WxAccount>
    },
    onSuccess: (data: WxAccount) => {
      toast.success(editingId ? '公众号更新成功' : '公众号添加成功')
      queryClient.invalidateQueries({ queryKey: ['accounts'] })
      setDialogOpen(false)
      form.reset()
      if (!currentApp || currentApp.id === data.id) {
        setCurrentApp(data)
      }
    },
  })

  // 删除
  const deleteMutation = useMutation({
    mutationFn: (id: number) => api.delete(`/api/accounts/${id}`),
    onSuccess: (_, id) => {
      toast.success('公众号已移除')
      queryClient.invalidateQueries({ queryKey: ['accounts'] })
      if (currentApp?.id === id) {
        setCurrentApp(null)
      }
    },
  })

  // 设为默认
  const setDefaultMutation = useMutation({
    mutationFn: (id: number) => api.post(`/api/accounts/${id}/set-default`),
    onSuccess: () => {
      toast.success('已设为默认公众号')
      queryClient.invalidateQueries({ queryKey: ['accounts'] })
    },
  })

  const handleOpenCreate = () => {
    setEditingId(null)
    form.reset({
      name: '',
      appid: '',
      appsecret: '',
      token: 'token_' + Math.random().toString(36).substring(2, 10),
      encoding_aes_key: '',
      description: '',
    })
    setDialogOpen(true)
  }

  const handleOpenEdit = (acc: any) => {
    setEditingId(acc.id)
    form.reset({
      name: acc.name,
      appid: acc.appid,
      appsecret: acc.appsecret || '',
      token: acc.token,
      encoding_aes_key: acc.encoding_aes_key,
      description: acc.description || '',
    })
    setDialogOpen(true)
  }

  const copyToClipboard = (text: string, label: string) => {
    navigator.clipboard.writeText(text)
    toast.success(`已复制 ${label} 到剪贴板`)
  }

  return (
    <div className="space-y-4 sm:space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-bold tracking-tight">微信公众号与开放平台管理</h2>
          <p className="text-sm text-muted-foreground">
            支持传统直连接入与微信开放平台第三方平台（免密扫码一键授权托管）双接入模式
          </p>
        </div>

        <div className="flex flex-col sm:flex-row flex-wrap items-stretch sm:items-center gap-2 w-full sm:w-auto">
          {/* 微信开放平台第三方平台设置 */}
          <Button variant="outline" onClick={handleOpenCompSettings} className="gap-1.5 text-xs w-full sm:w-auto min-h-[38px] justify-center">
            <Layers className="h-4 w-4 text-emerald-600" />
            <span>开放平台第三方设置</span>
          </Button>

          {/* 微信官方扫码一键授权 */}
          <Button
            onClick={handleStartWeChatAuth}
            className="gap-1.5 text-xs bg-emerald-600 hover:bg-emerald-700 text-white font-semibold shadow-sm w-full sm:w-auto min-h-[38px] justify-center"
          >
            <QrCode className="h-4 w-4" />
            <span>微信扫码一键授权接入</span>
          </Button>

          {/* 手动添加 */}
          <Button variant="secondary" onClick={handleOpenCreate} className="gap-1.5 text-xs w-full sm:w-auto min-h-[38px] justify-center">
            <Plus className="h-4 w-4" />
            <span>手动添加公众号</span>
          </Button>
        </div>
      </div>

      <TenantScopeBanner extra="新增账号会自动归属当前租户；需要为其他组织接入，请在「系统与权限 → 租户与组织」新建租户并切换视角" />

      {/* 开放平台第三方接入状态提示条 */}
      {openComp ? (
        <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-2.5 p-3.5 rounded-xl bg-emerald-500/10 border border-emerald-500/20 text-xs text-emerald-800 dark:text-emerald-300">
          <div className="flex items-center gap-2">
            <ShieldCheck className="h-4 w-4 text-emerald-600 shrink-0" />
            <span>
              已接入微信开放平台第三方平台: <strong>{openComp.name}</strong> ({openComp.component_appid})。支持公众号管理员扫码全自动免密代管托管！
            </span>
          </div>
          <button onClick={handleOpenCompSettings} className="text-emerald-600 font-semibold hover:underline shrink-0">
            查看配置参数
          </button>
        </div>
      ) : (
        <div className="flex items-center justify-between p-3.5 rounded-xl bg-muted/60 border text-xs text-muted-foreground">
          <div className="flex items-center gap-2">
            <Layers className="h-4 w-4 text-primary shrink-0" />
            <span>💡 提示：如需实现客户公众号免输入 AppSecret 一键扫码授权接入，请在右上角点击「开放平台第三方设置」配置第三方平台参数。</span>
          </div>
        </div>
      )}

      {/* 账号分类过滤导航条 */}
      <div className="flex flex-wrap items-center justify-between gap-2.5 pb-1">
        <div className="flex items-center gap-1.5 bg-muted/60 p-1 rounded-xl text-xs">
          <Button
            type="button"
            variant={filterType === 'all' ? 'default' : 'ghost'}
            size="sm"
            onClick={() => setFilterType('all')}
            className="h-7 text-xs px-2.5"
          >
            全部 ({accounts?.length || 0})
          </Button>
          <Button
            type="button"
            variant={filterType === 'service' ? 'default' : 'ghost'}
            size="sm"
            onClick={() => setFilterType('service')}
            className="h-7 text-xs px-2.5"
          >
            服务号 ({accounts?.filter((a: any) => (a.app_type === 'service' || (!a.app_type && a.account_type !== 'miniprogram'))).length || 0})
          </Button>
          <Button
            type="button"
            variant={filterType === 'subscription' ? 'default' : 'ghost'}
            size="sm"
            onClick={() => setFilterType('subscription')}
            className="h-7 text-xs px-2.5"
          >
            订阅号 ({accounts?.filter((a: any) => a.app_type === 'subscription' || a.service_type === 0 || a.service_type === 1).length || 0})
          </Button>
          <Button
            type="button"
            variant={filterType === 'miniprogram' ? 'default' : 'ghost'}
            size="sm"
            onClick={() => setFilterType('miniprogram')}
            className="h-7 text-xs px-2.5"
          >
            小程序 ({accounts?.filter((a: any) => a.app_type === 'miniprogram' || a.account_type === 'miniprogram').length || 0})
          </Button>
        </div>
        <span className="text-[11px] text-muted-foreground hidden sm:inline">
          提示：不同类型微信账号接口权限不同，服务号支持网页授权与模板消息，小程序支持代码发布
        </span>
      </div>

      {/* 公众号卡片列表 */}
      {isLoading ? (
        <div className="py-12 text-center text-muted-foreground">正在加载公众号数据...</div>
      ) : accounts && accounts.length > 0 ? (
        <div className="grid gap-4 sm:gap-6 grid-cols-1 sm:grid-cols-2 lg:grid-cols-3">
          {accounts
            .filter((acc: any) => {
              if (filterType === 'all') return true
              const isMini = acc.app_type === 'miniprogram' || acc.account_type === 'miniprogram'
              const isSub = acc.app_type === 'subscription' || acc.service_type === 0 || acc.service_type === 1
              const isServ = (acc.app_type === 'service' || (!acc.app_type && !isMini && !isSub))
              if (filterType === 'miniprogram') return isMini
              if (filterType === 'subscription') return isSub
              if (filterType === 'service') return isServ
              return true
            })
            .map((acc: any) => {
            const isSelected = currentApp?.id === acc.id
            const isAuthorized = acc.account_type === 'authorized'
            const isMini = acc.app_type === 'miniprogram' || acc.account_type === 'miniprogram'
            const isSub = acc.app_type === 'subscription' || acc.service_type === 0 || acc.service_type === 1
            const isVerified = acc.verify_type === 0

            return (
              <Card
                key={acc.id}
                className={`relative overflow-hidden transition-all duration-200 border-2 ${
                  isSelected ? 'border-primary ring-2 ring-primary/20 shadow-md' : 'hover:border-border/80'
                }`}
              >
                <CardHeader className="pb-3">
                  <div className="flex items-start justify-between">
                    <div className="flex items-center gap-3">
                      <div className="h-10 w-10 rounded-xl bg-primary/10 text-primary flex items-center justify-center font-bold text-base overflow-hidden">
                        {acc.avatar_url ? (
                          <img src={acc.avatar_url} alt="" className="h-full w-full object-cover" />
                        ) : (
                          <Smartphone className="h-5 w-5" />
                        )}
                      </div>
                      <div>
                        <CardTitle className="text-base flex items-center gap-1.5 flex-wrap">
                          <span>{acc.name}</span>
                          {acc.is_default && (
                            <Badge variant="success" className="text-[10px]">
                              默认
                            </Badge>
                          )}
                          {/* 核心区别：服务号 / 订阅号 / 小程序 */}
                          {isMini ? (
                            <span className="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-semibold bg-purple-500/15 text-purple-700 dark:text-purple-300 border border-purple-500/20">
                              小程序
                            </span>
                          ) : isSub ? (
                            <span className="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-semibold bg-sky-500/15 text-sky-700 dark:text-sky-300 border border-sky-500/20">
                              订阅号
                            </span>
                          ) : (
                            <span className="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-semibold bg-emerald-500/15 text-emerald-700 dark:text-emerald-300 border border-emerald-500/20">
                              服务号
                            </span>
                          )}
                          {/* 认证状态标识 */}
                          {isVerified ? (
                            <span className="inline-flex items-center gap-0.5 px-1.5 py-0.5 rounded text-[10px] font-medium bg-blue-500/10 text-blue-600 dark:text-blue-400">
                              <CheckCircle className="h-2.5 w-2.5" /> 微信认证
                            </span>
                          ) : (
                            <span className="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium bg-muted text-muted-foreground">
                              未认证
                            </span>
                          )}
                          <Badge variant={isAuthorized ? 'default' : 'secondary'} className="text-[10px]">
                            {isAuthorized ? '第三方代管' : '直连凭据'}
                          </Badge>
                        </CardTitle>
                        <CardDescription className="font-mono text-xs mt-0.5">
                          {acc.appid}
                        </CardDescription>
                      </div>
                    </div>
                  </div>
                </CardHeader>

                <CardContent className="space-y-3 text-xs">
                  {acc.description && (
                    <p className="text-muted-foreground line-clamp-2">{acc.description}</p>
                  )}

                  {/* 接入方式与参数展示 */}
                  <div className="p-2.5 rounded-lg bg-muted/60 space-y-1.5 font-mono text-[11px]">
                    <div className="flex items-center justify-between text-muted-foreground">
                      <span>{isAuthorized ? '接入方式:' : '服务器 URL (微信后台填写):'}</span>
                      {!isAuthorized && (
                        <button
                          onClick={() => copyToClipboard(formatSecureURL(acc.callback_url) || '', 'URL')}
                          className="hover:text-foreground text-primary inline-flex items-center gap-1"
                        >
                          <Copy className="h-3 w-3" />
                          <span>复制</span>
                        </button>
                      )}
                    </div>
                    <div className="truncate text-foreground select-all bg-background/50 px-2 py-1 rounded">
                      {isAuthorized
                        ? `由微信开放平台托管 (${acc.component_appid || 'Component'})`
                        : formatSecureURL(acc.callback_url) || `/wxmp/${acc.appid}/handler`}
                    </div>

                    {!isAuthorized && (
                      <>
                        <div className="flex items-center justify-between text-muted-foreground pt-1">
                          <span>Token:</span>
                          <button
                            onClick={() => copyToClipboard(acc.token, 'Token')}
                            className="hover:text-foreground text-primary inline-flex items-center gap-1"
                          >
                            <Copy className="h-3 w-3" />
                            <span>复制</span>
                          </button>
                        </div>
                        <div className="truncate text-foreground select-all bg-background/50 px-2 py-1 rounded">
                          {acc.token || '未设置'}
                        </div>
                      </>
                    )}
                  </div>

                  {/* 底部操作条 */}
                  <div className="pt-2 flex flex-col sm:flex-row items-start sm:items-center justify-between gap-2 border-t border-border/50">
                    <div className="flex items-center gap-1 flex-wrap">
                      {(acc as any).tenant_missing && (
                        <Badge variant="destructive" className="text-[10px]">
                          租户记录缺失
                        </Badge>
                      )}
                      {isSuperAdmin && (acc as any).tenant_name && (acc as any).tenant_id !== ownTenant?.id && (
                        <Badge variant="secondary" className="text-[10px]">
                          归属：{(acc as any).tenant_name}
                        </Badge>
                      )}
                      {isSelected ? (
                        <Badge variant="outline" className="text-primary border-primary gap-1">
                          <Check className="h-3 w-3" />
                          <span>当前激活</span>
                        </Badge>
                      ) : (
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => {
                            setCurrentApp(acc)
                            toast.success(`已切换至: ${acc.name}`)
                          }}
                          className="h-7 text-xs font-medium"
                        >
                          切换至该账号
                        </Button>
                      )}
                    </div>

                    <div className="flex items-center gap-1 flex-wrap">
                      {!acc.is_default && (
                        <Button
                          variant="ghost"
                          size="icon"
                          title="设为默认"
                          onClick={() => setDefaultMutation.mutate(acc.id)}
                          className="h-8 w-8 text-muted-foreground hover:text-amber-500"
                        >
                          <Star className="h-4 w-4" />
                        </Button>
                      )}
                      {isSuperAdmin && (
                        <Button
                          variant="ghost"
                          size="icon"
                          title="迁移该公众号到其它租户 (含其菜单/素材/粉丝等数据)"
                          onClick={() => {
                            setTransferTarget(acc)
                            setTransferTenantId(0)
                          }}
                          className="h-8 w-8 text-muted-foreground hover:text-sky-500"
                        >
                          <Building2 className="h-4 w-4" />
                        </Button>
                      )}
                      <Button
                        variant="ghost"
                        size="icon"
                        title="查看微信官方权威档案与权限"
                        onClick={() => handleViewDetail(acc)}
                        className="h-8 w-8 text-muted-foreground hover:text-primary"
                      >
                        <Info className="h-4 w-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        title="编辑"
                        onClick={() => handleOpenEdit(acc)}
                        className="h-8 w-8 text-muted-foreground hover:text-foreground"
                      >
                        <Edit className="h-4 w-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        title="删除"
                        onClick={() => {
                          if (confirm(`确认删除公众号 "${acc.name}" 吗？此操作不可逆`)) {
                            deleteMutation.mutate(acc.id)
                          }
                        }}
                        className="h-8 w-8 text-muted-foreground hover:text-destructive"
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </div>
                  </div>
                </CardContent>
              </Card>
            )
          })}
        </div>
      ) : (
        <Card className="p-12 text-center space-y-4">
          <div className="mx-auto h-12 w-12 rounded-full bg-primary/10 text-primary flex items-center justify-center">
            <Smartphone className="h-6 w-6" />
          </div>
          <div className="space-y-1">
            <h3 className="font-semibold text-lg">暂未配置任何微信公众号</h3>
            <p className="text-sm text-muted-foreground max-w-sm mx-auto">
              支持直接输入 AppID / AppSecret 手动添加，或配置第三方平台后微信扫码一键授权接入。
            </p>
          </div>
          <div className="pt-2 flex justify-center gap-3">
            <Button onClick={handleStartWeChatAuth} className="gap-2 bg-emerald-600 hover:bg-emerald-700">
              <QrCode className="h-4 w-4" />
              <span>微信扫码一键授权</span>
            </Button>
            <Button variant="outline" onClick={handleOpenCreate} className="gap-2">
              <Plus className="h-4 w-4" />
              <span>手动添加公众号</span>
            </Button>
          </div>
        </Card>
      )}

      {/* 微信开放平台第三方配置 Modal */}
      <Dialog open={openComponentModal} onOpenChange={setOpenComponentModal}>
        <DialogContent className="sm:max-w-xl max-h-[85vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>微信开放平台 (第三方平台) 配置</DialogTitle>
          </DialogHeader>

          <div className="space-y-4 py-2 text-sm">
            {openComp && (
              <div className="p-3 bg-muted/60 rounded-xl space-y-2 text-xs font-mono">
                <div className="font-semibold text-foreground font-sans">微信开放平台后台需配置的 URL:</div>
                <div className="space-y-1">
                  <div className="text-muted-foreground flex justify-between">
                    <span>授权事件接收 URL:</span>
                    <button onClick={() => copyToClipboard(formatSecureURL(openComp.auth_event_url), '授权事件URL')} className="text-primary hover:underline">复制</button>
                  </div>
                  <div className="bg-background p-1.5 rounded select-all break-all">{formatSecureURL(openComp.auth_event_url)}</div>
                </div>
                <div className="space-y-1">
                  <div className="text-muted-foreground flex justify-between">
                    <span>代公众号消息事件接收 URL:</span>
                    <button onClick={() => copyToClipboard(formatSecureURL(openComp.msg_notify_url), '消息事件URL')} className="text-primary hover:underline">复制</button>
                  </div>
                  <div className="bg-background p-1.5 rounded select-all break-all">{formatSecureURL(openComp.msg_notify_url)}</div>
                </div>
              </div>
            )}

            {/* Ticket 状态与极速自愈控制 */}
            <div className="p-3 rounded-xl border bg-muted/40 space-y-2.5 text-xs">
              <div className="flex items-center justify-between">
                <span className="font-semibold text-foreground flex items-center gap-1.5">
                  <span>ComponentVerifyTicket 状态:</span>
                  {openComp?.verify_ticket ? (
                    <Badge variant="success" className="text-[10px] py-0">已就绪</Badge>
                  ) : (
                    <Badge variant="destructive" className="text-[10px] py-0">未接收到</Badge>
                  )}
                </span>
                <Button
                  type="button"
                  size="sm"
                  variant="outline"
                  onClick={handleRequestPushTicket}
                  disabled={requestingTicket}
                  className="h-7 text-[11px] gap-1 text-primary border-primary/40 hover:bg-primary/10"
                  title="调用微信官方接口触发立即推送，无需等待10分钟"
                >
                  <RefreshCw className={`h-3 w-3 ${requestingTicket ? 'animate-spin' : ''}`} />
                  <span>{requestingTicket ? '请求中...' : '一键请求微信立即推送'}</span>
                </Button>
              </div>

              {openComp?.verify_ticket ? (
                <div className="p-2 bg-background/60 rounded font-mono text-[11px] truncate select-all text-muted-foreground">
                  {openComp.verify_ticket}
                </div>
              ) : (
                <div className="space-y-2">
                  <p className="text-[11px] text-muted-foreground">
                    💡 提示：微信通常每隔 10 分钟自动推送一次 Ticket。若刚刚部署，可点击右上方「一键请求微信立即推送」，微信将在数秒内推送到您的授权接收 URL；或手动在下方粘贴生效：
                  </p>
                  <div className="flex gap-1.5">
                    <Input
                      value={manualTicket}
                      onChange={(e) => setManualTicket(e.target.value)}
                      placeholder="在此手动粘贴 Ticket (选填)"
                      className="h-7 text-xs font-mono flex-1"
                    />
                    <Button type="button" size="sm" onClick={handleSaveManualTicket} className="h-7 text-xs px-2.5">
                      手动生效
                    </Button>
                  </div>
                </div>
              )}
            </div>

            <div className="space-y-1.5">
              <label className="text-xs font-semibold">第三方平台名称 *</label>
              <Input value={compName} onChange={(e) => setCompName(e.target.value)} placeholder="例如: 智能校园第三方管理平台" />
            </div>

            <div className="space-y-1.5">
              <label className="text-xs font-semibold">第三方平台 AppID (ComponentAppID) *</label>
              <Input value={compAppId} onChange={(e) => setCompAppId(e.target.value)} placeholder="wx50..." />
            </div>

            <div className="space-y-1.5">
              <label className="text-xs font-semibold">第三方平台 AppSecret *</label>
              <Input type="password" value={compSecret} onChange={(e) => setCompSecret(e.target.value)} placeholder="开放平台生成的 AppSecret" />
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <div className="space-y-1.5">
                <label className="text-xs font-semibold">消息校验 Token *</label>
                <Input value={compToken} onChange={(e) => setCompToken(e.target.value)} placeholder="与开放平台一致" />
              </div>
              <div className="space-y-1.5">
                <label className="text-xs font-semibold">消息加解密 Key (AESKey) *</label>
                <Input value={compAesKey} onChange={(e) => setCompAesKey(e.target.value)} placeholder="43位字符" />
              </div>
            </div>

            <DialogFooter className="pt-2">
              <Button variant="outline" onClick={() => setOpenComponentModal(false)}>取消</Button>
              <Button
                onClick={() => {
                  if (!compName || !compAppId || !compSecret || !compToken || !compAesKey) {
                    toast.error('所有第三方平台必填项均不能为空')
                    return
                  }
                  saveCompMutation.mutate({
                    name: compName,
                    component_appid: compAppId,
                    component_appsecret: compSecret,
                    token: compToken,
                    encoding_aes_key: compAesKey,
                  })
                }}
                disabled={saveCompMutation.isPending}
              >
                {saveCompMutation.isPending ? '保存中...' : '确定保存'}
              </Button>
            </DialogFooter>
          </div>
        </DialogContent>
      </Dialog>

      {/* 手动新增/编辑公众号 Modal */}
      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>{editingId ? '编辑微信公众号配置' : '手动添加微信公众号'}</DialogTitle>
          </DialogHeader>

          <form onSubmit={form.handleSubmit((v) => saveMutation.mutate(v))} className="space-y-4 py-2">
            <div className="space-y-1">
              <label className="text-xs font-semibold">公众号名称 *</label>
              <Input placeholder="例如: 校园官方通知" {...form.register('name')} />
              {form.formState.errors.name && (
                <span className="text-[11px] text-destructive">{form.formState.errors.name.message}</span>
              )}
            </div>

            <div className="space-y-1">
              <label className="text-xs font-semibold">AppID (开发者ID) *</label>
              <Input placeholder="wx1234567890abcdef" disabled={!!editingId} {...form.register('appid')} />
              {form.formState.errors.appid && (
                <span className="text-[11px] text-destructive">{form.formState.errors.appid.message}</span>
              )}
            </div>

            <div className="space-y-1">
              <label className="text-xs font-semibold">AppSecret (直连时必填)</label>
              <Input
                type="password"
                placeholder={editingId ? '若不修改请保持原样' : '微信公众平台生成的 AppSecret'}
                {...form.register('appsecret')}
              />
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <div className="space-y-1">
                <label className="text-xs font-semibold">令牌 (Token)</label>
                <Input placeholder="自定义字符串" {...form.register('token')} />
              </div>
              <div className="space-y-1">
                <label className="text-xs font-semibold">消息加解密密钥 (AESKey)</label>
                <Input placeholder="43位字符(可选)" {...form.register('encoding_aes_key')} />
              </div>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <div className="space-y-1">
                <label className="text-xs font-semibold">账号类型 (AppType) *</label>
                <select
                  {...form.register('app_type')}
                  className="w-full h-9 rounded-lg border border-input bg-background px-3 text-xs"
                >
                  <option value="service">服务号 (高级接口完整，推荐)</option>
                  <option value="subscription">订阅号 (群发与资讯)</option>
                  <option value="miniprogram">微信小程序</option>
                </select>
              </div>
              <div className="space-y-1">
                <label className="text-xs font-semibold">官方认证状态</label>
                <select
                  value={form.watch('verify_type')}
                  onChange={(e) => form.setValue('verify_type', Number(e.target.value))}
                  className="w-full h-9 rounded-lg border border-input bg-background px-3 text-xs"
                >
                  <option value={0}>已通过微信官方认证 (蓝V)</option>
                  <option value={-1}>未认证 / 个人主体</option>
                </select>
              </div>
            </div>

            <div className="space-y-1">
              <label className="text-xs font-semibold">账号描述/备注</label>
              <Input placeholder="输入该公众号用途或组织" {...form.register('description')} />
            </div>

            <DialogFooter className="pt-2">
              <Button type="button" variant="outline" onClick={() => setDialogOpen(false)}>
                取消
              </Button>
              <Button type="submit" disabled={saveMutation.isPending}>
                {saveMutation.isPending ? '保存中...' : '确定保存'}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      {/* 微信官方权威档案与权限集详情弹窗 (api_get_authorizer_info) */}
      <Dialog open={detailModalOpen} onOpenChange={setDetailModalOpen}>
        <DialogContent className="sm:max-w-2xl max-h-[88vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <Info className="h-5 w-5 text-primary" />
              <span>微信官方权威档案详情</span>
            </DialogTitle>
          </DialogHeader>

          {loadingDetail ? (
            <div className="py-14 text-center text-xs text-muted-foreground">正在向微信官方调取最新档案...</div>
          ) : detailError ? (
            <div className="space-y-4 py-2">
              <div className="p-3 rounded-xl bg-red-500/10 border border-red-500/25 text-xs text-red-700 dark:text-red-300 space-y-1">
                <div className="font-semibold">调取微信官方档案失败</div>
                <div className="break-words leading-relaxed">{detailError}</div>
              </div>
              <DialogFooter>
                <Button size="sm" variant="outline" onClick={() => detailTarget && fetchAuthorizerDetail(detailTarget)}>
                  重试
                </Button>
                <Button size="sm" onClick={() => setDetailModalOpen(false)}>关闭</Button>
              </DialogFooter>
            </div>
          ) : currentDetail ? (
            <AuthorizerDetailView detail={currentDetail} onClose={() => setDetailModalOpen(false)} />
          ) : null}
        </DialogContent>
      </Dialog>

      {/* 超级管理员：公众号跨租户迁移 */}
      <Dialog open={!!transferTarget} onOpenChange={(v) => !v && setTransferTarget(null)}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <Building2 className="h-5 w-5 text-sky-600" />
              <span>迁移公众号归属租户</span>
            </DialogTitle>
          </DialogHeader>
          <div className="space-y-3 py-2 text-sm">
            <div className="text-xs text-muted-foreground leading-relaxed">
              将 <span className="font-semibold text-foreground">{transferTarget?.name}</span>
              <span className="font-mono"> ({transferTarget?.appid})</span> 及其
              <strong>菜单 / 素材 / 自动回复 / 粉丝 / 标签 / 二维码 / AI 设置 / 日志</strong> 一并迁移到目标租户。
              迁移后原租户将看不到该公众号，请用目标租户的成员或切换视角继续管理。
            </div>
            {(transferTarget as any)?.tenant_missing && (
              <div className="text-[11px] text-destructive">
                该公众号当前归属的租户记录已不存在 (数据残留)，建议迁移到一个正式租户，否则无人可管理它。
              </div>
            )}
            <div className="space-y-1.5">
              <label className="text-xs font-semibold">目标租户 *</label>
              <select
                value={transferTenantId}
                onChange={(e) => setTransferTenantId(Number(e.target.value))}
                className="w-full h-9 rounded-lg border border-input bg-background px-3 text-sm"
              >
                <option value={0}>请选择目标租户...</option>
                {(allTenants || [])
                  .filter((t: any) => t.status === 1 && t.id !== transferTarget?.tenant_id)
                  .map((t: any) => (
                    <option key={t.id} value={t.id}>
                      #{t.id} {t.name} ({t.code}){t.ghost ? ' · 数据残留租户' : ''}
                    </option>
                  ))}
              </select>
              <p className="text-[11px] text-muted-foreground">
                列表仅展示已启用且当前未持有相同 AppID 的租户；同一微信账号不允许被两个租户同时接入。
              </p>
            </div>
            <DialogFooter className="pt-2">
              <Button variant="outline" onClick={() => setTransferTarget(null)}>取消</Button>
              <Button
                disabled={!transferTenantId || transferMutation.isPending}
                onClick={() => {
                  if (!window.confirm('迁移会改变该公众号及其业务数据的租户归属，确认执行？')) return
                  transferMutation.mutate({ id: transferTarget.id, tenant_id: transferTenantId })
                }}
              >
                {transferMutation.isPending ? '迁移中...' : '确认迁移'}
              </Button>
            </DialogFooter>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  )
}
