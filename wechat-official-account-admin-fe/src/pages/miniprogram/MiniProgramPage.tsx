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
  AppWindow,
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
} from 'lucide-react'
import { toast } from 'sonner'
import { formatDate } from '@/lib/utils'

export const MiniProgramPage: React.FC = () => {
  const queryClient = useQueryClient()
  const { currentApp } = useAppStore()

  const [activeTab, setActiveTab] = useState<'register' | 'domain' | 'code2session'>('register')
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

  // 获取快速注册任务列表
  const { data: taskResult, isLoading: loadingTasks } = useQuery<PageResult<any>>({
    queryKey: ['miniprogram-tasks', page],
    queryFn: () => api.get(`/api/miniprogram/tasks?page=${page}&page_size=10`),
  })

  // 获取小程序域名配置
  const { data: domainConfig } = useQuery<any>({
    queryKey: ['miniprogram-domain', currentApp?.appid],
    queryFn: () => api.get('/api/miniprogram/domain'),
    enabled: !!currentApp,
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

  // 测试 code2session
  const handleTestLogin = async () => {
    if (!currentApp?.appid) {
      toast.error('请在顶部先选择一个小程序账号')
      return
    }
    if (!jsCode.trim()) {
      toast.error('请输入 wx.login 获取到的 js_code')
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

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-bold tracking-tight flex items-center gap-2">
            <span>微信小程序代开发与快速注册</span>
            <Badge variant="success" className="text-xs">免 300 元认证费通道</Badge>
          </h2>
          <p className="text-sm text-muted-foreground">
            企业/个体工商户快速免密开通独立小程序、法人微信人脸核验跟踪、服务器合法域名代配置及登录网关
          </p>
        </div>

        {activeTab === 'register' && (
          <Button onClick={() => setCreateModalOpen(true)} className="gap-2 bg-emerald-600 hover:bg-emerald-700">
            <Plus className="h-4 w-4" />
            <span>快速创建小程序 (免认证费)</span>
          </Button>
        )}
      </div>

      {/* Tabs 分组 */}
      <Tabs value={activeTab} onValueChange={(v: any) => setActiveTab(v)}>
        <TabsList className="grid grid-cols-3 max-w-md">
          <TabsTrigger value="register" className="text-xs gap-1.5">
            <Sparkles className="h-3.5 w-3.5 text-amber-500" />
            <span>快速注册任务</span>
          </TabsTrigger>
          <TabsTrigger value="domain" className="text-xs gap-1.5">
            <Globe className="h-3.5 w-3.5 text-blue-500" />
            <span>合法域名代配置</span>
          </TabsTrigger>
          <TabsTrigger value="code2session" className="text-xs gap-1.5">
            <Key className="h-3.5 w-3.5 text-emerald-500" />
            <span>登录网关 (Code2Session)</span>
          </TabsTrigger>
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
                              <Clock className="h-3 w-3 animate-spin" />
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
        </TabsContent>

        {/* 标签页 2：代配置服务器域名 */}
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

        {/* 标签页 3：免密登录网关 Code2Session */}
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
    </div>
  )
}
