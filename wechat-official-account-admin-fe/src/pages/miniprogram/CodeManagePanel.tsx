import React, { useMemo, useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import {
  Table, TableHeader, TableBody, TableHead, TableRow, TableCell,
} from '@/components/ui/table'
import {
  Rocket, Upload, RefreshCw, Trash2, Plus, QrCode, Users, ShieldCheck,
  CheckCircle2, AlertCircle, Clock, RotateCcw, Send, Undo2, Globe2, Save, Info,
} from 'lucide-react'
import { toast } from 'sonner'
import { formatDate } from '@/lib/utils'

/** 微信提审状态字典 (get_auditstatus / get_latest_auditstatus) */
const AUDIT_STATUS: Record<number, { label: string; cls: string; icon: any }> = {
  0: { label: '审核通过 (已发布)', cls: 'bg-emerald-500/10 text-emerald-600 border-emerald-500/30', icon: CheckCircle2 },
  1: { label: '审核失败', cls: 'bg-red-500/10 text-red-600 border-red-500/30', icon: AlertCircle },
  2: { label: '待审核', cls: 'bg-amber-500/10 text-amber-600 border-amber-500/30', icon: Clock },
  3: { label: '审核成功，待发布', cls: 'bg-blue-500/10 text-blue-600 border-blue-500/30', icon: CheckCircle2 },
  4: { label: '审核中', cls: 'bg-amber-500/10 text-amber-600 border-amber-500/30', icon: Clock },
  5: { label: '撤回中', cls: 'bg-muted text-muted-foreground border-border', icon: Undo2 },
  6: { label: '已撤回', cls: 'bg-muted text-muted-foreground border-border', icon: Undo2 },
  7: { label: '审核延后', cls: 'bg-amber-500/10 text-amber-600 border-amber-500/30', icon: Clock },
}

const ACCOUNT_TYPE_LABEL: Record<string, string> = {
  miniprogram: '小程序',
  authorized: '第三方代管',
  direct: '直连公众号',
}

/** 秒级时间戳格式化 (微信返回的均为 Unix 秒) */
function fmtTs(sec?: number) {
  if (!sec) return '-'
  return formatDate(new Date(sec * 1000))
}

const emptyItem = {
  address: '', first_id: '', second_id: '', third_id: '',
  first_class: '', second_class: '', third_class: '', tag: '', title: '',
}

export const CodeManagePanel: React.FC = () => {
  const queryClient = useQueryClient()
  const [targetApp, setTargetApp] = useState('')
  const [subTab, setSubTab] = useState('overview')
  const ready = !!targetApp

  // 所有请求统一携带 appid，后端 TenantAppID 中间件会强制校验该小程序归属当前租户
  const withApp = useMemo(
    () => (path: string, params?: Record<string, string | number>) => {
      const q = new URLSearchParams()
      if (targetApp) q.set('appid', targetApp)
      Object.entries(params || {}).forEach(([k, v]) => q.set(k, String(v)))
      return `${path}?${q.toString()}`
    },
    [targetApp],
  )

  const refresh = () => queryClient.invalidateQueries({ queryKey: ['mini-code', targetApp] })

  /* --------------------------- 账号候选列表 --------------------------- */
  const { data: apps } = useQuery<any[]>({
    queryKey: ['miniprogram-apps'],
    queryFn: () => api.get('/api/miniprogram/apps'),
  })

  React.useEffect(() => {
    if (!targetApp && apps?.length) {
      const mp = apps.find((a: any) => a.account_type === 'miniprogram')
      if (mp) setTargetApp(mp.appid)
    }
  }, [apps, targetApp])

  /* ------------------------------ 查询 ------------------------------ */
  const { data: latestAudit, isFetching: fetchingAudit } = useQuery<any>({
    queryKey: ['mini-code', targetApp, 'latest-audit'],
    queryFn: () => api.get(withApp('/api/miniprogram/code/latest-audit')),
    enabled: ready,
  })

  const { data: visitStatus } = useQuery<any>({
    queryKey: ['mini-code', targetApp, 'visit-status'],
    queryFn: () => api.get(withApp('/api/miniprogram/code/visit-status')),
    enabled: ready,
  })

  const { data: supportVer } = useQuery<any>({
    queryKey: ['mini-code', targetApp, 'support-version'],
    queryFn: () => api.get(withApp('/api/miniprogram/code/support-version')),
    enabled: ready,
  })

  const { data: templates } = useQuery<any[]>({
    queryKey: ['miniprogram-templates', 'all'],
    queryFn: () => api.get('/api/miniprogram/templates?template_type=0'),
    enabled: ready && subTab === 'commit',
  })

  const { data: codePages } = useQuery<any>({
    queryKey: ['mini-code', targetApp, 'pages'],
    queryFn: () => api.get(withApp('/api/miniprogram/code/pages')),
    enabled: ready && subTab === 'audit',
  })

  const { data: privacyInfo, refetch: refetchPrivacy } = useQuery<any>({
    queryKey: ['mini-code', targetApp, 'privacy-info'],
    queryFn: () => api.get(withApp('/api/miniprogram/code/privacy-info')),
    enabled: ready && subTab === 'audit',
  })

  const { data: categories } = useQuery<any>({
    queryKey: ['mini-code', targetApp, 'categories'],
    queryFn: () => api.get(withApp('/api/miniprogram/categories')),
    enabled: ready && subTab === 'category',
  })

  const { data: testers } = useQuery<any>({
    queryKey: ['mini-code', targetApp, 'testers'],
    queryFn: () => api.get(withApp('/api/miniprogram/testers')),
    enabled: ready && subTab === 'category',
  })

  const { data: historyVersions } = useQuery<any>({
    queryKey: ['mini-code', targetApp, 'versions'],
    queryFn: () => api.get(withApp('/api/miniprogram/code/versions')),
    enabled: ready && subTab === 'release',
  })

  /* --------------------------- 局部表单状态 --------------------------- */
  const [templateId, setTemplateId] = useState('')
  const [userVersion, setUserVersion] = useState('')
  const [userDesc, setUserDesc] = useState('')
  const [extJson, setExtJson] = useState('')
  const [showAdvanced, setShowAdvanced] = useState(false)

  const [trialPath, setTrialPath] = useState('')
  const [trialQr, setTrialQr] = useState<string>('')

  const [auditItems, setAuditItems] = useState<any[]>([emptyItem])
  const [versionDesc, setVersionDesc] = useState('')
  const [feedbackInfo, setFeedbackInfo] = useState('')
  const [queryAuditId, setQueryAuditId] = useState('')
  const [privacyNotUse, setPrivacyNotUse] = useState(false)
  const [auditDetail, setAuditDetail] = useState<any>(null)

  const [catFirst, setCatFirst] = useState('')
  const [catSecond, setCatSecond] = useState('')
  const [catCertKey, setCatCertKey] = useState('')
  const [catCertValue, setCatCertValue] = useState('')

  const [testerWechat, setTesterWechat] = useState('')
  const [newSupportVer, setNewSupportVer] = useState('')

  const patchItem = (idx: number, key: string, val: string) =>
    setAuditItems((arr) => arr.map((x, i) => (i === idx ? { ...x, [key]: val } : x)))

  /* ------------------------------ 变更 ------------------------------ */
  const okThen = (msg?: string) => (data: any) => {
    toast.success(data?.message || msg || '操作已成功')
    queryClient.invalidateQueries({ queryKey: ['mini-code', targetApp] })
    queryClient.invalidateQueries({ queryKey: ['miniprogram-apps'] })
  }
  const failThen = (e: any) => toast.error(e?.message || '微信接口返回错误')

  const commitMutation = useMutation({
    mutationFn: (v: any) => api.post(withApp('/api/miniprogram/code/commit'), v),
    onSuccess: okThen('代码已成功提交至该小程序'),
    onError: failThen,
  })

  const submitAuditMutation = useMutation({
    mutationFn: (v: any) => api.post(withApp('/api/miniprogram/code/submit-audit'), v),
    onSuccess: (data: any) => {
      toast.success(data?.message || '提审成功')
      setSubTab('release')
      queryClient.invalidateQueries({ queryKey: ['mini-code', targetApp] })
    },
    onError: failThen,
  })

  const undoAuditMutation = useMutation({
    mutationFn: () => api.post(withApp('/api/miniprogram/code/undo-audit'), {}),
    onSuccess: okThen('已提交撤回审核请求'),
    onError: failThen,
  })

  const releaseMutation = useMutation({
    mutationFn: () => api.post(withApp('/api/miniprogram/code/release'), {}),
    onSuccess: okThen('小程序已成功全量发布上线'),
    onError: failThen,
  })

  const revertMutation = useMutation({
    mutationFn: (appVersion: string) => api.post(withApp('/api/miniprogram/code/revert'), { app_version: appVersion }),
    onSuccess: okThen('线上版本回退指令已提交至微信'),
    onError: failThen,
  })

  const visitMutation = useMutation({
    mutationFn: (action: 'open' | 'close') => api.post(withApp('/api/miniprogram/code/visit-status'), { action }),
    onSuccess: okThen('服务可访问状态已更新'),
    onError: failThen,
  })

  const supportMutation = useMutation({
    mutationFn: (version: string) => api.post(withApp('/api/miniprogram/code/support-version'), { version }),
    onSuccess: okThen('最低基础库版本已更新'),
    onError: failThen,
  })

  const addCatMutation = useMutation({
    mutationFn: (v: any) => api.post(withApp('/api/miniprogram/categories'), v),
    onSuccess: okThen('类目新增申请已提交，等待微信审核'),
    onError: failThen,
  })

  const delCatMutation = useMutation({
    mutationFn: (v: any) => api.post(withApp('/api/miniprogram/categories/delete'), v),
    onSuccess: okThen('服务类目已删除'),
    onError: failThen,
  })

  const bindMutation = useMutation({
    mutationFn: (wechatid: string) => api.post(withApp('/api/miniprogram/testers/bind'), { wechatid }),
    onSuccess: okThen('体验者绑定成功'),
    onError: failThen,
  })

  const unbindMutation = useMutation({
    mutationFn: (v: any) => api.post(withApp('/api/miniprogram/testers/unbind'), v),
    onSuccess: okThen('体验者已解绑'),
    onError: failThen,
  })

  const needApp = (fn: () => void) => () => {
    if (!ready) {
      toast.error('请先在上方选择要管理代码的小程序')
      return
    }
    fn()
  }

  const auditInfo = AUDIT_STATUS[latestAudit?.status ?? -1]
  const AuditIcon = auditInfo?.icon || Clock

  return (
    <div className="space-y-5">
      {/* 目标小程序选择器 */}
      <Card>
        <CardContent className="pt-5 pb-4 flex flex-col md:flex-row md:items-end gap-3">
          <div className="space-y-1.5 flex-1 min-w-[280px]">
            <label className="text-xs font-semibold">目标小程序 (代码管理仅对小程序账号有效)</label>
            <select
              value={targetApp}
              onChange={(e) => setTargetApp(e.target.value)}
              className="w-full h-9 rounded-lg border border-input bg-background px-2.5 text-xs"
            >
              <option value="">请选择要管理代码的小程序...</option>
              {(apps || []).map((a: any) => (
                <option key={a.appid} value={a.appid}>
                  {a.name} · {a.appid} · {ACCOUNT_TYPE_LABEL[a.account_type] || a.account_type}
                </option>
              ))}
            </select>
          </div>
          <div className="flex items-center gap-2">
            {targetApp && <Badge variant="outline" className="font-mono text-[11px]">{targetApp}</Badge>}
            <Button variant="outline" size="sm" className="gap-1.5 h-9" onClick={refresh} disabled={!ready}>
              <RefreshCw className={`h-3.5 w-3.5 ${fetchingAudit ? 'animate-spin' : ''}`} />
              <span>刷新状态</span>
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* 状态总览 */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
        <Card>
          <CardContent className="pt-4 text-xs space-y-1.5">
            <div className="text-muted-foreground">最近一次提审记录</div>
            {latestAudit ? (
              <>
                <Badge variant="outline" className={`gap-1 ${auditInfo?.cls || ''}`}>
                  <AuditIcon className="h-3 w-3" />
                  <span>{auditInfo?.label || `状态码 ${latestAudit.status}`}</span>
                </Badge>
                <div className="font-semibold">{latestAudit.user_version || '-'} · {latestAudit.user_desc || '无描述'}</div>
                <div className="text-muted-foreground">审核单 {latestAudit.auditid || '-'} · {fmtTs(latestAudit.submit_audit_time)}</div>
                {latestAudit.reason && <div className="text-red-600">原因: {latestAudit.reason}</div>}
              </>
            ) : (
              <div className="text-muted-foreground">{ready ? '暂无提审记录' : '未选择小程序'}</div>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardContent className="pt-4 text-xs space-y-1.5">
            <div className="text-muted-foreground">线上服务可访问状态</div>
            <Badge variant={visitStatus?.status === 1 ? 'success' : visitStatus?.status === 0 ? 'destructive' : 'outline'}>
              {visitStatus?.status === 1 ? '正常对外服务' : visitStatus?.status === 0 ? '已暂停服务' : '未知'}
            </Badge>
            <div className="flex gap-2 pt-1">
              <Button size="sm" variant="outline" className="h-7 text-[11px] gap-1"
                disabled={!ready || visitMutation.isPending}
                onClick={needApp(() => visitMutation.mutate('open'))}>
                <Globe2 className="h-3 w-3" />恢复可见
              </Button>
              <Button size="sm" variant="outline" className="h-7 text-[11px] gap-1"
                disabled={!ready || visitMutation.isPending}
                onClick={needApp(() => visitMutation.mutate('close'))}>
                <AlertCircle className="h-3 w-3" />暂停服务
              </Button>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="pt-4 text-xs space-y-1.5">
            <div className="text-muted-foreground">最低基础库版本</div>
            <div className="font-semibold text-base">{supportVer?.now_version || '-'}</div>
            <div className="flex gap-2">
              <Input value={newSupportVer} onChange={(e) => setNewSupportVer(e.target.value)}
                placeholder="如 2.30.4" className="h-7 text-[11px] font-mono" />
              <Button size="sm" variant="outline" className="h-7 text-[11px] shrink-0"
                disabled={!ready || supportMutation.isPending}
                onClick={needApp(() => {
                  if (!newSupportVer.trim()) { toast.error('请输入要设置的基础库版本号'); return }
                  supportMutation.mutate(newSupportVer.trim())
                })}>
                设置
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>

      <Tabs value={subTab} onValueChange={(v: any) => setSubTab(v)}>
        <TabsList className="flex flex-wrap max-w-none">
          <TabsTrigger value="overview" className="text-xs gap-1.5"><Info className="h-3.5 w-3.5 text-slate-500" />版本详情</TabsTrigger>
          <TabsTrigger value="commit" className="text-xs gap-1.5"><Upload className="h-3.5 w-3.5 text-emerald-500" />提交代码与体验</TabsTrigger>
          <TabsTrigger value="audit" className="text-xs gap-1.5"><Send className="h-3.5 w-3.5 text-blue-500" />提交审核</TabsTrigger>
          <TabsTrigger value="release" className="text-xs gap-1.5"><Rocket className="h-3.5 w-3.5 text-rose-500" />发布与回退</TabsTrigger>
          <TabsTrigger value="category" className="text-xs gap-1.5"><ShieldCheck className="h-3.5 w-3.5 text-violet-500" />类目与体验者</TabsTrigger>
        </TabsList>

        {/* 版本详情 */}
        <TabsContent value="overview" className="mt-5">
          <Card>
            <CardHeader className="pb-3">
              <CardTitle className="text-base">当前版本与审核链路详情</CardTitle>
              <CardDescription className="text-xs">
                数据来源：微信官方 wxa/get_latest_auditstatus、wxa/getvisitstatus、getweappsupportversion
              </CardDescription>
            </CardHeader>
            <CardContent className="text-xs">
              {!ready ? (
                <div className="py-10 text-center text-muted-foreground">请先在上方选择目标小程序</div>
              ) : (
                <div className="grid sm:grid-cols-2 gap-3">
                  <div className="p-3 rounded-lg border bg-muted/40 space-y-1">
                    <div className="text-muted-foreground">代码版本号 (user_version)</div>
                    <div className="font-mono font-semibold">{latestAudit?.user_version || '-'}</div>
                    <div className="text-muted-foreground pt-1">代码描述 (user_desc)</div>
                    <div>{latestAudit?.user_desc || '-'}</div>
                  </div>
                  <div className="p-3 rounded-lg border bg-muted/40 space-y-1">
                    <div className="text-muted-foreground">提审时间</div>
                    <div className="font-semibold">{fmtTs(latestAudit?.submit_audit_time)}</div>
                    <div className="text-muted-foreground pt-1">审核单号</div>
                    <div className="font-mono">{latestAudit?.auditid || '-'}</div>
                  </div>
                  {!!latestAudit?.reason && (
                    <div className="sm:col-span-2 p-3 rounded-lg border border-red-500/30 bg-red-500/5 space-y-1">
                      <div className="text-red-600 font-semibold">微信审核意见</div>
                      <div className="leading-relaxed">{latestAudit.reason}</div>
                      {!!latestAudit.screenshot && (
                        <div className="font-mono text-[10.5px] text-muted-foreground break-all">驳回截图: {latestAudit.screenshot}</div>
                      )}
                    </div>
                  )}
                  {latestAudit?.status === 3 && (
                    <div className="sm:col-span-2 p-3 rounded-lg border border-blue-500/30 bg-blue-500/5 text-blue-700">
                      ✓ 该版本已通过微信审核，可切换到「发布与回退」标签一键全量上线。
                    </div>
                  )}
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        {/* 提交代码 + 体验版 */}
        <TabsContent value="commit" className="mt-5 space-y-5">
          <div className="grid lg:grid-cols-2 gap-5">
            <Card>
              <CardHeader className="pb-3">
                <CardTitle className="text-base flex items-center gap-2"><Upload className="h-4 w-4 text-emerald-600" />提交代码 (commit)</CardTitle>
                <CardDescription className="text-xs">使用代码模板库中的模板为该小程序生成一个新的开发版本</CardDescription>
              </CardHeader>
              <CardContent className="space-y-3 text-xs">
                <div className="space-y-1">
                  <label className="font-semibold">代码模板 *</label>
                  <select value={templateId} onChange={(e) => setTemplateId(e.target.value)}
                    className="w-full h-9 rounded-lg border border-input bg-background px-2.5 text-xs">
                    <option value="">请选择模板 (可在「小程序代码模板库」查看)...</option>
                    {(templates || []).map((t: any) => (
                      <option key={t.template_id} value={t.template_id}>
                        #{t.template_id} · {t.user_version} · {t.user_desc}
                      </option>
                    ))}
                  </select>
                </div>
                <div className="grid grid-cols-2 gap-3">
                  <div className="space-y-1">
                    <label className="font-semibold">代码版本号 *</label>
                    <Input value={userVersion} onChange={(e) => setUserVersion(e.target.value)} placeholder="如 1.0.0" className="h-9 text-xs font-mono" />
                  </div>
                  <div className="space-y-1">
                    <label className="font-semibold">代码描述</label>
                    <Input value={userDesc} onChange={(e) => setUserDesc(e.target.value)} placeholder="本次改动说明" className="h-9 text-xs" />
                  </div>
                </div>

                <button type="button" className="text-[11px] text-primary underline" onClick={() => setShowAdvanced((v) => !v)}>
                  {showAdvanced ? '收起' : '展开'} ext_json 高级配置 (代码模板内含 {'{{extAppid}}'} 占位时必填)
                </button>
                {showAdvanced && (
                  <div className="space-y-1">
                    <label className="font-semibold">ext_json (留空则自动按目标小程序生成)</label>
                    <textarea value={extJson} onChange={(e) => setExtJson(e.target.value)} rows={4}
                      placeholder={'{"extAppid":"wx...","extEnable":true,"ext":{"env":"prod"}}'}
                      className="w-full rounded-lg border border-input bg-background px-2.5 py-2 font-mono text-[11px]" />
                  </div>
                )}

                <div className="flex justify-end pt-1">
                  <Button size="sm" className="gap-1.5 bg-emerald-600 hover:bg-emerald-700"
                    disabled={!ready || commitMutation.isPending}
                    onClick={needApp(() => {
                      if (!templateId) { toast.error('请选择代码模板'); return }
                      if (!userVersion.trim()) { toast.error('代码版本号不能为空'); return }
                      commitMutation.mutate({
                        template_id: Number(templateId),
                        user_version: userVersion.trim(),
                        user_desc: userDesc.trim(),
                        ext_json: extJson.trim(),
                      })
                    })}>
                    <Upload className="h-3.5 w-3.5" />
                    {commitMutation.isPending ? '提交中...' : '提交代码至微信'}
                  </Button>
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="pb-3">
                <CardTitle className="text-base flex items-center gap-2"><QrCode className="h-4 w-4 text-sky-600" />体验版二维码</CardTitle>
                <CardDescription className="text-xs">commit 后即可扫码体验，体验者需在「类目与体验者」中绑定</CardDescription>
              </CardHeader>
              <CardContent className="space-y-3 text-xs">
                <div className="space-y-1">
                  <label className="font-semibold">落地页面 path (可选)</label>
                  <div className="flex gap-2">
                    <Input value={trialPath} onChange={(e) => setTrialPath(e.target.value)} placeholder="留空则默认首页" className="h-9 text-xs font-mono" />
                    <Button size="sm" variant="outline" className="h-9 shrink-0 gap-1.5" disabled={!ready}
                      onClick={needApp(async () => {
                        try {
                          const res: any = await api.get(withApp('/api/miniprogram/code/trial-qrcode', { path: trialPath.trim() }))
                          setTrialQr(res?.image || '')
                          toast.success('体验版二维码已获取')
                        } catch {
                          /* 拦截器已提示 */
                        }
                      })}>
                      <RefreshCw className="h-3.5 w-3.5" />获取二维码
                    </Button>
                  </div>
                </div>

                <div className="min-h-[200px] rounded-xl border bg-muted/30 flex items-center justify-center p-3">
                  {trialQr ? (
                    <div className="space-y-2 text-center">
                      <img src={trialQr} alt="体验版二维码" className="mx-auto h-40 w-40 rounded-lg border bg-white object-contain p-1" />
                      <div className="text-[11px] text-muted-foreground">请使用已绑定的体验者微信扫码进入体验版</div>
                    </div>
                  ) : (
                    <div className="text-[11px] text-muted-foreground text-center leading-relaxed whitespace-pre-line">
                      {ready
                        ? '尚未获取体验版二维码\n请先完成「提交代码 (commit)」，再点击右侧「获取二维码」'
                        : '请先选择目标小程序'}
                    </div>
                  )}
                </div>
              </CardContent>
            </Card>
          </div>
        </TabsContent>

        {/* 提交审核 */}
        <TabsContent value="audit" className="mt-5 space-y-5">
          <Card>
            <CardHeader className="pb-3">
              <CardTitle className="text-base flex items-center gap-2"><Send className="h-4 w-4 text-blue-600" />提交代码审核 (submit_audit)</CardTitle>
              <CardDescription className="text-xs">类目必须已配置且通过审核；页面地址可从当前代码包的页面列表中选择</CardDescription>
            </CardHeader>
            <CardContent className="space-y-3 text-xs">
              {(codePages?.pages || []).length > 0 && (
                <div className="space-y-1">
                  <label className="font-semibold">当前代码包页面列表 ({codePages.pages.length} 个，点击填入第 1 项)</label>
                  <div className="flex flex-wrap gap-1.5 max-h-28 overflow-y-auto p-2 rounded-lg border bg-muted/30">
                    {codePages.pages.map((pg: string) => (
                      <button key={pg} type="button"
                        onClick={() => patchItem(0, 'address', pg)}
                        className="px-2 py-0.5 rounded-md bg-background border text-[11px] font-mono hover:border-primary hover:text-primary">
                        {pg}
                      </button>
                    ))}
                  </div>
                </div>
              )}

              {auditItems.map((it, idx) => (
                <div key={idx} className="p-3 rounded-lg border bg-muted/20 space-y-2">
                  <div className="flex items-center justify-between">
                    <span className="font-semibold">审核项 {idx + 1} (至多 5 项)</span>
                    {auditItems.length > 1 && (
                      <button type="button" className="text-red-500 text-[11px]"
                        onClick={() => setAuditItems((arr) => arr.filter((_, i) => i !== idx))}>
                        移除本项
                      </button>
                    )}
                  </div>
                  <div className="grid sm:grid-cols-2 gap-2">
                    <Input placeholder="页面地址 address" value={it.address} className="h-8 text-[11px] font-mono" onChange={(e) => patchItem(idx, 'address', e.target.value)} />
                    <Input placeholder="页面标题 title" value={it.title} className="h-8 text-[11px]" onChange={(e) => patchItem(idx, 'title', e.target.value)} />
                    <Input placeholder="一级类目 ID first_id *" value={it.first_id} className="h-8 text-[11px] font-mono" onChange={(e) => patchItem(idx, 'first_id', e.target.value)} />
                    <Input placeholder="二级类目 ID second_id *" value={it.second_id} className="h-8 text-[11px] font-mono" onChange={(e) => patchItem(idx, 'second_id', e.target.value)} />
                    <Input placeholder="一级类目名称 first_class" value={it.first_class} className="h-8 text-[11px]" onChange={(e) => patchItem(idx, 'first_class', e.target.value)} />
                    <Input placeholder="二级类目名称 second_class" value={it.second_class} className="h-8 text-[11px]" onChange={(e) => patchItem(idx, 'second_class', e.target.value)} />
                    <Input placeholder="三级类目 ID / 名称 (可选)" value={it.third_id} className="h-8 text-[11px] font-mono" onChange={(e) => patchItem(idx, 'third_id', e.target.value)} />
                    <Input placeholder="标签 tag (空格分隔)" value={it.tag} className="h-8 text-[11px]" onChange={(e) => patchItem(idx, 'tag', e.target.value)} />
                  </div>
                </div>
              ))}

              {auditItems.length < 5 && (
                <Button size="sm" variant="outline" className="h-7 text-[11px] gap-1"
                  onClick={() => setAuditItems((arr) => [...arr, { ...emptyItem }])}>
                  <Plus className="h-3 w-3" />增加审核项
                </Button>
              )}

              <div className="grid sm:grid-cols-2 gap-2 pt-1">
                <Input placeholder="版本说明 version_desc" value={versionDesc} className="h-8 text-[11px]" onChange={(e) => setVersionDesc(e.target.value)} />
                <Input placeholder="反馈给审核人员 feedback_info" value={feedbackInfo} className="h-8 text-[11px]" onChange={(e) => setFeedbackInfo(e.target.value)} />
              </div>

              <div className="flex flex-wrap justify-end gap-2 pt-1">
                <Button size="sm" variant="outline" className="gap-1.5" disabled={!ready || undoAuditMutation.isPending}
                  onClick={needApp(() => undoAuditMutation.mutate())}>
                  <Undo2 className="h-3.5 w-3.5" />撤回当前审核
                </Button>
                <Button size="sm" className="gap-1.5 bg-blue-600 hover:bg-blue-700" disabled={!ready || submitAuditMutation.isPending}
                  onClick={needApp(() => {
                    const items = auditItems
                      .filter((it) => it.address || it.first_id || it.second_id)
                      .map((it) => ({
                        address: it.address || undefined,
                        tag: it.tag || undefined,
                        title: it.title || undefined,
                        first_class: it.first_class || undefined,
                        second_class: it.second_class || undefined,
                        first_id: it.first_id ? Number(it.first_id) : undefined,
                        second_id: it.second_id ? Number(it.second_id) : undefined,
                        third_id: it.third_id ? Number(it.third_id) : undefined,
                      }))
                    if (items.length === 0) { toast.error('请至少填写一个审核项 (页面地址与类目 ID)'); return }
                    submitAuditMutation.mutate({
                      item_list: items,
                      version_desc: versionDesc || undefined,
                      feedback_info: feedbackInfo || undefined,
                      privacy_api_not_use: privacyNotUse || undefined,
                    })
                  })}>
                  <Send className="h-3.5 w-3.5" />
                  {submitAuditMutation.isPending ? '提审中...' : '提交微信审核'}
                </Button>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="pb-3 flex flex-row items-start justify-between gap-3 space-y-0">
              <div>
                <CardTitle className="text-base">代码隐私接口检测 (提审前置检查)</CardTitle>
                <CardDescription className="text-xs">
                  commit 后微信需静态检测代码中的隐私接口，检测未完成就提审会报 61039；无权限则报 61040
                </CardDescription>
              </div>
              <Button size="sm" variant="outline" className="h-7 text-[11px] gap-1 shrink-0" disabled={!ready} onClick={needApp(() => refetchPrivacy())}>
                <RefreshCw className="h-3 w-3" />重新检测
              </Button>
            </CardHeader>
            <CardContent className="text-xs space-y-2">
              {privacyInfo ? (
                <>
                  <div className="flex flex-wrap gap-2">
                    <Badge variant={privacyInfo.without_auth_list?.length ? 'destructive' : 'success'}>
                      无权限接口 {privacyInfo.without_auth_list?.length || 0} 个
                    </Badge>
                    <Badge variant={privacyInfo.without_conf_list?.length ? 'default' : 'success'}>
                      ext.json 未配置 {privacyInfo.without_conf_list?.length || 0} 个
                    </Badge>
                  </div>
                  {privacyInfo.without_auth_list?.length > 0 && (
                    <div className="font-mono text-[11px] text-red-600 break-all">无权限: {privacyInfo.without_auth_list.join(', ')}</div>
                  )}
                  {privacyInfo.without_conf_list?.length > 0 && (
                    <div className="font-mono text-[11px] text-amber-600 break-all">未配置: {privacyInfo.without_conf_list.join(', ')}</div>
                  )}
                </>
              ) : (
                <div className="text-muted-foreground">{ready ? '暂无检测结果 (尚未提交代码或检测中)' : '请先选择目标小程序'}</div>
              )}
              <label className="flex items-center gap-2 pt-1 cursor-pointer select-none">
                <input type="checkbox" checked={privacyNotUse} onChange={(e) => setPrivacyNotUse(e.target.checked)} className="h-3.5 w-3.5 rounded border-input" />
                <span>声明不使用「代码中检测出但未配置」的隐私接口 (privacy_api_not_use = true，可规避 61040)</span>
              </label>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="pb-3">
              <CardTitle className="text-base">按审核单号精确查询进度</CardTitle>
              <CardDescription className="text-xs">wxa/get_auditstatus，可查询历史任意审核单</CardDescription>
            </CardHeader>
            <CardContent className="text-xs space-y-2">
              <div className="flex gap-2">
                <Input value={queryAuditId} onChange={(e) => setQueryAuditId(e.target.value)} placeholder="输入提审返回的 auditid" className="h-8 text-[11px] font-mono" />
                <Button size="sm" variant="outline" className="h-8 shrink-0" disabled={!ready}
                  onClick={needApp(async () => {
                    if (!queryAuditId.trim()) { toast.error('请输入审核单号'); return }
                    const res: any = await api.get(withApp('/api/miniprogram/code/audit-status', { auditid: queryAuditId.trim() }))
                    setAuditDetail(res)
                  })}>
                  查询
                </Button>
              </div>
              {auditDetail && (
                <div className="p-3 rounded-lg border bg-muted/30 space-y-1">
                  <Badge variant="outline" className={AUDIT_STATUS[auditDetail.status]?.cls || ''}>
                    {AUDIT_STATUS[auditDetail.status]?.label || `状态码 ${auditDetail.status}`}
                  </Badge>
                  {!!auditDetail.reason && <div className="text-red-600">{auditDetail.reason}</div>}
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        {/* 发布与回退 */}
        <TabsContent value="release" className="mt-5 space-y-5">
          <Card>
            <CardHeader className="pb-3">
              <CardTitle className="text-base flex items-center gap-2"><Rocket className="h-4 w-4 text-rose-600" />发布上线与版本回退</CardTitle>
              <CardDescription className="text-xs">仅当最近一次审核状态为「审核成功，待发布」时才可全量发布</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4 text-xs">
              <div className="flex flex-wrap gap-2">
                <Button size="sm" className="gap-1.5 bg-rose-600 hover:bg-rose-700" disabled={!ready || releaseMutation.isPending}
                  onClick={needApp(() => {
                    if (!window.confirm('确认将该小程序已通过审核的版本全量发布上线？线上用户将立即生效')) return
                    releaseMutation.mutate()
                  })}>
                  <Rocket className="h-3.5 w-3.5" />全量发布 (release)
                </Button>
                <Button size="sm" variant="outline" className="gap-1.5" disabled={!ready || revertMutation.isPending}
                  onClick={needApp(() => {
                    if (!window.confirm('确认回退到上一个线上版本？该操作会立即改变用户访问内容')) return
                    revertMutation.mutate('')
                  })}>
                  <RotateCcw className="h-3.5 w-3.5" />回退上一版本
                </Button>
              </div>

              <div className="space-y-1.5">
                <div className="font-semibold">可回退的历史线上版本</div>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>app_version</TableHead>
                      <TableHead>版本号</TableHead>
                      <TableHead>描述</TableHead>
                      <TableHead>提交时间</TableHead>
                      <TableHead className="text-right">操作</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {(historyVersions?.versions || []).length > 0 ? (
                      historyVersions.versions.map((v: any) => (
                        <TableRow key={v.app_version} className="text-xs">
                          <TableCell className="font-mono">{v.app_version}</TableCell>
                          <TableCell className="font-semibold">{v.user_version}</TableCell>
                          <TableCell className="max-w-[240px] truncate" title={v.user_desc}>{v.user_desc}</TableCell>
                          <TableCell className="text-muted-foreground">{fmtTs(v.commit_time)}</TableCell>
                          <TableCell className="text-right">
                            <Button size="sm" variant="outline" className="h-7 text-[11px] gap-1" disabled={revertMutation.isPending}
                              onClick={needApp(() => {
                                if (!window.confirm(`确认回退到版本 ${v.user_version} (app_version=${v.app_version})？`)) return
                                revertMutation.mutate(String(v.app_version))
                              })}>
                              <RotateCcw className="h-3 w-3" />回退至此版本
                            </Button>
                          </TableCell>
                        </TableRow>
                      ))
                    ) : (
                      <TableRow>
                        <TableCell colSpan={5} className="h-24 text-center text-muted-foreground">
                          {ready ? '暂无可回退的历史版本 (需已有线上发布记录)' : '请先选择目标小程序'}
                        </TableCell>
                      </TableRow>
                    )}
                  </TableBody>
                </Table>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* 类目与体验者 */}
        <TabsContent value="category" className="mt-5 space-y-5">
          <div className="grid lg:grid-cols-2 gap-5">
            <Card>
              <CardHeader className="pb-3">
                <CardTitle className="text-base flex items-center gap-2"><ShieldCheck className="h-4 w-4 text-violet-600" />服务类目管理</CardTitle>
                <CardDescription className="text-xs">提审前必须至少有一个审核通过的类目；下方显示本更改周期配额</CardDescription>
              </CardHeader>
              <CardContent className="space-y-3 text-xs">
                <div className="flex flex-wrap gap-2 items-center">
                  <Badge variant="outline">周期可改 {categories?.limit ?? '-'}</Badge>
                  <Badge variant="outline">剩余 {categories?.quota ?? '-'}</Badge>
                  <Badge variant="outline">最多类目 {categories?.category_limit ?? '-'}</Badge>
                </div>

                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>一级 / 二级类目</TableHead>
                      <TableHead>审核状态</TableHead>
                      <TableHead className="text-right">操作</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {(categories?.categories || []).length > 0 ? (
                      categories.categories.map((cat: any, i: number) => (
                        <TableRow key={i} className="text-xs">
                          <TableCell>
                            <div className="font-semibold">{cat.first_name || cat.first} / {cat.second_name || cat.second}</div>
                            <div className="text-[10.5px] text-muted-foreground font-mono">ID {cat.first} / {cat.second}</div>
                          </TableCell>
                          <TableCell>
                            <Badge variant={cat.audit_status === 3 ? 'success' : cat.audit_status === 2 ? 'destructive' : 'outline'}>
                              {cat.audit_status === 1 ? '审核中' : cat.audit_status === 2 ? '审核不通过' : cat.audit_status === 3 ? '审核通过' : '未设置'}
                            </Badge>
                            {!!cat.audit_reason && <div className="text-[10.5px] text-red-600 mt-0.5">{cat.audit_reason}</div>}
                          </TableCell>
                          <TableCell className="text-right">
                            <Button size="sm" variant="outline" className="h-7 text-[11px] gap-1 text-red-600" disabled={delCatMutation.isPending}
                              onClick={needApp(() => {
                                if (!window.confirm(`确认删除类目 ${cat.first_name || cat.first} / ${cat.second_name || cat.second}？`)) return
                                delCatMutation.mutate({ first: cat.first, second: cat.second })
                              })}>
                              <Trash2 className="h-3 w-3" />删除
                            </Button>
                          </TableCell>
                        </TableRow>
                      ))
                    ) : (
                      <TableRow>
                        <TableCell colSpan={3} className="h-24 text-center text-muted-foreground">
                          {ready ? '尚未设置任何服务类目' : '请先选择目标小程序'}
                        </TableCell>
                      </TableRow>
                    )}
                  </TableBody>
                </Table>

                <div className="p-3 rounded-lg border bg-muted/20 space-y-2">
                  <div className="font-semibold">新增类目</div>
                  <div className="grid grid-cols-2 gap-2">
                    <Input value={catFirst} onChange={(e) => setCatFirst(e.target.value)} placeholder="一级类目 ID *" className="h-8 text-[11px] font-mono" />
                    <Input value={catSecond} onChange={(e) => setCatSecond(e.target.value)} placeholder="二级类目 ID *" className="h-8 text-[11px] font-mono" />
                    <Input value={catCertKey} onChange={(e) => setCatCertKey(e.target.value)} placeholder="资质名称 (需资质时填)" className="h-8 text-[11px]" />
                    <Input value={catCertValue} onChange={(e) => setCatCertValue(e.target.value)} placeholder="资质图片 media_id" className="h-8 text-[11px] font-mono" />
                  </div>
                  <div className="flex justify-end">
                    <Button size="sm" className="gap-1.5" disabled={!ready || addCatMutation.isPending}
                      onClick={needApp(() => {
                        if (!catFirst || !catSecond) { toast.error('一级与二级类目 ID 不能为空'); return }
                        const certs = catCertKey && catCertValue ? [{ key: catCertKey, value: catCertValue }] : undefined
                        addCatMutation.mutate({ first: Number(catFirst), second: Number(catSecond), certicates: certs })
                      })}>
                      <Plus className="h-3.5 w-3.5" />提交新增类目
                    </Button>
                  </div>
                </div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="pb-3">
                <CardTitle className="text-base flex items-center gap-2"><Users className="h-4 w-4 text-sky-600" />体验者成员管理</CardTitle>
                <CardDescription className="text-xs">只有已绑定的体验者才能扫描体验版二维码访问体验版</CardDescription>
              </CardHeader>
              <CardContent className="space-y-3 text-xs">
                <div className="flex gap-2">
                  <Input value={testerWechat} onChange={(e) => setTesterWechat(e.target.value)} placeholder="体验者微信号 (非昵称)" className="h-9 text-xs font-mono" />
                  <Button size="sm" className="h-9 shrink-0 gap-1.5" disabled={!ready || bindMutation.isPending}
                    onClick={needApp(() => {
                      if (!testerWechat.trim()) { toast.error('请输入体验者微信号'); return }
                      bindMutation.mutate(testerWechat.trim())
                    })}>
                    <Plus className="h-3.5 w-3.5" />绑定
                  </Button>
                </div>

                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>微信号</TableHead>
                      <TableHead>唯一标识 userstr</TableHead>
                      <TableHead className="text-right">操作</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {(testers?.members || []).length > 0 ? (
                      testers.members.map((m: any, i: number) => (
                        <TableRow key={i} className="text-xs">
                          <TableCell className="font-mono">{m.wechatid || '-'}</TableCell>
                          <TableCell className="font-mono text-muted-foreground break-all text-[10.5px]">{m.userstr || '-'}</TableCell>
                          <TableCell className="text-right">
                            <Button size="sm" variant="outline" className="h-7 text-[11px] gap-1 text-red-600" disabled={unbindMutation.isPending}
                              onClick={needApp(() => unbindMutation.mutate({ wechatid: m.wechatid || '', userstr: m.userstr || '' }))}>
                              <Trash2 className="h-3 w-3" />解绑
                            </Button>
                          </TableCell>
                        </TableRow>
                      ))
                    ) : (
                      <TableRow>
                        <TableCell colSpan={3} className="h-24 text-center text-muted-foreground">
                          {ready ? '尚未绑定任何体验者' : '请先选择目标小程序'}
                        </TableCell>
                      </TableRow>
                    )}
                  </TableBody>
                </Table>

                <div className="p-3 rounded-lg bg-sky-500/10 border border-sky-500/20 text-sky-900 dark:text-sky-200 leading-relaxed">
                  <Save className="h-3.5 w-3.5 inline mr-1" />
                  完整发布流程：代码模板库取 template_id → 提交代码 (commit) → 绑定体验者扫码体验 → 配置服务类目 → 提交审核 (submit_audit) → 审核通过后全量发布 (release)。
                </div>
              </CardContent>
            </Card>
          </div>
        </TabsContent>
      </Tabs>
    </div>
  )
}
