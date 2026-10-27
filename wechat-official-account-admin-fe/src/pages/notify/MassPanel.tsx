import React, { useMemo, useRef, useState } from 'react'
import { useQuery, useMutation } from '@tanstack/react-query'
import { api } from '@/lib/api'
import { useAppStore } from '@/stores/app'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Switch } from '@/components/ui/switch'
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from '@/components/ui/select'
import {
  Megaphone, Send, Eye, Trash2, Gauge, RefreshCw, Upload, FilePlus2,
  ShieldAlert, Users, CheckCircle2, XCircle, Clock, Info,
} from 'lucide-react'
import { toast } from 'sonner'

/** 群发消息类型 */
const MSG_TYPES: { value: string; label: string; hint: string }[] = [
  { value: 'text', label: '文本消息', hint: '直接发送纯文本，最长不超过 20 条字符限制以微信为准' },
  { value: 'mpnews', label: '图文消息', hint: '需提供图文素材 media_id (可用下方「新建图文素材」获得)' },
  { value: 'image', label: '贴图消息', hint: '需提供微信素材库中的图片 media_id，可多张 (逗号或换行分隔)' },
  { value: 'voice', label: '语音消息', hint: '需提供微信素材库中的语音 media_id' },
  { value: 'mpvideo', label: '视频消息', hint: '需提供微信素材库中的视频 media_id' },
  { value: 'wxcard', label: '卡券消息', hint: '需提供卡券 ID (card_id)' },
]

/** 群发速度档位 (微信官方定义) */
const SPEED_LEVELS: { value: string; label: string }[] = [
  { value: '0', label: '0 档 · 80 万/分钟' },
  { value: '1', label: '1 档 · 60 万/分钟' },
  { value: '2', label: '2 档 · 45 万/分钟' },
  { value: '3', label: '3 档 · 30 万/分钟' },
  { value: '4', label: '4 档 · 10 万/分钟' },
]

/** 群发任务状态字典 */
const MSG_STATUS: Record<string, { label: string; cls: string; icon: any }> = {
  SEND_SUCCESS: { label: '发送成功', cls: 'bg-emerald-500/10 text-emerald-600 border-emerald-500/30', icon: CheckCircle2 },
  SENDING: { label: '发送中', cls: 'bg-amber-500/10 text-amber-600 border-amber-500/30', icon: Clock },
  SEND_FAIL: { label: '发送失败', cls: 'bg-red-500/10 text-red-600 border-red-500/30', icon: XCircle },
  DELETE: { label: '已删除', cls: 'bg-muted text-muted-foreground border-border', icon: Trash2 },
}

/** 将多行/逗号分隔文本切成列表 */
function toList(text: string): string[] {
  return (text || '')
    .split(/[\n\r,，;；\s]+/)
    .map((s) => s.trim())
    .filter(Boolean)
}

/** 公众号群发消息面板 (message/mass/*) */
export const MassPanel: React.FC = () => {
  const { currentApp } = useAppStore()

  /* ---------------------- 群发内容 ---------------------- */
  const [msgType, setMsgType] = useState('text')
  const [textContent, setTextContent] = useState('')
  const [mediaId, setMediaId] = useState('')
  const [mediaIds, setMediaIds] = useState('')
  const [cardId, setCardId] = useState('')
  const [videoTitle, setVideoTitle] = useState('')
  const [videoDesc, setVideoDesc] = useState('')
  const [imgTitle, setImgTitle] = useState('')
  const [imgRecommend, setImgRecommend] = useState('')
  const [openComment, setOpenComment] = useState(false)
  const [fansCommentOnly, setFansCommentOnly] = useState(false)

  /* ---------------------- 接收者 ---------------------- */
  const [audience, setAudience] = useState<'all' | 'tag' | 'openid'>('all')
  const [tagId, setTagId] = useState('')
  const [openidText, setOpenidText] = useState('')
  const [clientMsgId, setClientMsgId] = useState('')
  const [ignoreReprint, setIgnoreReprint] = useState(true)
  const [previewOpenid, setPreviewOpenid] = useState('')
  const [previewWxname, setPreviewWxname] = useState('')

  /* ---------------------- 结果 / 查询 ---------------------- */
  const [sendResult, setSendResult] = useState<any>(null)
  const [queryMsgId, setQueryMsgId] = useState('')
  const [statusInfo, setStatusInfo] = useState<any>(null)
  const [delMsgId, setDelMsgId] = useState('')
  const [delUrl, setDelUrl] = useState('')
  const [delIdx, setDelIdx] = useState('0')
  const [speed, setSpeed] = useState('4')

  /* ---------------------- 图文素材准备 ---------------------- */
  const newsFileRef = useRef<HTMLInputElement>(null)
  const thumbFileRef = useRef<HTMLInputElement>(null)
  const [newsImgUrl, setNewsImgUrl] = useState('')
  const [thumbMediaId, setThumbMediaId] = useState('')
  const [nTitle, setNTitle] = useState('')
  const [nAuthor, setNAuthor] = useState('')
  const [nDigest, setNDigest] = useState('')
  const [nSourceUrl, setNSourceUrl] = useState('')
  const [nContent, setNContent] = useState('')

  const { data: tags } = useQuery<any[]>({
    queryKey: ['fans-tags', currentApp?.appid],
    queryFn: () => api.get('/api/fans/tags'),
    enabled: !!currentApp,
  })

  const { data: fans, refetch: refetchFans } = useQuery<any>({
    queryKey: ['notify', currentApp?.appid, 'mass-fans'],
    queryFn: () => api.get('/api/fans?page=1&page_size=100'),
    enabled: !!currentApp && audience === 'openid',
  })

  const { data: speedInfo, refetch: refetchSpeed } = useQuery<any>({
    queryKey: ['notify', currentApp?.appid, 'mass-speed'],
    queryFn: () => api.get('/api/notify/mass/speed'),
    enabled: !!currentApp,
  })

  const selectedOpenIDs = useMemo(() => toList(openidText), [openidText])
  const currentType = MSG_TYPES.find((t) => t.value === msgType)!

  const buildMessage = () => ({
    msgtype: msgType,
    text: textContent,
    media_id: mediaId,
    media_ids: toList(mediaIds),
    card_id: cardId,
    title: msgType === 'image' ? imgTitle : videoTitle,
    description: videoDesc,
    recommend: imgRecommend,
    need_open_comment: openComment ? 1 : 0,
    only_fans_can_comment: fansCommentOnly ? 1 : 0,
  })

  const contentReady = () => {
    if (msgType === 'text') return !!textContent.trim()
    if (msgType === 'wxcard') return !!cardId.trim()
    if (msgType === 'image') return !!toList(mediaIds).length || !!mediaId.trim()
    return !!mediaId.trim()
  }

  const validateAudience = (): string => {
    if (audience === 'tag') return tagId ? '' : '请选择要群发的粉丝标签'
    if (audience === 'openid') {
      if (selectedOpenIDs.length < 2) return 'openid 列表至少需要 2 个'
      if (selectedOpenIDs.length > 10000) return 'openid 列表最多 10000 个'
      return ''
    }
    return ''
  }

  const guard = (): boolean => {
    if (!contentReady()) {
      toast.error(`请按提示补全${currentType.label}的内容`)
      return false
    }
    const err = validateAudience()
    if (err) {
      toast.error(err)
      return false
    }
    if (clientMsgId.length > 32) {
      toast.error('clientmsgid 长度不能超过 32 字节')
      return false
    }
    return true
  }

  const massMutation = useMutation({
    mutationFn: (v: any) =>
      api.post(audience === 'openid' ? '/api/notify/mass/send' : '/api/notify/mass/sendall', v),
    onSuccess: (d: any) => {
      const r = d?.result || {}
      setSendResult(r)
      if (r.msg_id) setQueryMsgId(String(r.msg_id))
      toast.success(d?.message || '群发任务已提交')
    },
  })

  const previewMutation = useMutation({
    mutationFn: (v: any) => api.post('/api/notify/mass/preview', v),
    onSuccess: (d: any) => toast.success(d?.message || '预览已发送'),
  })

  const statusMutation = useMutation({
    mutationFn: (msgId: string) => api.post('/api/notify/mass/status', { msg_id: msgId }),
    onSuccess: (d: any) => setStatusInfo(d),
  })

  const deleteMutation = useMutation({
    mutationFn: (v: any) => api.post('/api/notify/mass/delete', v),
    onSuccess: (d: any) => {
      toast.success(d?.message || '已删除')
      setDelMsgId('')
      setDelUrl('')
    },
  })

  const speedMutation = useMutation({
    mutationFn: (v: any) => api.post('/api/notify/mass/speed', v),
    onSuccess: (d: any) => {
      toast.success(d?.message || '设置成功')
      refetchSpeed()
    },
  })

  const uploadImgMutation = useMutation({
    mutationFn: (fd: FormData) => api.post('/api/notify/mass/uploadimg', fd),
    onSuccess: (d: any) => {
      setNewsImgUrl(d?.url || '')
      toast.success('正文图片已上传 (不占素材库配额)')
    },
  })

  const uploadThumbMutation = useMutation({
    mutationFn: (fd: FormData) => api.post('/api/notify/mass/uploadthumb', fd),
    onSuccess: (d: any) => {
      setThumbMediaId(d?.media_id || '')
      toast.success('封面已存入微信素材库，media_id 已填入')
    },
  })

  const uploadNewsMutation = useMutation({
    mutationFn: (v: any) => api.post('/api/notify/mass/uploadnews', v),
    onSuccess: (d: any) => {
      const id = d?.media_id || ''
      setMediaId(id)
      setMsgType('mpnews')
      toast.success(`图文素材已创建: ${id}`)
    },
  })

  const pickFile = (ref: any, fdName: string, m: any) => (e: React.ChangeEvent<HTMLInputElement>) => {
    const f = e.target.files?.[0]
    if (!f) return
    const fd = new FormData()
    fd.append(fdName, f, f.name)
    m.mutate(fd)
    e.target.value = ''
  }

  const toggleOpenid = (openid: string) => {
    const list = toList(openidText)
    const next = list.includes(openid) ? list.filter((x) => x !== openid) : [...list, openid]
    setOpenidText(next.join('\n'))
  }

  const statusDict = statusInfo?.msg_status ? MSG_STATUS[statusInfo.msg_status] : null

  if (!currentApp) {
    return (
      <Card>
        <CardContent className="py-12 text-center text-sm text-muted-foreground">
          请先在右上角选择公众号
        </CardContent>
      </Card>
    )
  }

  return (
    <div className="space-y-5">
      {/* 规则提示 */}
      <div className="p-4 rounded-xl bg-sky-500/10 border border-sky-500/20 text-xs text-sky-900 dark:text-sky-200 flex items-start gap-3">
        <ShieldAlert className="h-5 w-5 text-sky-600 shrink-0 mt-0.5" />
        <div className="space-y-1 leading-relaxed">
          <div className="font-semibold text-sm">群发消息使用要点 (仅认证公众号/服务号)</div>
          <div>
            1. 频次限制：<strong>订阅号每天 1 次</strong>，<strong>认证服务号每月 4 次</strong>，超出返回错误码 44004。<br />
            2. 第三方平台代群发需勾选<strong>「群发消息」权限集 (id=7)</strong>；上传素材另需素材类权限集。<br />
            3. 图文群发前微信会做<strong>原创校验</strong>；被判为转载时，勾选下方「转载仍继续群发」才会发送。<br />
            4. 若账号开启了「API 群发保护」，<strong>群发全部用户需管理员在 30 分钟内确认</strong>，否则该次群发失败。<br />
            5. 群发为异步任务，接口返回成功仅代表<strong>任务已提交</strong>，最终结果请用 msg_id 查询或查看 MASSSENDJOBFINISH 事件回调。
          </div>
        </div>
      </div>

      {currentApp.account_type === 'miniprogram' && (
        <div className="p-3 rounded-xl bg-amber-500/10 border border-amber-500/25 text-xs text-amber-900 dark:text-amber-200 flex items-center gap-2">
          <Info className="h-4 w-4 shrink-0" />
          当前账号是小程序，微信未向小程序开放「群发消息」接口，请在顶部切换到公众号后再操作。
        </div>
      )}

      <div className="grid lg:grid-cols-2 gap-5">
        {/* 群发内容 */}
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-base flex items-center gap-2">
              <Megaphone className="h-4 w-4 text-sky-600" />1) 群发内容
            </CardTitle>
            <CardDescription className="text-xs">{currentType.hint}</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3 text-xs">
            <Select value={msgType} onValueChange={setMsgType}>
              <SelectTrigger className="h-9 text-xs">
                <SelectValue placeholder="消息类型" />
              </SelectTrigger>
              <SelectContent>
                {MSG_TYPES.map((t) => (
                  <SelectItem key={t.value} value={t.value} className="text-xs">{t.label}</SelectItem>
                ))}
              </SelectContent>
            </Select>

            {msgType === 'text' && (
              <textarea
                className="w-full min-h-[96px] rounded-md border bg-background px-3 py-2 text-xs focus:outline-none focus:ring-2 focus:ring-ring"
                placeholder="请输入群发的文本内容"
                value={textContent}
                onChange={(e) => setTextContent(e.target.value)}
              />
            )}

            {msgType !== 'text' && msgType !== 'wxcard' && (
              <div className="space-y-2">
                <Input value={mediaId} onChange={(e) => setMediaId(e.target.value)}
                  placeholder={msgType === 'mpnews' ? '图文素材 media_id' : '素材 media_id (微信素材库)'}
                  className="h-9 text-xs font-mono" />
                <div className="text-[11px] text-muted-foreground">
                  可用下方「3) 图文素材准备」一键创建 mpnews 素材，或粘贴你在微信素材库中的 media_id。
                  本系统素材库中 <code className="font-mono">mock_</code> 开头的 ID 为本地占位，不能用于群发。
                </div>
              </div>
            )}

            {msgType === 'image' && (
              <div className="space-y-2">
                <textarea
                  className="w-full min-h-[64px] rounded-md border bg-background px-3 py-2 text-xs font-mono focus:outline-none focus:ring-2 focus:ring-ring"
                  placeholder="多个图片 media_id，用换行或逗号分隔"
                  value={mediaIds}
                  onChange={(e) => setMediaIds(e.target.value)}
                />
                <div className="grid grid-cols-2 gap-2">
                  <Input value={imgTitle} onChange={(e) => setImgTitle(e.target.value)} placeholder="贴图标题 title" className="h-9 text-xs" />
                  <Input value={imgRecommend} onChange={(e) => setImgRecommend(e.target.value)} placeholder="推荐语 recommend" className="h-9 text-xs" />
                </div>
              </div>
            )}

            {msgType === 'mpvideo' && (
              <div className="grid grid-cols-2 gap-2">
                <Input value={videoTitle} onChange={(e) => setVideoTitle(e.target.value)} placeholder="视频标题 (可选)" className="h-9 text-xs" />
                <Input value={videoDesc} onChange={(e) => setVideoDesc(e.target.value)} placeholder="视频描述 (可选)" className="h-9 text-xs" />
              </div>
            )}

            {msgType === 'wxcard' && (
              <Input value={cardId} onChange={(e) => setCardId(e.target.value)} placeholder="卡券 ID (card_id)" className="h-9 text-xs font-mono" />
            )}

            {(msgType === 'mpnews' || msgType === 'image') && (
              <div className="flex flex-wrap gap-4 pt-1">
                <label className="flex items-center gap-2">
                  <Switch checked={openComment} onCheckedChange={setOpenComment} />
                  <span>开启评论</span>
                </label>
                <label className="flex items-center gap-2">
                  <Switch checked={fansCommentOnly} onCheckedChange={setFansCommentOnly} />
                  <span>仅粉丝可评论</span>
                </label>
              </div>
            )}
          </CardContent>
        </Card>

        {/* 接收者 */}
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-base flex items-center gap-2">
              <Users className="h-4 w-4 text-sky-600" />2) 接收者与防重
            </CardTitle>
            <CardDescription className="text-xs">
              全体粉丝 / 指定标签 / 指定 openid 列表 (2~10000 个)
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-3 text-xs">
            <div className="flex flex-wrap gap-2">
              {[
                { v: 'all', label: '全部用户' },
                { v: 'tag', label: '按标签' },
                { v: 'openid', label: '指定 openid' },
              ].map((o) => (
                <button key={o.v} type="button" onClick={() => setAudience(o.v as any)}
                  className={`px-3 py-1.5 rounded-lg border text-[11px] transition ${
                    audience === o.v ? 'bg-sky-600 text-white border-sky-600' : 'bg-background hover:border-primary'
                  }`}>
                  {o.label}
                </button>
              ))}
            </div>

            {audience === 'tag' && (
              <Select value={tagId} onValueChange={setTagId}>
                <SelectTrigger className="h-9 text-xs">
                  <SelectValue placeholder="选择粉丝标签" />
                </SelectTrigger>
                <SelectContent>
                  {(tags || []).map((t: any) => (
                    <SelectItem key={t.tag_id} value={String(t.tag_id)} className="text-xs">
                      {t.name} (tag_id={t.tag_id}，粉丝 {t.count || 0})
                    </SelectItem>
                  ))}
                  {!tags?.length && (
                    <SelectItem value="__none" className="text-xs" disabled>暂无标签，请先在「粉丝与标签管理」创建</SelectItem>
                  )}
                </SelectContent>
              </Select>
            )}

            {audience === 'openid' && (
              <div className="space-y-2">
                <textarea
                  className="w-full min-h-[80px] rounded-md border bg-background px-3 py-2 text-xs font-mono focus:outline-none focus:ring-2 focus:ring-ring"
                  placeholder="每行一个 openid (也可从下方粉丝列表点选)"
                  value={openidText}
                  onChange={(e) => setOpenidText(e.target.value)}
                />
                <div className="flex items-center justify-between">
                  <span className="text-[11px] text-muted-foreground">已选 {selectedOpenIDs.length} 个</span>
                  <div className="flex gap-2">
                    <Button size="sm" variant="ghost" className="h-7 text-[11px]" onClick={() => setOpenidText('')}>清空</Button>
                    <Button size="sm" variant="outline" className="h-7 text-[11px] gap-1" onClick={() => refetchFans()}>
                      <RefreshCw className="h-3 w-3" />刷新粉丝
                    </Button>
                  </div>
                </div>
                <div className="max-h-44 overflow-y-auto rounded-lg border bg-muted/30 divide-y">
                  {(fans?.list || []).map((f: any) => (
                    <button key={f.openid} type="button" onClick={() => toggleOpenid(f.openid)}
                      className={`w-full flex items-center justify-between px-2 py-1.5 text-[11px] text-left hover:bg-muted ${
                        selectedOpenIDs.includes(f.openid) ? 'bg-sky-500/15' : ''
                      }`}>
                      <span className="truncate">{f.remark || f.nickname || '未命名粉丝'}</span>
                      <span className="font-mono text-muted-foreground shrink-0 ml-2">{f.openid.slice(-10)}</span>
                    </button>
                  ))}
                  {!fans?.list?.length && (
                    <div className="p-4 text-center text-muted-foreground">
                      暂无粉丝数据，请先在「粉丝与标签管理」中同步粉丝
                    </div>
                  )}
                </div>
              </div>
            )}

            <div className="space-y-1.5 pt-1">
              <Input value={clientMsgId} onChange={(e) => setClientMsgId(e.target.value)}
                placeholder="clientmsgid (选填，24 小时内重复提交会被微信合并，用于防重复群发)"
                className="h-9 text-xs font-mono" />
              <label className="flex items-center gap-2">
                <Switch checked={ignoreReprint} onCheckedChange={setIgnoreReprint} />
                <span>图文被判为转载时仍继续群发 (send_ignore_reprint)</span>
              </label>
            </div>

            <div className="grid grid-cols-2 gap-2 pt-1">
              <Input value={previewOpenid} onChange={(e) => setPreviewOpenid(e.target.value)}
                placeholder="预览 openid (不群发，仅测试)" className="h-9 text-xs font-mono" />
              <Input value={previewWxname} onChange={(e) => setPreviewWxname(e.target.value)}
                placeholder="或微信号 (towxname)" className="h-9 text-xs" />
            </div>

            <div className="flex flex-wrap gap-2 pt-1">
              <Button size="sm" className="gap-1.5" disabled={massMutation.isPending}
                onClick={() => {
                  if (!guard()) return
                  const msg = buildMessage()
                  massMutation.mutate(
                    audience === 'openid'
                      ? { ...msg, touser: selectedOpenIDs, clientmsgid: clientMsgId, send_ignore_reprint: ignoreReprint ? 1 : 0 }
                      : { ...msg, is_to_all: audience === 'all', tag_id: tagId, clientmsgid: clientMsgId, send_ignore_reprint: ignoreReprint ? 1 : 0 }
                  )
                }}>
              <Send className="h-4 w-4" />{massMutation.isPending ? '提交中...' : '提交群发任务'}
            </Button>
              <Button size="sm" variant="outline" className="gap-1.5" disabled={previewMutation.isPending}
                onClick={() => {
                  if (!contentReady()) {
                    toast.error('请先补全消息内容')
                    return
                  }
                  if (!previewOpenid.trim() && !previewWxname.trim()) {
                    toast.error('请填写预览接收 openid 或微信号')
                    return
                  }
                  previewMutation.mutate({
                    ...buildMessage(),
                    touser: previewOpenid.trim(),
                    towxname: previewWxname.trim(),
                  })
                }}>
              <Eye className="h-4 w-4" />仅预览
            </Button>
            </div>

            {sendResult && (
              <div className="p-3 rounded-lg bg-emerald-500/10 border border-emerald-500/25 text-[11px] space-y-1">
                <div className="font-semibold text-emerald-700 dark:text-emerald-300">群发任务已提交</div>
                <div className="font-mono">msg_id: {String(sendResult.msg_id ?? '-')}</div>
                {sendResult.msg_data_id !== undefined && (
                  <div className="font-mono">msg_data_id: {String(sendResult.msg_data_id)}</div>
                )}
                {sendResult.clientmsgid && (
                  <div className="font-mono">clientmsgid: {String(sendResult.clientmsgid)}</div>
                )}
                <div className="text-muted-foreground">范围: {sendResult.audience || '-'}</div>
              </div>
            )}
          </CardContent>
        </Card>

        {/* 图文素材准备 */}
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-base flex items-center gap-2">
              <FilePlus2 className="h-4 w-4 text-sky-600" />3) 图文素材准备
            </CardTitle>
            <CardDescription className="text-xs">
              正文图片 uploadimg (不占素材库) · 封面 uploadthumb (永久素材) · 新建图文 uploadnews
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-3 text-xs">
            <div className="grid grid-cols-2 gap-2">
              <div>
                <input ref={newsFileRef} type="file" accept="image/jpeg,image/png" className="hidden"
                  onChange={pickFile(newsFileRef, 'file', uploadImgMutation)} />
                <Button size="sm" variant="outline" className="w-full h-8 text-[11px] gap-1.5"
                  disabled={uploadImgMutation.isPending} onClick={() => newsFileRef.current?.click()}>
                  <Upload className="h-3.5 w-3.5" />{uploadImgMutation.isPending ? '上传中...' : '上传正文图片'}
                </Button>
              </div>
              <div>
                <input ref={thumbFileRef} type="file" accept="image/jpeg,image/png,image/gif" className="hidden"
                  onChange={pickFile(thumbFileRef, 'file', uploadThumbMutation)} />
                <Button size="sm" variant="outline" className="w-full h-8 text-[11px] gap-1.5"
                  disabled={uploadThumbMutation.isPending} onClick={() => thumbFileRef.current?.click()}>
                  <Upload className="h-3.5 w-3.5" />{uploadThumbMutation.isPending ? '上传中...' : '上传图文封面'}
                </Button>
              </div>
            </div>
            {newsImgUrl && (
              <div className="space-y-1">
                <div className="text-[11px] text-muted-foreground">正文图片 URL (可直接插入图文正文):</div>
                <div className="flex gap-2">
                  <Input readOnly value={newsImgUrl} className="h-8 text-[11px] font-mono" onFocus={(e) => e.target.select()} />
                </div>
                <img src={newsImgUrl} alt="正文图片预览" className="max-h-32 rounded border" />
              </div>
            )}
            {thumbMediaId && (
              <div className="text-[11px]">
                <span className="text-muted-foreground">封面 media_id: </span>
                <code className="font-mono bg-muted px-1.5 py-0.5 rounded break-all">{thumbMediaId}</code>
              </div>
            )}

            <div className="border-t pt-3 space-y-2">
              <div className="grid grid-cols-2 gap-2">
                <Input value={nTitle} onChange={(e) => setNTitle(e.target.value)} placeholder="标题 (必填)" className="h-9 text-xs" />
                <Input value={nAuthor} onChange={(e) => setNAuthor(e.target.value)} placeholder="作者 (选填)" className="h-9 text-xs" />
              </div>
              <div className="grid grid-cols-2 gap-2">
                <Input value={thumbMediaId} onChange={(e) => setThumbMediaId(e.target.value)}
                  placeholder="封面 thumb_media_id (必填)" className="h-9 text-xs font-mono" />
                <Input value={nSourceUrl} onChange={(e) => setNSourceUrl(e.target.value)} placeholder="阅读原文链接 (选填)" className="h-9 text-xs" />
              </div>
              <Input value={nDigest} onChange={(e) => setNDigest(e.target.value)} placeholder="摘要 (选填，留空自动抓取正文前 64 字)" className="h-9 text-xs" />
              <textarea
                className="w-full min-h-[110px] rounded-md border bg-background px-3 py-2 text-xs focus:outline-none focus:ring-2 focus:ring-ring"
                placeholder="正文内容 (必填，支持 HTML，可用上方正文图片 URL 插入 <img>)"
                value={nContent}
                onChange={(e) => setNContent(e.target.value)}
              />
              <div className="flex items-center justify-between">
                <span className="text-[11px] text-muted-foreground">
                  一条图文消息最多 8 篇，此处创建单篇图文；群发前请确认封面为微信素材库真实 media_id。
                </span>
                <Button size="sm" className="h-8 gap-1.5" disabled={uploadNewsMutation.isPending}
                  onClick={() => {
                    if (!nTitle.trim() || !nContent.trim() || !thumbMediaId.trim()) {
                      toast.error('标题、正文与封面 thumb_media_id 均为必填')
                      return
                    }
                    uploadNewsMutation.mutate({
                      articles: [{
                        title: nTitle,
                        author: nAuthor,
                        thumb_media_id: thumbMediaId,
                        content: nContent,
                        digest: nDigest,
                        content_source_url: nSourceUrl,
                        show_cover_pic: 1,
                        need_open_comment: openComment ? 1 : 0,
                        only_fans_can_comment: fansCommentOnly ? 1 : 0,
                      }],
                    })
                  }}>
                <FilePlus2 className="h-3.5 w-3.5" />生成图文 media_id
              </Button>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* 状态查询 / 删除 / 速度 */}
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-base flex items-center gap-2">
              <Gauge className="h-4 w-4 text-sky-600" />4) 任务状态、撤回与速率
            </CardTitle>
            <CardDescription className="text-xs">
              查询 msg_id 状态 · 删除未发送部分 · 设置群发速度
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-3 text-xs">
            <div className="flex gap-2">
              <Input value={queryMsgId} onChange={(e) => setQueryMsgId(e.target.value)}
                placeholder="群发消息 msg_id" className="h-9 text-xs font-mono" />
              <Button size="sm" variant="outline" className="h-9 shrink-0 gap-1.5"
                disabled={!queryMsgId.trim() || statusMutation.isPending}
                onClick={() => statusMutation.mutate(queryMsgId.trim())}>
                <RefreshCw className="h-3.5 w-3.5" />查询
              </Button>
            </div>

            {statusInfo && (
              <div className="p-3 rounded-lg bg-muted/40 border space-y-1">
                <div className="flex items-center gap-2">
                  <span className="font-mono text-[11px]">msg_id: {String(statusInfo.msg_id ?? queryMsgId)}</span>
                  {statusDict ? (
                    <Badge variant="outline" className={`gap-1 text-[10px] ${statusDict.cls}`}>
                      <statusDict.icon className="h-3 w-3" />{statusDict.label}
                    </Badge>
                  ) : (
                    <Badge variant="outline" className="text-[10px]">{String(statusInfo.msg_status || '未知状态')}</Badge>
                  )}
                </div>
                {(statusInfo.send_count !== undefined || statusInfo.filter_count !== undefined) && (
                  <div className="font-mono text-[11px] text-muted-foreground">
                    发送 {String(statusInfo.send_count ?? '-')} / 筛选 {String(statusInfo.filter_count ?? '-')}
                  </div>
                )}
                <pre className="text-[10px] font-mono text-muted-foreground whitespace-pre-wrap break-all max-h-32 overflow-y-auto">
{JSON.stringify(statusInfo, null, 2)}
                </pre>
              </div>
            )}

            <div className="border-t pt-3 space-y-2">
              <div className="grid grid-cols-3 gap-2">
                <Input value={delMsgId} onChange={(e) => setDelMsgId(e.target.value)} placeholder="msg_id" className="h-9 text-xs font-mono" />
                <Input value={delUrl} onChange={(e) => setDelUrl(e.target.value)} placeholder="或图文文章 URL" className="h-9 text-xs" />
                <Input value={delIdx} onChange={(e) => setDelIdx(e.target.value)} placeholder="article_idx (0=全部)" className="h-9 text-xs" />
              </div>
              <Button size="sm" variant="destructive" className="h-8 gap-1.5"
                disabled={!delMsgId.trim() && !delUrl.trim()}
                onClick={() =>
                  deleteMutation.mutate({
                    msg_id: delMsgId.trim(),
                    url: delUrl.trim(),
                    article_idx: parseInt(delIdx || '0', 10) || 0,
                  })
                }>
                <Trash2 className="h-3.5 w-3.5" />删除群发消息
              </Button>
              <div className="text-[11px] text-muted-foreground">
                删除仅对尚未收到的用户生效，已送达的消息无法撤回。
              </div>
            </div>

            <div className="border-t pt-3 space-y-2">
              <div className="flex items-center justify-between">
                <span>当前群发速度</span>
                <span className="font-mono text-[11px]">
                  {speedInfo?.speed !== undefined ? `${speedInfo.speed} 档` : '未知'}
                  {speedInfo?.realspeed ? ` · 实际 ${speedInfo.realspeed} 万/分钟` : ''}
                </span>
              </div>
              <div className="flex gap-2">
                <Select value={speed} onValueChange={setSpeed}>
                  <SelectTrigger className="h-9 text-xs">
                    <SelectValue placeholder="速度档位" />
                  </SelectTrigger>
                  <SelectContent>
                    {SPEED_LEVELS.map((l) => (
                      <SelectItem key={l.value} value={l.value} className="text-xs">{l.label}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <Button size="sm" variant="outline" className="h-9 shrink-0" disabled={speedMutation.isPending}
                  onClick={() => speedMutation.mutate({ speed: parseInt(speed, 10) })}>
                  设置
                </Button>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
