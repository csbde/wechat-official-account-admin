import React, { useMemo, useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'
import { useAppStore } from '@/stores/app'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import {
  Table, TableHeader, TableBody, TableHead, TableRow, TableCell,
} from '@/components/ui/table'
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter,
} from '@/components/ui/dialog'
import {
  MessageSquare, Send, Trash2, Plus, Library, RefreshCw, ShieldAlert, Bell,
  FileText, ChevronLeft, ChevronRight, Search,
} from 'lucide-react'
import { toast } from 'sonner'

/** 解析模板 content 中的 {{key.DATA}} 占位，生成可填写的表单字段 */
function parseTemplate(content?: string, example?: string) {
  const lines = (content || '').split('\n').map((l) => l.trim()).filter(Boolean)
  const exLines = (example || '').split('\n').map((l) => l.trim()).filter(Boolean)
  const out: { key: string; label: string; ex: string }[] = []
  lines.forEach((line, i) => {
    const m = line.match(/\{\s*\{\s*([A-Za-z0-9_]+)\s*(?:\.[A-Za-z_]+)?\s*\}\s*\}/)
    if (!m) return
    const key = m[1]
    const label = line.slice(0, m.index).replace(/[:：]/g, '').trim() || key
    const exLine = exLines[i] || ''
    let ex = exLine
    const sep = exLine.includes(':') ? exLine.indexOf(':') : exLine.indexOf('：')
    if (sep > -1) ex = exLine.slice(sep + 1).trim()
    out.push({ key, label, ex })
  })
  return out
}

export const NotifyPage: React.FC = () => {
  const queryClient = useQueryClient()
  const { currentApp } = useAppStore()
  const [tab, setTab] = useState<'oa' | 'sub'>('oa')

  /* ============================ 公众号模板消息 ============================ */
  const { data: industry } = useQuery<any>({
    queryKey: ['notify', currentApp?.appid, 'industry'],
    queryFn: () => api.get('/api/notify/oa/industry'),
    enabled: !!currentApp && tab === 'oa',
  })
  const { data: oaTemplates, isFetching: fetchingOa } = useQuery<any[]>({
    queryKey: ['notify', currentApp?.appid, 'oa-templates'],
    queryFn: () => api.get('/api/notify/oa/templates'),
    enabled: !!currentApp && tab === 'oa',
  })

  const [ind1, setInd1] = useState('')
  const [ind2, setInd2] = useState('')
  const [addShort, setAddShort] = useState('')
  const [addKeywords, setAddKeywords] = useState('')

  const [sendOpen, setSendOpen] = useState(false)
  const [sendTmpl, setSendTmpl] = useState<any>(null)
  const [sendValues, setSendValues] = useState<Record<string, string>>({})
  const [sendToUser, setSendToUser] = useState('')
  const [sendUrl, setSendUrl] = useState('')
  const [sendMiniAppid, setSendMiniAppid] = useState('')
  const [sendPagepath, setSendPagepath] = useState('')
  const [sendColor, setSendColor] = useState('')
  const [sendClientMsg, setSendClientMsg] = useState('')

  const [blockId, setBlockId] = useState('')
  const [blockResult, setBlockResult] = useState<any>(null)

  const setIndustryMutation = useMutation({
    mutationFn: (v: any) => api.post('/api/notify/oa/industry', v),
    onSuccess: (d: any) => {
      toast.success(d?.message || '行业设置成功')
      queryClient.invalidateQueries({ queryKey: ['notify', currentApp?.appid] })
    },
  })
  const addTemplateMutation = useMutation({
    mutationFn: (v: any) => api.post('/api/notify/oa/templates', v),
    onSuccess: (d: any) => {
      toast.success(`选用成功，模板 ID: ${d?.template_id}`)
      queryClient.invalidateQueries({ queryKey: ['notify', currentApp?.appid, 'oa-templates'] })
      setAddShort('')
      setAddKeywords('')
    },
  })
  const delTemplateMutation = useMutation({
    mutationFn: (template_id: string) => api.post('/api/notify/oa/templates/delete', { template_id }),
    onSuccess: () => {
      toast.success('模板已删除')
      queryClient.invalidateQueries({ queryKey: ['notify', currentApp?.appid, 'oa-templates'] })
    },
  })
  const sendMutation = useMutation({
    mutationFn: (v: any) => api.post('/api/notify/oa/send', v),
    onSuccess: (d: any) => {
      toast.success(`发送成功，微信消息号 msgid=${d?.msgid ?? '-'}`)
      setSendOpen(false)
    },
  })
  const blockMutation = useMutation({
    mutationFn: (v: any) => api.post('/api/notify/oa/blocked', v),
    onSuccess: (d: any) => setBlockResult(d),
  })

  const openSend = (tmpl: any) => {
    setSendTmpl(tmpl)
    setSendValues({})
    setSendOpen(true)
  }

  /* ============================ 订阅消息 ============================ */
  const [subCatIds, setSubCatIds] = useState<string[]>([])
  const [subPage, setSubPage] = useState(0)
  const subLimit = 30

  const { data: subCats } = useQuery<any[]>({
    queryKey: ['notify', currentApp?.appid, 'sub-categories'],
    queryFn: () => api.get('/api/notify/sub/categories'),
    enabled: !!currentApp && tab === 'sub',
  })
  const { data: pubTitles, isFetching: fetchingPub } = useQuery<any>({
    queryKey: ['notify', currentApp?.appid, 'sub-pub', subCatIds.join(','), subPage],
    queryFn: () => api.get(`/api/notify/sub/pub-titles?ids=${subCatIds.join(',')}&start=${subPage * subLimit}&limit=${subLimit}`),
    enabled: !!currentApp && tab === 'sub' && subCatIds.length > 0,
  })
  const { data: subMine, isFetching: fetchingMine } = useQuery<any[]>({
    queryKey: ['notify', currentApp?.appid, 'sub-mine'],
    queryFn: () => api.get('/api/notify/sub/templates'),
    enabled: !!currentApp && tab === 'sub',
  })

  const [addSubOpen, setAddSubOpen] = useState(false)
  const [addSubTmpl, setAddSubTmpl] = useState<any>(null)
  const { data: subKeywords } = useQuery<any[]>({
    queryKey: ['notify', currentApp?.appid, 'sub-kids', addSubTmpl?.tid],
    queryFn: () => api.get(`/api/notify/sub/pub-keywords?tid=${addSubTmpl.tid}`),
    enabled: !!addSubTmpl,
  })
  const [pickedKids, setPickedKids] = useState<number[]>([])
  const [sceneDesc, setSceneDesc] = useState('')

  const [subSendOpen, setSubSendOpen] = useState(false)
  const [subSendTmpl, setSubSendTmpl] = useState<any>(null)
  const [subSendValues, setSubSendValues] = useState<Record<string, string>>({})
  const [subToUser, setSubToUser] = useState('')
  const [subPage2, setSubPage2] = useState('')
  const [subState, setSubState] = useState('formal')
  const [subLang, setSubLang] = useState('zh_CN')
  const [subMode, setSubMode] = useState<'send' | 'bizsend'>('send')
  const [subUrl, setSubUrl] = useState('')

  const addSubMutation = useMutation({
    mutationFn: (v: any) => api.post('/api/notify/sub/templates', v),
    onSuccess: (d: any) => {
      toast.success(`已加入我的模板库，priTmplId: ${d?.priTmplId}`)
      queryClient.invalidateQueries({ queryKey: ['notify', currentApp?.appid, 'sub-mine'] })
      setAddSubOpen(false)
      setPickedKids([])
      setSceneDesc('')
    },
  })
  const delSubMutation = useMutation({
    mutationFn: (priTmplId: string) => api.post('/api/notify/sub/templates/delete', { priTmplId }),
    onSuccess: () => {
      toast.success('订阅模板已删除')
      queryClient.invalidateQueries({ queryKey: ['notify', currentApp?.appid, 'sub-mine'] })
    },
  })
  const subSendMutation = useMutation({
    mutationFn: (v: any) => api.post(subMode === 'send' ? '/api/notify/sub/send' : '/api/notify/sub/bizsend', v),
    onSuccess: () => {
      toast.success('订阅消息已成功下发至微信')
      setSubSendOpen(false)
    },
  })

  const toggleCat = (id: string) =>
    setSubCatIds((arr) => (arr.includes(id) ? arr.filter((x) => x !== id) : [...arr, id].slice(0, 5)))

  const subTotalPages = useMemo(
    () => Math.max(1, Math.ceil((pubTitles?.total || 0) / subLimit)),
    [pubTitles],
  )

  const sendFields = sendTmpl ? parseTemplate(sendTmpl.content, sendTmpl.example) : []
  const subSendFields = subSendTmpl ? parseTemplate(subSendTmpl.content, subSendTmpl.example) : []

  if (!currentApp) {
    return (
      <Card>
        <CardContent className="py-16 text-center text-sm text-muted-foreground">
          请先在右上角选择一个公众号或小程序，再进行模板消息 / 订阅消息管理
        </CardContent>
      </Card>
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-bold tracking-tight flex items-center gap-2">
            <Bell className="h-6 w-6 text-primary" />
            <span>模板消息与订阅消息</span>
          </h2>
          <p className="text-sm text-muted-foreground">
            当前账号：
            <span className="font-semibold text-foreground">{currentApp.name}</span>
            <span className="font-mono text-xs ml-1.5 text-muted-foreground">{currentApp.appid}</span>
            <Badge variant="outline" className="ml-2 text-[10px]">
              {currentApp.account_type === 'miniprogram' ? '小程序 (订阅消息)' : '公众号 (模板消息/订阅消息)'}
            </Badge>
          </p>
        </div>
        <Button variant="outline" className="gap-2"
          onClick={() => queryClient.invalidateQueries({ queryKey: ['notify', currentApp.appid] })}>
          <RefreshCw className="h-4 w-4" />刷新全部
        </Button>
      </div>

      <Tabs value={tab} onValueChange={(v: any) => setTab(v)}>
        <TabsList className="grid w-full max-w-md grid-cols-2">
          <TabsTrigger value="oa" className="text-xs gap-1.5">
            <MessageSquare className="h-3.5 w-3.5 text-emerald-500" />公众号模板消息
          </TabsTrigger>
          <TabsTrigger value="sub" className="text-xs gap-1.5">
            <Bell className="h-3.5 w-3.5 text-violet-500" />订阅消息
          </TabsTrigger>
        </TabsList>

        {/* ===================== 公众号模板消息 ===================== */}
        <TabsContent value="oa" className="mt-6 space-y-5">
          <div className="p-4 rounded-xl bg-emerald-500/10 border border-emerald-500/20 text-xs text-emerald-900 dark:text-emerald-200 flex items-start gap-3">
            <ShieldAlert className="h-5 w-5 text-emerald-600 shrink-0 mt-0.5" />
            <div className="space-y-1 leading-relaxed">
              <div className="font-semibold text-sm">模板消息使用要点 (仅认证服务号可用)</div>
              <div>
                1. 必须先<strong>设置所属行业</strong>，行业代码见微信官方《模板消息行业代码表》；<strong>更改行业会删除原行业下已选用的模板</strong>。<br />
                2. 每个账号最多同时使用 25 个模板，日调用上限 10 万次；模板内容关键词以 <code>{'{{xxx.DATA}}'}</code> 占位。<br />
                3. 微信下发结果会通过服务器事件 <code>TEMPLATESENDJOBFINISH</code> 回调，可在「请求与追踪日志」中查看。
              </div>
            </div>
          </div>

          <div className="grid lg:grid-cols-2 gap-5">
            {/* 行业设置 */}
            <Card>
              <CardHeader className="pb-3">
                <CardTitle className="text-base">所属行业</CardTitle>
                <CardDescription className="text-xs">
                  当前: {industry?.primary_industry?.first_class || '未设置'} / {industry?.primary_industry?.second_class || '-'}
                  {' · '}副行业: {industry?.secondary_industry?.first_class || '-'} / {industry?.secondary_industry?.second_class || '-'}
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-3 text-xs">
                <div className="grid grid-cols-2 gap-2">
                  <Input value={ind1} onChange={(e) => setInd1(e.target.value)} placeholder="一级行业代码 industry_id1" className="h-9 text-xs font-mono" />
                  <Input value={ind2} onChange={(e) => setInd2(e.target.value)} placeholder="二级行业代码 industry_id2" className="h-9 text-xs font-mono" />
                </div>
                <div className="text-muted-foreground">
                  常见示例：<code>TRANSPORTATION</code>(交通)、<code>WEATHER</code>(天气生活)、<code>FINANCE</code>(金融)、
                  <code>IT</code>(IT科技)、<code>EDUCATION</code>(教育)、<code>MEDICAL</code>(医疗)
                </div>
                <div className="flex justify-end">
                  <Button size="sm" className="gap-1.5" disabled={setIndustryMutation.isPending}
                    onClick={() => {
                      if (!ind1.trim() || !ind2.trim()) { toast.error('请填写一级与二级行业代码'); return }
                      if (!window.confirm('更改行业后，微信会删除原行业下已选用的全部模板，确认继续？')) return
                      setIndustryMutation.mutate({ industry_id1: ind1.trim(), industry_id2: ind2.trim() })
                    }}>
                    <Send className="h-3.5 w-3.5" />设置行业
                  </Button>
                </div>
              </CardContent>
            </Card>

            {/* 选用新模板 */}
            <Card>
              <CardHeader className="pb-3">
                <CardTitle className="text-base flex items-center gap-2"><Library className="h-4 w-4 text-sky-600" />从公共模板库选用</CardTitle>
                <CardDescription className="text-xs">
                  template_id_short 可在公众平台「模板消息 → 类目模板库 / 行业模板库」中查看复制
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-3 text-xs">
                <div className="space-y-1">
                  <label className="font-semibold">公共模板编号 template_id_short *</label>
                  <Input value={addShort} onChange={(e) => setAddShort(e.target.value)} placeholder="如 47123" className="h-9 text-xs font-mono" />
                </div>
                <div className="space-y-1">
                  <label className="font-semibold">指定关键词 (可选，逗号分隔)</label>
                  <Input value={addKeywords} onChange={(e) => setAddKeywords(e.target.value)} placeholder="如 时间,地点,金额" className="h-9 text-xs" />
                </div>
                <div className="flex justify-end">
                  <Button size="sm" variant="outline" className="gap-1.5" disabled={addTemplateMutation.isPending}
                    onClick={() => {
                      if (!addShort.trim()) { toast.error('请填写公共模板编号'); return }
                      const list = addKeywords.split(/[,，]/).map((x) => x.trim()).filter(Boolean)
                      addTemplateMutation.mutate({
                        template_id_short: addShort.trim(),
                        keyword_name_list: list.length ? list : undefined,
                      })
                    }}>
                    <Plus className="h-3.5 w-3.5" />选用模板
                  </Button>
                </div>
              </CardContent>
            </Card>
          </div>

          {/* 我的模板库 */}
          <Card>
            <CardHeader className="pb-3">
              <CardTitle className="text-base flex items-center gap-2">
                <FileText className="h-4 w-4 text-emerald-600" />我的模板库
                {oaTemplates?.length ? <Badge variant="outline" className="text-[10px]">{oaTemplates.length} / 25</Badge> : null}
              </CardTitle>
              <CardDescription className="text-xs">点击「发送」可按模板关键词自动填充表单</CardDescription>
            </CardHeader>
            <CardContent className="p-0">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>模板标题</TableHead>
                    <TableHead>模板 ID</TableHead>
                    <TableHead>所属行业</TableHead>
                    <TableHead>关键词</TableHead>
                    <TableHead className="text-right">操作</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {fetchingOa && !oaTemplates ? (
                    <TableRow><TableCell colSpan={5} className="h-28 text-center text-muted-foreground">加载中...</TableCell></TableRow>
                  ) : oaTemplates && oaTemplates.length > 0 ? (
                    oaTemplates.map((t: any) => (
                      <TableRow key={t.template_id} className="text-xs">
                        <TableCell className="font-semibold max-w-[180px] truncate" title={t.title}>{t.title || '-'}</TableCell>
                        <TableCell className="font-mono text-[10.5px] break-all max-w-[200px]">{t.template_id}</TableCell>
                        <TableCell className="text-muted-foreground">{[t.primary_industry, t.deputy_industry].filter(Boolean).join(' | ') || '-'}</TableCell>
                        <TableCell>
                          <div className="flex flex-wrap gap-1">
                            {(t.keywords || []).map((k: string) => (
                              <Badge key={k} variant="secondary" className="text-[10px] py-0 font-mono">{k}</Badge>
                            ))}
                          </div>
                        </TableCell>
                        <TableCell className="text-right space-x-1 whitespace-nowrap">
                          <Button size="sm" className="h-7 text-[11px] gap-1 bg-emerald-600 hover:bg-emerald-700" onClick={() => openSend(t)}>
                            <Send className="h-3 w-3" />发送
                          </Button>
                          <Button size="sm" variant="outline" className="h-7 text-[11px] gap-1 text-red-600"
                            disabled={delTemplateMutation.isPending}
                            onClick={() => {
                              if (!window.confirm(`确认删除模板「${t.title}」？`)) return
                              delTemplateMutation.mutate(t.template_id)
                            }}>
                            <Trash2 className="h-3 w-3" />删除
                          </Button>
                        </TableCell>
                      </TableRow>
                    ))
                  ) : (
                    <TableRow><TableCell colSpan={5} className="h-28 text-center text-muted-foreground">模板库为空，请先设置行业并选用模板</TableCell></TableRow>
                  )}
                </TableBody>
              </Table>
            </CardContent>
          </Card>

          {/* 拦截查询 */}
          <Card>
            <CardHeader className="pb-3">
              <CardTitle className="text-base">查询被拦截的模板消息</CardTitle>
              <CardDescription className="text-xs">wxa/sec/queryblocktmplmsg，用于排查模板消息被判为骚扰/营销而拦截的原因</CardDescription>
            </CardHeader>
            <CardContent className="text-xs space-y-2">
              <div className="flex flex-wrap gap-2">
                <Input value={blockId} onChange={(e) => setBlockId(e.target.value)} placeholder="模板消息 msgid (可留空查最新)" className="h-9 text-xs font-mono max-w-[280px]" />
                <Button size="sm" variant="outline" className="h-9 gap-1.5" disabled={blockMutation.isPending}
                  onClick={() => blockMutation.mutate({ tmpl_msg_id: blockId.trim(), largest_id: 0, limit: 20 })}>
                  <Search className="h-3.5 w-3.5" />查询
                </Button>
              </div>
              {blockResult && (
                <pre className="p-3 rounded-lg bg-muted/60 border font-mono text-[10.5px] overflow-auto max-h-52">
{JSON.stringify(blockResult, null, 2)}
                </pre>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        {/* ===================== 订阅消息 ===================== */}
        <TabsContent value="sub" className="mt-6 space-y-5">
          <div className="p-4 rounded-xl bg-violet-500/10 border border-violet-500/20 text-xs text-violet-900 dark:text-violet-200 flex items-start gap-3">
            <ShieldAlert className="h-5 w-5 text-violet-600 shrink-0 mt-0.5" />
            <div className="space-y-1 leading-relaxed">
              <div className="font-semibold text-sm">订阅消息使用要点 (小程序 / 服务号)</div>
              <div>
                1. 流程：<strong>选类目 → 拉公共模板 → 勾选 2~5 个关键词 → 加入我的模板库</strong>，得到 <code>priTmplId</code> 后方可发送。<br />
                2. 一次性订阅需用户在小程序内 <code>wx.requestSubscribeMessage</code> 授权，每次授权仅可下发一条；长期订阅需特殊资质。<br />
                3. 发送时的 <code>data</code> 键名必须与模板关键词一致 (如 thing1、character_string2、amount3)。
              </div>
            </div>
          </div>

          {/* 类目 + 公共模板库 */}
          <Card>
            <CardHeader className="pb-3">
              <CardTitle className="text-base">1) 选择类目并拉取公共模板库</CardTitle>
              <CardDescription className="text-xs">已选类目：{subCatIds.length ? subCats?.filter((x) => subCatIds.includes(x.id)).map((x) => x.name).join('、') : '未选择 (可多选，至多 5 个)'}</CardDescription>
            </CardHeader>
            <CardContent className="space-y-3 text-xs">
              <div className="flex flex-wrap gap-1.5 max-h-32 overflow-y-auto p-2 rounded-lg border bg-muted/30">
                {(subCats || []).map((cat: any) => (
                  <button key={cat.id} type="button"
                    onClick={() => { toggleCat(cat.id); setSubPage(0) }}
                    className={`px-2 py-0.5 rounded-md border text-[11px] transition ${
                      subCatIds.includes(cat.id)
                        ? 'bg-violet-600 text-white border-violet-600'
                        : 'bg-background hover:border-primary'
                    }`}>
                    {cat.name} <span className="font-mono opacity-70">#{cat.id}</span>
                  </button>
                ))}
                {!subCats?.length && <span className="text-muted-foreground p-2">正在拉取类目列表...</span>}
              </div>

              {subCatIds.length === 0 ? (
                <div className="py-8 text-center text-muted-foreground">请先点击上方类目以拉取该类目下的公共模板</div>
              ) : (
                <>
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>tid</TableHead>
                        <TableHead>模板标题</TableHead>
                        <TableHead>订阅类型</TableHead>
                        <TableHead>类目</TableHead>
                        <TableHead className="text-right">操作</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {fetchingPub ? (
                        <TableRow><TableCell colSpan={5} className="h-24 text-center text-muted-foreground">拉取中...</TableCell></TableRow>
                      ) : pubTitles?.list?.length ? (
                        pubTitles.list.map((t: any) => (
                          <TableRow key={t.tid} className="text-xs">
                            <TableCell className="font-mono">{t.tid}</TableCell>
                            <TableCell className="font-semibold">{t.title}</TableCell>
                            <TableCell>
                              <Badge variant={t.type === 3 ? 'success' : 'outline'}>{t.type === 3 ? '长期订阅' : '一次性订阅'}</Badge>
                            </TableCell>
                            <TableCell className="font-mono text-muted-foreground">{t.categoryId}</TableCell>
                            <TableCell className="text-right">
                              <Button size="sm" variant="outline" className="h-7 text-[11px] gap-1"
                                onClick={() => { setAddSubTmpl(t); setPickedKids([]); setAddSubOpen(true) }}>
                                <Plus className="h-3 w-3" />选关键词并添加
                              </Button>
                            </TableCell>
                          </TableRow>
                        ))
                      ) : (
                        <TableRow><TableCell colSpan={5} className="h-24 text-center text-muted-foreground">该类目下暂无公共模板</TableCell></TableRow>
                      )}
                    </TableBody>
                  </Table>

                  <div className="flex items-center justify-end gap-2">
                    <Button size="sm" variant="outline" className="h-7 text-[11px] gap-1" disabled={subPage <= 0} onClick={() => setSubPage((p) => Math.max(0, p - 1))}>
                      <ChevronLeft className="h-3 w-3" />上一页
                    </Button>
                    <span className="text-[11px] text-muted-foreground">
                      第 {subPage + 1} / {subTotalPages} 页 · 共 {pubTitles?.total || 0} 条
                    </span>
                    <Button size="sm" variant="outline" className="h-7 text-[11px] gap-1" disabled={subPage + 1 >= subTotalPages} onClick={() => setSubPage((p) => p + 1)}>
                      下一页<ChevronRight className="h-3 w-3" />
                    </Button>
                  </div>
                </>
              )}
            </CardContent>
          </Card>

          {/* 我的订阅模板库 */}
          <Card>
            <CardHeader className="pb-3">
              <CardTitle className="text-base flex items-center gap-2">
                2) 我的订阅模板库
                {subMine?.length ? <Badge variant="outline" className="text-[10px]">{subMine.length} 个</Badge> : null}
              </CardTitle>
              <CardDescription className="text-xs">发送订阅消息需使用这里的 priTmplId</CardDescription>
            </CardHeader>
            <CardContent className="p-0">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>模板标题</TableHead>
                    <TableHead>priTmplId</TableHead>
                    <TableHead>类型</TableHead>
                    <TableHead>关键词</TableHead>
                    <TableHead className="text-right">操作</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {fetchingMine && !subMine ? (
                    <TableRow><TableCell colSpan={5} className="h-24 text-center text-muted-foreground">加载中...</TableCell></TableRow>
                  ) : subMine && subMine.length > 0 ? (
                    subMine.map((t: any) => (
                      <TableRow key={t.priTmplId} className="text-xs">
                        <TableCell className="font-semibold max-w-[180px] truncate" title={t.title}>{t.title}</TableCell>
                        <TableCell className="font-mono text-[10.5px] break-all max-w-[200px]">{t.priTmplId}</TableCell>
                        <TableCell><Badge variant={t.type === 3 ? 'success' : 'outline'}>{t.type === 3 ? '长期' : '一次性'}</Badge></TableCell>
                        <TableCell>
                          <div className="flex flex-wrap gap-1">
                            {(t.keywords || []).map((k: string) => (
                              <Badge key={k} variant="secondary" className="text-[10px] py-0 font-mono">{k}</Badge>
                            ))}
                          </div>
                        </TableCell>
                        <TableCell className="text-right space-x-1 whitespace-nowrap">
                          <Button size="sm" className="h-7 text-[11px] gap-1 bg-violet-600 hover:bg-violet-700"
                            onClick={() => { setSubSendTmpl(t); setSubSendValues({}); setSubSendOpen(true) }}>
                            <Send className="h-3 w-3" />发送
                          </Button>
                          <Button size="sm" variant="outline" className="h-7 text-[11px] gap-1 text-red-600"
                            disabled={delSubMutation.isPending}
                            onClick={() => {
                              if (!window.confirm(`确认删除订阅模板「${t.title}」？`)) return
                              delSubMutation.mutate(t.priTmplId)
                            }}>
                            <Trash2 className="h-3 w-3" />删除
                          </Button>
                        </TableCell>
                      </TableRow>
                    ))
                  ) : (
                    <TableRow><TableCell colSpan={5} className="h-24 text-center text-muted-foreground">尚未添加任何订阅模板</TableCell></TableRow>
                  )}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      {/* Dialog: 发送模板消息 */}
      <Dialog open={sendOpen} onOpenChange={setSendOpen}>
        <DialogContent className="sm:max-w-lg max-h-[88vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>发送模板消息</DialogTitle>
          </DialogHeader>
          {sendTmpl && (
            <div className="space-y-3 text-xs">
              <div className="p-2.5 rounded-lg bg-muted/60 border space-y-1">
                <div className="font-semibold">{sendTmpl.title}</div>
                <div className="font-mono text-[10.5px] text-muted-foreground break-all">{sendTmpl.template_id}</div>
                <pre className="whitespace-pre-wrap text-[10.5px] max-h-24 overflow-auto">{sendTmpl.content}</pre>
              </div>
              <div className="grid grid-cols-2 gap-2">
                <Input value={sendToUser} onChange={(e) => setSendToUser(e.target.value)} placeholder="接收者 openid *" className="h-8 text-[11px] font-mono" />
                <Input value={sendClientMsg} onChange={(e) => setSendClientMsg(e.target.value)} placeholder="client_msg_id (防重入，可选)" className="h-8 text-[11px] font-mono" />
                <Input value={sendUrl} onChange={(e) => setSendUrl(e.target.value)} placeholder="点击跳转 url (可选)" className="h-8 text-[11px]" />
                <Input value={sendColor} onChange={(e) => setSendColor(e.target.value)} placeholder="字体颜色 如 #173177 (可选)" className="h-8 text-[11px] font-mono" />
                <Input value={sendMiniAppid} onChange={(e) => setSendMiniAppid(e.target.value)} placeholder="跳转小程序 appid (可选)" className="h-8 text-[11px] font-mono" />
                <Input value={sendPagepath} onChange={(e) => setSendPagepath(e.target.value)} placeholder="小程序页面路径 index?foo=bar" className="h-8 text-[11px] font-mono" />
              </div>

              <div className="space-y-1.5">
                <div className="font-semibold">模板内容填充 (共 {sendFields.length} 个关键词)</div>
                {sendFields.map((f) => (
                  <div key={f.key} className="flex items-center gap-2">
                    <span className="w-28 shrink-0 text-muted-foreground truncate" title={f.label}>{f.label}</span>
                    <span className="font-mono text-[10.5px] w-28 shrink-0">{f.key}</span>
                    <Input value={sendValues[f.key] || ''} onChange={(e) => setSendValues((v) => ({ ...v, [f.key]: e.target.value }))}
                      placeholder={f.ex || '请输入内容'} className="h-8 text-[11px]" />
                  </div>
                ))}
              </div>

              <DialogFooter className="pt-2">
                <Button variant="outline" onClick={() => setSendOpen(false)}>取消</Button>
                <Button className="gap-1.5 bg-emerald-600 hover:bg-emerald-700" disabled={sendMutation.isPending}
                  onClick={() => {
                    if (!sendToUser.trim()) { toast.error('请填写接收者 openid'); return }
                    const empty = sendFields.filter((f) => !(sendValues[f.key] || '').trim())
                    if (empty.length) { toast.error(`请填写：${empty.map((e) => e.label).join('、')}`); return }
                    sendMutation.mutate({
                      touser: sendToUser.trim(),
                      template_id: sendTmpl.template_id,
                      url: sendUrl.trim() || undefined,
                      mini_appid: sendMiniAppid.trim() || undefined,
                      pagepath: sendPagepath.trim() || undefined,
                      client_msg_id: sendClientMsg.trim() || undefined,
                      color: sendColor.trim() || undefined,
                      values: sendValues,
                    })
                  }}>
                  <Send className="h-3.5 w-3.5" />{sendMutation.isPending ? '发送中...' : '立即发送'}
                </Button>
              </DialogFooter>
            </div>
          )}
        </DialogContent>
      </Dialog>

      {/* Dialog: 添加订阅模板 */}
      <Dialog open={addSubOpen} onOpenChange={setAddSubOpen}>
        <DialogContent className="sm:max-w-lg max-h-[88vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>选择关键词并加入我的模板库</DialogTitle>
          </DialogHeader>
          {addSubTmpl && (
            <div className="space-y-3 text-xs">
              <div className="p-2.5 rounded-lg bg-muted/60 border">
                <div className="font-semibold">{addSubTmpl.title}</div>
                <div className="text-muted-foreground">tid = {addSubTmpl.tid} · {addSubTmpl.type === 3 ? '长期订阅' : '一次性订阅'}</div>
              </div>
              <div className="space-y-1.5">
                <div className="font-semibold">勾选关键词 (至少 2 个，至多 5 个)</div>
                {!subKeywords?.length && <div className="text-muted-foreground py-3 text-center">正在拉取关键词...</div>}
                {(subKeywords || []).map((k: any) => (
                  <label key={k.kid} className="flex items-center gap-2 p-2 rounded-lg border hover:bg-muted/40 cursor-pointer">
                    <input type="checkbox" className="h-3.5 w-3.5" checked={pickedKids.includes(k.kid)}
                      onChange={(e) => setPickedKids((arr) => e.target.checked ? [...arr, k.kid] : arr.filter((x) => x !== k.kid))} />
                    <span className="font-mono text-[10.5px] w-16">{k.rule}{k.kid}</span>
                    <span className="font-semibold">{k.name}</span>
                    <span className="text-muted-foreground ml-auto truncate max-w-[140px]" title={k.example}>示例: {k.example}</span>
                  </label>
                ))}
              </div>
              <div className="space-y-1">
                <label className="font-semibold">服务场景描述 sceneDesc (15 字内)</label>
                <Input value={sceneDesc} onChange={(e) => setSceneDesc(e.target.value)} placeholder="如：订单物流状态提醒" className="h-8 text-[11px]" />
              </div>
              <DialogFooter className="pt-2">
                <Button variant="outline" onClick={() => setAddSubOpen(false)}>取消</Button>
                <Button className="gap-1.5" disabled={addSubMutation.isPending}
                  onClick={() => {
                    if (pickedKids.length < 2 || pickedKids.length > 5) { toast.error('请勾选 2~5 个关键词'); return }
                    if (!sceneDesc.trim()) { toast.error('请填写服务场景描述'); return }
                    addSubMutation.mutate({ tid: addSubTmpl.tid, kid_list: pickedKids, scene_desc: sceneDesc.trim() })
                  }}>
                  <Plus className="h-3.5 w-3.5" />加入我的模板库
                </Button>
              </DialogFooter>
            </div>
          )}
        </DialogContent>
      </Dialog>

      {/* Dialog: 发送订阅消息 */}
      <Dialog open={subSendOpen} onOpenChange={setSubSendOpen}>
        <DialogContent className="sm:max-w-lg max-h-[88vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>发送订阅消息</DialogTitle>
          </DialogHeader>
          {subSendTmpl && (
            <div className="space-y-3 text-xs">
              <div className="p-2.5 rounded-lg bg-muted/60 border space-y-1">
                <div className="font-semibold">{subSendTmpl.title}</div>
                <div className="font-mono text-[10.5px] text-muted-foreground break-all">{subSendTmpl.priTmplId}</div>
                <pre className="whitespace-pre-wrap text-[10.5px] max-h-24 overflow-auto">{subSendTmpl.content}</pre>
              </div>

              <div className="flex gap-2">
                {(['send', 'bizsend'] as const).map((m) => (
                  <button key={m} type="button" onClick={() => setSubMode(m)}
                    className={`flex-1 h-8 rounded-lg border text-[11px] ${subMode === m ? 'bg-violet-600 text-white border-violet-600' : 'bg-background hover:border-primary'}`}>
                    {m === 'send' ? '标准订阅发送 (小程序)' : '一次性订阅下发 (服务号)'}
                  </button>
                ))}
              </div>

              <div className="grid grid-cols-2 gap-2">
                <Input value={subToUser} onChange={(e) => setSubToUser(e.target.value)} placeholder="接收者 openid *" className="h-8 text-[11px] font-mono" />
                <select value={subState} onChange={(e) => setSubState(e.target.value)} className="h-8 rounded-lg border border-input bg-background px-2 text-[11px]">
                  <option value="formal">formal 正式版</option>
                  <option value="trial">trial 体验版</option>
                  <option value="developer">developer 开发版</option>
                </select>
                <Input value={subPage2} onChange={(e) => setSubPage2(e.target.value)} placeholder="跳转页面 page (小程序内路径)" className="h-8 text-[11px] font-mono" />
                <Input value={subUrl} onChange={(e) => setSubUrl(e.target.value)} placeholder={subMode === 'bizsend' ? '跳转 url (服务号必填可选)' : 'lang 固定 zh_CN'} className="h-8 text-[11px]" />
              </div>

              <div className="space-y-1.5">
                <div className="font-semibold">模板内容填充 (共 {subSendFields.length} 个关键词)</div>
                {subSendFields.map((f) => (
                  <div key={f.key} className="flex items-center gap-2">
                    <span className="w-28 shrink-0 text-muted-foreground truncate" title={f.label}>{f.label}</span>
                    <span className="font-mono text-[10.5px] w-28 shrink-0">{f.key}</span>
                    <Input value={subSendValues[f.key] || ''} onChange={(e) => setSubSendValues((v) => ({ ...v, [f.key]: e.target.value }))}
                      placeholder={f.ex || '请输入内容'} className="h-8 text-[11px]" />
                  </div>
                ))}
              </div>

              <DialogFooter className="pt-2">
                <Button variant="outline" onClick={() => setSubSendOpen(false)}>取消</Button>
                <Button className="gap-1.5 bg-violet-600 hover:bg-violet-700" disabled={subSendMutation.isPending}
                  onClick={() => {
                    if (!subToUser.trim()) { toast.error('请填写接收者 openid'); return }
                    const empty = subSendFields.filter((f) => !(subSendValues[f.key] || '').trim())
                    if (empty.length) { toast.error(`请填写：${empty.map((e) => e.label).join('、')}`); return }
                    subSendMutation.mutate({
                      touser: subToUser.trim(),
                      template_id: subSendTmpl.priTmplId,
                      page: subPage2.trim() || undefined,
                      miniprogram_state: subState,
                      lang: subLang,
                      url: subUrl.trim() || undefined,
                      values: subSendValues,
                    })
                  }}>
                  <Send className="h-3.5 w-3.5" />{subSendMutation.isPending ? '发送中...' : '立即发送'}
                </Button>
              </DialogFooter>
            </div>
          )}
        </DialogContent>
      </Dialog>
    </div>
  )
}
