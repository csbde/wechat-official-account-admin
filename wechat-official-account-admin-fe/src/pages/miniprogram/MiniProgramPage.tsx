import React, { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api, PageResult } from '@/lib/api'
import { useAppStore } from '@/stores/app'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import {
  Table,
  TableHeader,
  TableBody,
  TableHead,
  TableRow,
  TableCell,
} from '@/components/ui/table'
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
  Globe,
  CheckCircle2,
  AlertCircle,
  Clock,
  Sparkles,
  ShieldCheck,
  Send,
  Save,
  Key,
  Link2,
  Unlink,
  Library,
  Trash2,
  ChevronLeft,
  ChevronRight,
  Star,
} from 'lucide-react'
import { toast } from 'sonner'
import { formatDate } from '@/lib/utils'

/** 关联状态字典 (微信官方 wxamplinkget 定义) */
const LINK_STATUS: Record<number, { label: string; cls: string }> = {
  1: { label: '已关联', cls: 'bg-emerald-500/10 text-emerald-600 border-emerald-500/30' },
  2: { label: '待小程序管理员确认', cls: 'bg-amber-500/10 text-amber-600 border-amber-500/30' },
  3: { label: '小程序管理员已拒绝', cls: 'bg-red-500/10 text-red-600 border-red-500/30' },
  12: { label: '待公众号管理员确认', cls: 'bg-blue-500/10 text-blue-600 border-blue-500/30' },
}

/** 微信返回的 create_time 为 Unix 秒级时间戳，需单独转换 */
function formatUnixSec(sec?: number) {
  if (!sec) return '-'
  const d = new Date(sec * 1000)
  if (isNaN(d.getTime())) return '-'
  return d.toLocaleString('zh-CN', { hour12: false })
}

const MINI_TABS = [
  { value: 'register', label: '快速注册任务', icon: Sparkles, color: 'text-amber-500' },
  { value: 'linked', label: '公众号关联小程序', icon: Link2, color: 'text-sky-500' },
  { value: 'templates', label: '小程序代码模板库', icon: Library, color: 'text-violet-500' },
  { value: 'domain', label: '合法域名配置', icon: Globe, color: 'text-blue-500' },
  { value: 'code2session', label: '登录网关', icon: Key, color: 'text-emerald-500' },
] as const

export const MiniProgramPage: React.FC = () => {
  const queryClient = useQueryClient()
  const { currentApp } = useAppStore()

  const [activeTab, setActiveTab] = useState<string>('register')
  const [createModalOpen, setCreateModalOpen] = useState(false)
  const [page, setPage] = useState(1)

  // 快速注册表单状态
  const [corpName, setCorpName] = useState('')
  const [corpCode, setCorpCode] = useState('')
  const [codeType, setCodeType] = useState('1')
  const [legalWechat, setLegalWechat] = useState('')
  const [legalName, setLegalName] = useState('')
  const [phone, setPhone] = useState('')

  // 服务器域名配置状态
  const [reqDomain, setReqDomain] = useState('')
  const [wsDomain, setWsDomain] = useState('')
  const [uploadDomain, setUploadDomain] = useState('')
  const [downloadDomain, setDownloadDomain] = useState('')
  const [webviewDomain, setWebviewDomain] = useState('')

  // Code2Session 测试状态
  const [jsCode, setJsCode] = useState('')
  const [loginResult, setLoginResult] = useState<any>(null)
  const [testingLogin, setTestingLogin] = useState(false)

  // 关联小程序弹窗状态
  const [linkModalOpen, setLinkModalOpen] = useState(false)
  const [linkAppID, setLinkAppID] = useState('')
  const [notifyUsers, setNotifyUsers] = useState('1')
  const [showProfile, setShowProfile] = useState('0')

  // 模板库子标签
  const [tplTab, setTplTab] = useState<'templates' | 'drafts'>('templates')
  const [tplType, setTplType] = useState('0')

  /* ------------------------------ 数据查询 ------------------------------ */

  // 快速注册任务列表
  const { data: taskResult, isLoading: loadingTasks } = useQuery<PageResult<any>>({
    queryKey: ['miniprogram-tasks', page],
    queryFn: () => api.get(`/api/miniprogram/tasks?page=${page}&page_size=10`),
  })

  // 小程序域名配置
  const { data: domainConfig } = useQuery<any>({
    queryKey: ['miniprogram-domain', currentApp?.appid],
    queryFn: () => api.get('/api/miniprogram/domain'),
    enabled: !!currentApp && activeTab === 'domain',
  })

  // 公众号关联的小程序列表
  const {
    data: linkedRes,
    isLoading: loadingLinked,
    refetch: refetchLinked,
    isFetching: fetchingLinked,
  } = useQuery<any>({
    queryKey: ['miniprogram-linked', currentApp?.appid],
    queryFn: () => api.get('/api/miniprogram/linked'),
    enabled: !!currentApp && activeTab === 'linked',
  })

  // 代码模板库
  const {
    data: templates,
    isLoading: loadingTemplates,
    refetch: refetchTemplates,
    isFetching: fetchingTemplates,
  } = useQuery<any[]>({
    queryKey: ['miniprogram-templates', tplType],
    queryFn: () => api.get(`/api/miniprogram/templates?template_type=${tplType}`),
    enabled: activeTab === 'templates' && tplTab === 'templates',
  })

  // 代码草稿箱
  const {
    data: drafts,
    isLoading: loadingDrafts,
    refetch: refetchDrafts,
    isFetching: fetchingDrafts,
  } = useQuery<any[]>({
    queryKey: ['miniprogram-drafts'],
    queryFn: () => api.get('/api/miniprogram/drafts'),
    enabled: activeTab === 'templates' && tplTab === 'drafts',
  })

  React.useEffect(() => {
    if (domainConfig) {
      setReqDomain(domainConfig.requestdomain || '')
      setWsDomain(domainConfig.wsrequestdomain || '')
      setUploadDomain(domainConfig.uploaddomain || '')
      setDownloadDomain(domainConfig.downloaddomain || '')
      setWebviewDomain(domainConfig.webviewdomain || '')
    }
  }, [domainConfig])

  /* ------------------------------ 变更操作 ------------------------------ */

  // 创建快速注册任务
  const createTaskMutation = useMutation({
    mutationFn: (values: any) => api.post('/api/miniprogram/tasks', values),
    onSuccess: () => {
      toast.success('申请任务已成功提交微信！微信已向法人微信号发送人脸核身通知')
      queryClient.invalidateQueries({ queryKey: ['miniprogram-tasks'] })
      setCreateModalOpen(false)
      setCorpName('')
      setCorpCode('')
      setLegalWechat('')
      setLegalName('')
    },
    onError: (err: any) => {
      toast.error('提交失败: ' + (err.message || '微信接口报错'))
    },
  })

  // 刷新查询单个任务进度
  const refreshTaskMutation = useMutation({
    mutationFn: (id: number) => api.post(`/api/miniprogram/tasks/${id}/refresh`),
    onSuccess: (data: any) => {
      if (data?.status === 1) {
        toast.success(`🎉 小程序开通成功！AppID: ${data.appid}，已自动同步至系统！`)
        queryClient.invalidateQueries({ queryKey: ['accounts'] })
      } else if (data?.status === 2) {
        toast.error(`微信审核已驳回: ${data.reject_reason || data.status_desc}`)
      } else {
        toast.info(data?.status_desc || '当前任务正在等待法人人脸核身')
      }
      queryClient.invalidateQueries({ queryKey: ['miniprogram-tasks'] })
    },
  })

  // 保存服务器域名配置
  const saveDomainMutation = useMutation({
    mutationFn: (values: any) => api.post('/api/miniprogram/domain', values),
    onSuccess: () => {
      toast.success('小程序服务器合法域名已成功同步配置至微信官方！')
      queryClient.invalidateQueries({ queryKey: ['miniprogram-domain'] })
    },
    onError: (err: any) => {
      toast.error('配置失败: ' + (err.message || '微信接口报错'))
    },
  })

  // 关联小程序
  const linkMutation = useMutation({
    mutationFn: (values: any) => api.post('/api/miniprogram/link', values),
    onSuccess: (data: any) => {
      toast.success(data?.message || '关联申请已提交微信')
      setLinkModalOpen(false)
      setLinkAppID('')
      queryClient.invalidateQueries({ queryKey: ['miniprogram-linked'] })
    },
  })

  // 解绑小程序
  const unlinkMutation = useMutation({
    mutationFn: (appid: string) => api.post('/api/miniprogram/unlink', { appid }),
    onSuccess: () => {
      toast.success('小程序已成功解绑')
      queryClient.invalidateQueries({ queryKey: ['miniprogram-linked'] })
    },
  })

  // 草稿加入模板库
  const addTemplateMutation = useMutation({
    mutationFn: (values: any) => api.post('/api/miniprogram/templates/add-from-draft', values),
    onSuccess: (data: any) => {
      toast.success(data?.message || '草稿已成功添加至正式模板库')
      queryClient.invalidateQueries({ queryKey: ['miniprogram-templates'] })
      queryClient.invalidateQueries({ queryKey: ['miniprogram-drafts'] })
    },
  })

  // 删除模板
  const delTemplateMutation = useMutation({
    mutationFn: (id: number) => api.delete(`/api/miniprogram/templates/${id}`),
    onSuccess: () => {
      toast.success('模板已删除')
      queryClient.invalidateQueries({ queryKey: ['miniprogram-templates'] })
    },
  })

  // 测试 code2session
  const handleTestLogin = async () => {
    if (!currentApp?.appid) {
      toast.error('请在顶部先选择一个小程序账号')
      return
    }
    if (!jsCode.trim()) {
      toast.error('请输入 wx.login 返回的临时 code')
      return
    }
    try {
      setTestingLogin(true)
      const res: any = await api.post('/api/miniprogram/code2session', {
        appid: currentApp.appid,
        js_code: jsCode.trim(),
      })
      setLoginResult(res.data)
      toast.success('免密换取 session 成功！')
    } catch (e: any) {
      toast.error('换取失败: ' + (e.message || 'code 无效或已过期'))
    } finally {
      setTestingLogin(false)
    }
  }

  const taskTotal = taskResult?.pagination?.total || 0
  const taskTotalPages = Math.max(1, Math.ceil(taskTotal / 10))
  const linkedList: any[] = linkedRes?.items || []

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-bold tracking-tight flex items-center gap-2">
            <span>微信小程序代开发与快速注册</span>
            <Badge variant="success" className="text-xs">免 300 元认证费通道</Badge>
          </h2>
          <p className="text-sm text-muted-foreground">
            快速免密开通小程序、公众号维度关联小程序、开放平台代码模板库/草稿箱、服务器域名代配置及免密登录网关
          </p>
        </div>

        {activeTab === 'register' && (
          <Button onClick={() => setCreateModalOpen(true)} className="gap-2 bg-emerald-600 hover:bg-emerald-700">
            <Plus className="h-4 w-4" />
            <span>快速创建小程序 (免认证费)</span>
          </Button>
        )}
        {activeTab === 'linked' && (
          <div className="flex gap-2">
            <Button variant="outline" onClick={() => refetchLinked()} className="gap-2">
              <RefreshCw className={`h-4 w-4 ${fetchingLinked ? 'animate-spin' : ''}`} />
              <span>刷新</span>
            </Button>
            <Button onClick={() => setLinkModalOpen(true)} className="gap-2 bg-sky-600 hover:bg-sky-700">
              <Link2 className="h-4 w-4" />
              <span>关联新小程序</span>
            </Button>
          </div>
        )}
        {activeTab === 'templates' && (
          <Button
            variant="outline"
            className="gap-2"
            onClick={() =>
              tplTab === 'templates' ? refetchTemplates() : refetchDrafts()
            }
          >
            <RefreshCw
              className={`h-4 w-4 ${tplTab === 'templates' ? fetchingTemplates : fetchingDrafts} ${
                (tplTab === 'templates' ? fetchingTemplates : fetchingDrafts) ? 'animate-spin' : ''
              }`}
            />
            <span>刷新</span>
          </Button>
        )}
      </div>

      {/* Tabs 分组 */}
      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <TabsList className="flex flex-wrap max-w-none">
          {MINI_TABS.map((t) => {
            const Icon = t.icon
            return (
              <TabsTrigger key={t.value} value={t.value} className="text-xs gap-1.5">
                <Icon className={`h-3.5 w-3.5 ${t.color}`} />
                <span>{t.label}</span>
              </TabsTrigger>
            )
          })}
        </TabsList>

        {/* 标签页 1：快速注册小程序任务列表 */}
        <TabsContent value="register" className="mt-6 space-y-6">
          {/* 政策说明卡片 */}
          <div className="p-4 rounded-xl bg-emerald-500/10 border border-emerald-500/20 text-xs text-emerald-900 dark:text-emerald-200 flex items-start gap-3">
            <ShieldCheck className="h-5 w-5 text-emerald-600 shrink-0 mt-0.5" />
            <div className="space-y-1 leading-relaxed">
              <div className="font-semibold text-sm">💡 微信开放平台快速注册小程序规则说明:</div>
              <div>
                1. <strong>免收 300 元认证费</strong>：依托第三方平台资质，企业与个体工商户无需向腾讯支付认证审核费。<br />
                2. <strong>法人微信人脸核验</strong>：提交后微信会向填写的【法人微信号】下发模板消息，法人 24 小时内点击人脸刷脸即可通过。<br />
                3. <strong>开通自动代管</strong>：法人刷脸通过后，微信将全自动生成小程序 AppID 并自动授权托管至本系统，开箱即用。
              </div>
            </div>
          </div>

          {/* 任务表格 */}
          <Card>
            <CardHeader className="pb-3">
              <CardTitle className="text-base">注册任务申请记录</CardTitle>
              <CardDescription className="text-xs">实时追踪法人人脸认证状态与微信下发的 AppID</CardDescription>
            </CardHeader>
            <CardContent className="p-0">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>企业名称</TableHead>
                    <TableHead>信用代码</TableHead>
                    <TableHead>法人姓名 / 微信号</TableHead>
                    <TableHead>微信审核状态</TableHead>
                    <TableHead>下发小程序 AppID</TableHead>
                    <TableHead>申请时间</TableHead>
                    <TableHead className="text-right">操作</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {loadingTasks ? (
                    <TableRow>
                      <TableCell colSpan={7} className="h-32 text-center text-muted-foreground">加载中...</TableCell>
                    </TableRow>
                  ) : taskResult?.list && taskResult.list.length > 0 ? (
                    taskResult.list.map((task: any) => (
                      <TableRow key={task.id} className="text-xs">
                        <TableCell className="font-semibold">{task.name}</TableCell>
                        <TableCell className="font-mono">{task.code}</TableCell>
                        <TableCell>
                          <span>{task.legal_persona_name}</span>
                          <span className="text-muted-foreground ml-1.5 font-mono">({task.legal_persona_wechat})</span>
                        </TableCell>
                        <TableCell>
                          {task.status === 1 ? (
                            <Badge variant="success" className="gap-1 text-[10px]">
                              <CheckCircle2 className="h-3 w-3" />
                              <span>注册成功</span>
                            </Badge>
                          ) : task.status === 2 ? (
                            <Badge variant="destructive" className="gap-1 text-[10px]" title={task.reject_reason}>
                              <AlertCircle className="h-3 w-3" />
                              <span>已驳回</span>
                            </Badge>
                          ) : (
                            <Badge variant="secondary" className="gap-1 text-[10px] text-amber-600 bg-amber-500/10 border-amber-500/30">
                              <Clock className="h-3 w-3" />
                              <span>待法人刷脸核身</span>
                            </Badge>
                          )}
                        </TableCell>
                        <TableCell className="font-mono font-semibold text-emerald-600">
                          {task.appid || '-'}
                        </TableCell>
                        <TableCell className="text-muted-foreground">{formatDate(task.created_at)}</TableCell>
                        <TableCell className="text-right">
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => refreshTaskMutation.mutate(task.id)}
                            disabled={refreshTaskMutation.isPending}
                            className="h-7 text-xs gap-1"
                          >
                            <RefreshCw className={`h-3 w-3 ${refreshTaskMutation.isPending ? 'animate-spin' : ''}`} />
                            <span>查询微信进度</span>
                          </Button>
                        </TableCell>
                      </TableRow>
                    ))
                  ) : (
                    <TableRow>
                      <TableCell colSpan={7} className="h-32 text-center text-muted-foreground">
                        暂无小程序快速注册记录，点击右上角开始创建
                      </TableCell>
                    </TableRow>
                  )}
                </TableBody>
              </Table>
            </CardContent>
          </Card>

          {taskTotal > 10 && (
            <div className="flex items-center justify-end gap-1">
              <Button
                variant="outline"
                size="sm"
                disabled={page <= 1}
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                className="h-8 px-2.5 text-xs gap-1"
              >
                <ChevronLeft className="h-3.5 w-3.5" />
                <span>上一页</span>
              </Button>
              <span className="px-2 text-xs font-medium">第 {page} / {taskTotalPages} 页 (共 {taskTotal} 条)</span>
              <Button
                variant="outline"
                size="sm"
                disabled={page >= taskTotalPages}
                onClick={() => setPage((p) => Math.min(taskTotalPages, p + 1))}
                className="h-8 px-2.5 text-xs gap-1"
              >
                <span>下一页</span>
                <ChevronRight className="h-3.5 w-3.5" />
              </Button>
            </div>
          )}
        </TabsContent>

        {/* 标签页 2：公众号关联小程序 */}
        <TabsContent value="linked" className="mt-6 space-y-6">
          <div className="p-4 rounded-xl bg-sky-500/10 border border-sky-500/20 text-xs text-sky-900 dark:text-sky-200 flex items-start gap-3">
            <ShieldCheck className="h-5 w-5 text-sky-600 shrink-0 mt-0.5" />
            <div className="space-y-1 leading-relaxed">
              <div className="font-semibold text-sm">ℹ️ 公众号关联小程序 (属于公众号维度，用公众号 access_token 调用):</div>
              <div>
                1. 本功能作用于<strong>顶部当前选中的公众号</strong>
                {currentApp ? (
                  <span className="font-mono"> ({currentApp.name} / {currentApp.appid})</span>
                ) : (
                  <span className="text-destructive">（当前未选中账号，请先在顶部选择公众号）</span>
                )
                }。<br />
                2. 一个公众号最多可关联 <strong>10 个</strong>小程序；关联需小程序管理员在微信内确认。<br />
                3. <strong>show_profile = 1</strong> 时，该小程序会在公众号资料页 (我的-关注公众号) 中展示入口。
              </div>
            </div>
          </div>

          <Card>
            <CardHeader className="pb-3">
              <CardTitle className="text-base flex items-center gap-2">
                <Link2 className="h-4 w-4 text-sky-600" />
                <span>已关联的小程序列表</span>
                {linkedList.length > 0 && <Badge variant="outline" className="text-[10px]">{linkedList.length} 个</Badge>}
              </CardTitle>
              <CardDescription className="text-xs">数据来源：微信官方 cgi-bin/wxopen/wxamplinkget 实时接口</CardDescription>
            </CardHeader>
            <CardContent className="p-0">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="w-[72px]">头像</TableHead>
                    <TableHead>小程序名称</TableHead>
                    <TableHead>AppID</TableHead>
                    <TableHead>原始 ID</TableHead>
                    <TableHead>关联状态</TableHead>
                    <TableHead>公众号资料页展示</TableHead>
                    <TableHead>绑定邮箱</TableHead>
                    <TableHead className="text-right">操作</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {loadingLinked ? (
                    <TableRow>
                      <TableCell colSpan={8} className="h-32 text-center text-muted-foreground">正在向微信官方调取关联列表...</TableCell>
                    </TableRow>
                  ) : linkedList.length > 0 ? (
                    linkedList.map((mp: any) => {
                      const st = LINK_STATUS[mp.status] || { label: `状态码 ${mp.status}`, cls: '' }
                      return (
                        <TableRow key={mp.appid || mp.username} className="text-xs">
                          <TableCell>
                            <div className="h-10 w-10 rounded-lg overflow-hidden border bg-muted">
                              {mp.headimg_url ? (
                                <img src={mp.headimg_url} referrerPolicy="no-referrer" alt="" className="h-full w-full object-cover" />
                              ) : (
                                <div className="h-full w-full flex items-center justify-center text-[10px] text-muted-foreground">无</div>
                              )}
                            </div>
                          </TableCell>
                          <TableCell className="font-semibold">{mp.nickname || '-'}</TableCell>
                          <TableCell className="font-mono">{mp.appid || '-'}</TableCell>
                          <TableCell className="font-mono text-muted-foreground">{mp.username || '-'}</TableCell>
                          <TableCell>
                            <Badge variant="outline" className={`text-[10px] ${st.cls}`}>{st.label}</Badge>
                          </TableCell>
                          <TableCell>
                            {mp.selected === 1 ? (
                              <span className="inline-flex items-center gap-1 text-emerald-600 font-medium">
                                <Star className="h-3.5 w-3.5 fill-current" />
                                <span>已展示</span>
                              </span>
                            ) : (
                              <span className="text-muted-foreground">未展示</span>
                            )}
                          </TableCell>
                          <TableCell className="text-muted-foreground">{mp.email || '-'}</TableCell>
                          <TableCell className="text-right">
                            <Button
                              variant="outline"
                              size="sm"
                              className="h-7 text-xs gap-1 text-red-600 hover:text-red-700"
                              disabled={unlinkMutation.isPending}
                              onClick={() => {
                                if (!window.confirm(`确认解除公众号与小程序「${mp.nickname || mp.appid}」的关联？`)) return
                                unlinkMutation.mutate(mp.appid)
                              }}
                            >
                              <Unlink className="h-3 w-3" />
                              <span>解绑</span>
                            </Button>
                          </TableCell>
                        </TableRow>
                      )
                    })
                  ) : (
                    <TableRow>
                      <TableCell colSpan={8} className="h-32 text-center text-muted-foreground">
                        当前公众号还没有关联任何小程序，点击右上角「关联新小程序」开始
                      </TableCell>
                    </TableRow>
                  )}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        </TabsContent>

        {/* 标签页 3：代码模板库 / 草稿箱 */}
        <TabsContent value="templates" className="mt-6 space-y-6">
          <div className="p-4 rounded-xl bg-violet-500/10 border border-violet-500/20 text-xs text-violet-900 dark:text-violet-200 flex items-start gap-3">
            <ShieldCheck className="h-5 w-5 text-violet-600 shrink-0 mt-0.5" />
            <div className="space-y-1 leading-relaxed">
              <div className="font-semibold text-sm">📦 小程序代码模板库 (属于小程序，与公众号无关):</div>
              <div>
                开发者在使用<strong>小程序开发者工具</strong>上传代码后，代码会先进入<strong>草稿箱</strong>；
                将草稿「加入模板库」得到 template_id，之后即可用该模板为已授权的小程序批量提审 / 发布。<br />
                <strong>归属说明</strong>：草稿箱与代码模板库是<strong>微信开放平台 (第三方平台) 账号级别的全局资源</strong>，
                使用 <code className="font-mono">component_access_token</code> 调用，<strong>不隶属于任何公众号，也不隶属于单个小程序</strong>，
                因此本标签页无需在顶部选择账号。
              </div>
            </div>
          </div>

          <Tabs value={tplTab} onValueChange={(v: any) => setTplTab(v)} className="w-full">
            <TabsList className="grid w-full max-w-xs grid-cols-2">
              <TabsTrigger value="templates" className="text-xs gap-1.5">
                <Library className="h-3.5 w-3.5" />
                <span>正式模板库</span>
              </TabsTrigger>
              <TabsTrigger value="drafts" className="text-xs gap-1.5">
                <Sparkles className="h-3.5 w-3.5" />
                <span>代码草稿箱</span>
              </TabsTrigger>
            </TabsList>

            {/* 正式模板库 */}
            <TabsContent value="templates" className="mt-4">
              <Card>
                <CardHeader className="pb-3 flex flex-row items-start justify-between gap-3 space-y-0">
                  <div>
                    <CardTitle className="text-base">代码模板库</CardTitle>
                    <CardDescription className="text-xs">用于「代码提交 (commit)」与「发布」的模板版本仓库</CardDescription>
                  </div>
                  <select
                    value={tplType}
                    onChange={(e) => setTplType(e.target.value)}
                    className="h-8 rounded-lg border border-input bg-background px-2 text-xs"
                  >
                    <option value="0">全部类型</option>
                    <option value="1">仅标准模版</option>
                  </select>
                </CardHeader>
                <CardContent className="p-0">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>模板 ID</TableHead>
                        <TableHead>版本号</TableHead>
                        <TableHead>模板描述</TableHead>
                        <TableHead>类型</TableHead>
                        <TableHead>开发者</TableHead>
                        <TableHead>创建时间</TableHead>
                        <TableHead className="text-right">操作</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {loadingTemplates ? (
                        <TableRow>
                          <TableCell colSpan={7} className="h-32 text-center text-muted-foreground">加载中...</TableCell>
                        </TableRow>
                      ) : templates && templates.length > 0 ? (
                        templates.map((t: any) => (
                          <TableRow key={t.template_id} className="text-xs">
                            <TableCell className="font-mono font-semibold">{t.template_id}</TableCell>
                            <TableCell><Badge variant="secondary" className="text-[10px]">{t.user_version}</Badge></TableCell>
                            <TableCell className="max-w-[280px] truncate" title={t.user_desc}>{t.user_desc}</TableCell>
                            <TableCell>{t.template_type === 1 ? '标准模版' : '普通模版'}</TableCell>
                            <TableCell className="text-muted-foreground">{t.developer || '-'}</TableCell>
                            <TableCell className="text-muted-foreground">{formatUnixSec(t.create_time)}</TableCell>
                            <TableCell className="text-right">
                              <Button
                                variant="outline"
                                size="sm"
                                className="h-7 text-xs gap-1 text-red-600 hover:text-red-700"
                                disabled={delTemplateMutation.isPending}
                                onClick={() => {
                                  if (!window.confirm(`确认删除模板「${t.user_desc}」(ID: ${t.template_id})？删除后不可恢复`)) return
                                  delTemplateMutation.mutate(t.template_id)
                                }}
                              >
                                <Trash2 className="h-3 w-3" />
                                <span>删除</span>
                              </Button>
                            </TableCell>
                          </TableRow>
                        ))
                      ) : (
                        <TableRow>
                          <TableCell colSpan={7} className="h-32 text-center text-muted-foreground">
                            模板库为空，请先到「代码草稿箱」将草稿加入模板库
                          </TableCell>
                        </TableRow>
                      )}
                    </TableBody>
                  </Table>
                </CardContent>
              </Card>
            </TabsContent>

            {/* 草稿箱 */}
            <TabsContent value="drafts" className="mt-4">
              <Card>
                <CardHeader className="pb-3">
                  <CardTitle className="text-base">代码草稿箱</CardTitle>
                  <CardDescription className="text-xs">开发者工具上传的原始代码包，可直接转为正式模板</CardDescription>
                </CardHeader>
                <CardContent className="p-0">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>草稿 ID</TableHead>
                        <TableHead>版本号</TableHead>
                        <TableHead>草稿说明</TableHead>
                        <TableHead>类型</TableHead>
                        <TableHead>开发者</TableHead>
                        <TableHead>上传时间</TableHead>
                        <TableHead className="text-right">操作</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {loadingDrafts ? (
                        <TableRow>
                          <TableCell colSpan={7} className="h-32 text-center text-muted-foreground">加载中...</TableCell>
                        </TableRow>
                      ) : drafts && drafts.length > 0 ? (
                        drafts.map((d: any) => (
                          <TableRow key={d.draft_id} className="text-xs">
                            <TableCell className="font-mono font-semibold">{d.draft_id}</TableCell>
                            <TableCell><Badge variant="secondary" className="text-[10px]">{d.user_version}</Badge></TableCell>
                            <TableCell className="max-w-[280px] truncate" title={d.user_desc}>{d.user_desc}</TableCell>
                            <TableCell>{d.template_type === 1 ? '标准模版' : '普通模版'}</TableCell>
                            <TableCell className="text-muted-foreground">{d.developer || '-'}</TableCell>
                            <TableCell className="text-muted-foreground">{formatUnixSec(d.create_time)}</TableCell>
                            <TableCell className="text-right">
                              <Button
                                variant="outline"
                                size="sm"
                                className="h-7 text-xs gap-1"
                                disabled={addTemplateMutation.isPending}
                                onClick={() =>
                                  addTemplateMutation.mutate({
                                    draft_id: d.draft_id,
                                    template_type: d.template_type === 1 ? 1 : 0,
                                  })
                                }
                              >
                                <Plus className="h-3 w-3" />
                                <span>加入模板库</span>
                              </Button>
                            </TableCell>
                          </TableRow>
                        ))
                      ) : (
                        <TableRow>
                          <TableCell colSpan={7} className="h-32 text-center text-muted-foreground">
                            草稿箱暂无内容，请先在微信开发者工具中上传代码到第三方平台
                          </TableCell>
                        </TableRow>
                      )}
                    </TableBody>
                  </Table>
                </CardContent>
              </Card>
            </TabsContent>
          </Tabs>
        </TabsContent>

        {/* 标签页 4：代配置服务器域名 */}
        <TabsContent value="domain" className="mt-6 space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="text-base flex items-center gap-2">
                <Globe className="h-5 w-5 text-blue-600" />
                <span>小程序服务器合法域名配置</span>
              </CardTitle>
              <CardDescription className="text-xs">
                无需登录微信公众平台后台，直接代当前选中的小程序配置合法网络请求、Socket 与上传下载域名
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4 text-xs">
              <div className="space-y-1.5">
                <label className="font-semibold text-foreground">request 合法域名 (多条可用逗号或换行分隔) *</label>
                <Input
                  value={reqDomain}
                  onChange={(e) => setReqDomain(e.target.value)}
                  placeholder="https://api.yourcompany.com"
                  className="font-mono text-xs"
                />
              </div>

              <div className="space-y-1.5">
                <label className="font-semibold text-foreground">socket 合法域名 (wss 协议)</label>
                <Input
                  value={wsDomain}
                  onChange={(e) => setWsDomain(e.target.value)}
                  placeholder="wss://api.yourcompany.com"
                  className="font-mono text-xs"
                />
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1.5">
                  <label className="font-semibold text-foreground">uploadFile 合法域名</label>
                  <Input
                    value={uploadDomain}
                    onChange={(e) => setUploadDomain(e.target.value)}
                    placeholder="https://api.yourcompany.com"
                    className="font-mono text-xs"
                  />
                </div>

                <div className="space-y-1.5">
                  <label className="font-semibold text-foreground">downloadFile 合法域名</label>
                  <Input
                    value={downloadDomain}
                    onChange={(e) => setDownloadDomain(e.target.value)}
                    placeholder="https://api.yourcompany.com"
                    className="font-mono text-xs"
                  />
                </div>
              </div>

              <div className="space-y-1.5">
                <label className="font-semibold text-foreground">业务域名 webview (内嵌网页需配置)</label>
                <Input
                  value={webviewDomain}
                  onChange={(e) => setWebviewDomain(e.target.value)}
                  placeholder="https://h5.yourcompany.com"
                  className="font-mono text-xs"
                />
              </div>

              <div className="pt-2 flex justify-end">
                <Button
                  onClick={() =>
                    saveDomainMutation.mutate({
                      action: 'set',
                      requestdomain: reqDomain,
                      wsrequestdomain: wsDomain,
                      uploaddomain: uploadDomain,
                      downloaddomain: downloadDomain,
                      webviewdomain: webviewDomain,
                    })
                  }
                  disabled={saveDomainMutation.isPending}
                  className="gap-2 bg-emerald-600 hover:bg-emerald-700"
                >
                  <Save className="h-4 w-4" />
                  <span>{saveDomainMutation.isPending ? '保存中...' : '同步配置至微信官方'}</span>
                </Button>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* 标签页 5：免密登录网关 Code2Session */}
        <TabsContent value="code2session" className="mt-6 space-y-6">
          <Card>
            <CardHeader>
              <CardTitle className="text-base flex items-center gap-2">
                <Key className="h-5 w-5 text-emerald-600" />
                <span>微信小程序免密登录网关 (Code2Session)</span>
              </CardTitle>
              <CardDescription className="text-xs">
                第三方平台代管的小程序无需在客户端暴露 AppSecret，前端或子系统直接调用网关即可换取 OpenID 和 SessionKey
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4 text-xs">
              <div className="p-3 bg-muted/60 rounded-xl space-y-1.5 font-mono text-[11px]">
                <div className="font-sans font-semibold text-foreground text-xs">移动端小程序接入接口:</div>
                <div className="bg-background p-2 rounded select-all border text-primary">
                  POST {typeof window !== 'undefined' ? window.location.origin : ''}/api/miniprogram/code2session
                </div>
                <div className="text-muted-foreground text-[10.5px]">
                  入参 JSON: <code>{`{"appid": "${currentApp?.appid || 'wx...'} ", "js_code": "091abc..."}`}</code>
                </div>
              </div>

              <div className="space-y-1.5">
                <label className="font-semibold text-foreground">在线模拟登录测试 (输入 wx.login 获取的临时 js_code):</label>
                <div className="flex gap-2">
                  <Input
                    value={jsCode}
                    onChange={(e) => setJsCode(e.target.value)}
                    placeholder="输入微信返回的 32 位 js_code"
                    className="font-mono text-xs flex-1"
                  />
                  <Button size="sm" onClick={handleTestLogin} disabled={testingLogin} className="gap-1.5">
                    <Send className="h-3.5 w-3.5" />
                    <span>{testingLogin ? '请求中...' : '测试换取 Session'}</span>
                  </Button>
                </div>
              </div>

              {loginResult && (
                <div className="space-y-1 p-3 bg-muted/60 rounded-xl border font-mono text-[11px]">
                  <div className="font-sans font-semibold text-emerald-600 flex items-center gap-1 text-xs">
                    <CheckCircle2 className="h-3.5 w-3.5" />
                    <span>换取成功响应数据:</span>
                  </div>
                  <div>OpenID: {loginResult.openid}</div>
                  <div>SessionKey: {loginResult.session_key ? loginResult.session_key.slice(0, 8) + '...' : '-'}</div>
                  {loginResult.unionid && <div>UnionID: {loginResult.unionid}</div>}
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      {/* 快速注册小程序申请 Modal */}
      <Dialog open={createModalOpen} onOpenChange={setCreateModalOpen}>
        <DialogContent className="sm:max-w-lg max-h-[85vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>申请快速注册小程序 (免300元认证费)</DialogTitle>
          </DialogHeader>

          <div className="space-y-4 py-2 text-xs">
            <div className="space-y-1.5">
              <label className="font-semibold text-foreground">企业 / 个体工商户全称 (与执照严格一致) *</label>
              <Input
                value={corpName}
                onChange={(e) => setCorpName(e.target.value)}
                placeholder="例如: 腾讯科技（深圳）有限公司"
              />
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1.5">
                <label className="font-semibold text-foreground">企业代码类型</label>
                <select
                  value={codeType}
                  onChange={(e) => setCodeType(e.target.value)}
                  className="w-full h-9 rounded-lg border border-input bg-background px-2.5 text-xs text-foreground"
                >
                  <option value="1">1: 统一社会信用代码 (推荐)</option>
                  <option value="2">2: 组织机构代码</option>
                  <option value="3">3: 营业执照注册号</option>
                </select>
              </div>

              <div className="space-y-1.5">
                <label className="font-semibold text-foreground">企业代码 (18位统一信用代码) *</label>
                <Input
                  value={corpCode}
                  onChange={(e) => setCorpCode(e.target.value)}
                  placeholder="91440300..."
                  className="font-mono text-xs"
                />
              </div>
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1.5">
                <label className="font-semibold text-foreground">法人姓名 (与执照一致) *</label>
                <Input
                  value={legalName}
                  onChange={(e) => setLegalName(e.target.value)}
                  placeholder="例如: 马化腾"
                />
              </div>

              <div className="space-y-1.5">
                <label className="font-semibold text-foreground">法人微信号 (接收人脸核身通知) *</label>
                <Input
                  value={legalWechat}
                  onChange={(e) => setLegalWechat(e.target.value)}
                  placeholder="输入法人的个人微信号"
                  className="font-mono text-xs"
                />
              </div>
            </div>

            <div className="space-y-1.5">
              <label className="font-semibold text-foreground">第三方联系电话</label>
              <Input
                value={phone}
                onChange={(e) => setPhone(e.target.value)}
                placeholder="例如: 13800000000"
                className="font-mono text-xs"
              />
            </div>

            <DialogFooter className="pt-2">
              <Button variant="outline" onClick={() => setCreateModalOpen(false)}>取消</Button>
              <Button
                onClick={() => {
                  if (!corpName.trim() || !corpCode.trim() || !legalName.trim() || !legalWechat.trim()) {
                    toast.error('企业名称、代码、法人姓名与微信号均不能为空')
                    return
                  }
                  createTaskMutation.mutate({
                    name: corpName.trim(),
                    code: corpCode.trim(),
                    code_type: parseInt(codeType),
                    legal_persona_name: legalName.trim(),
                    legal_persona_wechat: legalWechat.trim(),
                    component_phone: phone.trim(),
                  })
                }}
                disabled={createTaskMutation.isPending}
                className="bg-emerald-600 hover:bg-emerald-700"
              >
                {createTaskMutation.isPending ? '正在提交微信审核...' : '立即提交微信申请'}
              </Button>
            </DialogFooter>
          </div>
        </DialogContent>
      </Dialog>

      {/* 关联新小程序 Modal */}
      <Dialog open={linkModalOpen} onOpenChange={setLinkModalOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>为当前公众号关联小程序</DialogTitle>
          </DialogHeader>

          <div className="space-y-4 py-2 text-xs">
            <div className="p-3 rounded-lg bg-muted/60 border space-y-0.5">
              <div className="text-muted-foreground">当前操作的公众号:</div>
              <div className="font-semibold">{currentApp?.name || '未选择'}</div>
              <div className="font-mono text-muted-foreground">{currentApp?.appid || '-'}</div>
            </div>

            <div className="space-y-1.5">
              <label className="font-semibold text-foreground">小程序 AppID *</label>
              <Input
                value={linkAppID}
                onChange={(e) => setLinkAppID(e.target.value)}
                placeholder="wx 开头的小程序 AppID"
                className="font-mono text-xs"
              />
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1.5">
                <label className="font-semibold text-foreground">发送通知给小程序管理员</label>
                <select
                  value={notifyUsers}
                  onChange={(e) => setNotifyUsers(e.target.value)}
                  className="w-full h-9 rounded-lg border border-input bg-background px-2.5 text-xs"
                >
                  <option value="1">1: 发送 (推荐)</option>
                  <option value="0">0: 不发送</option>
                </select>
              </div>
              <div className="space-y-1.5">
                <label className="font-semibold text-foreground">在公众号资料页展示</label>
                <select
                  value={showProfile}
                  onChange={(e) => setShowProfile(e.target.value)}
                  className="w-full h-9 rounded-lg border border-input bg-background px-2.5 text-xs"
                >
                  <option value="0">0: 不展示</option>
                  <option value="1">1: 展示</option>
                </select>
              </div>
            </div>

            <div className="p-3 rounded-lg bg-sky-500/10 border border-sky-500/20 text-sky-900 dark:text-sky-200 leading-relaxed">
              提交后微信会向小程序管理员下发确认通知，对方在微信中同意后方能完成关联。
            </div>

            <DialogFooter className="pt-2">
              <Button variant="outline" onClick={() => setLinkModalOpen(false)}>取消</Button>
              <Button
                onClick={() => {
                  if (!currentApp?.appid) {
                    toast.error('请先在顶部选择一个微信公众号')
                    return
                  }
                  if (!linkAppID.trim()) {
                    toast.error('小程序 AppID 不能为空')
                    return
                  }
                  linkMutation.mutate({
                    appid: linkAppID.trim(),
                    notify_users: parseInt(notifyUsers),
                    show_profile: parseInt(showProfile),
                  })
                }}
                disabled={linkMutation.isPending}
                className="bg-sky-600 hover:bg-sky-700 gap-1.5"
              >
                <Link2 className="h-4 w-4" />
                <span>{linkMutation.isPending ? '提交中...' : '提交关联申请'}</span>
              </Button>
            </DialogFooter>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  )
}
