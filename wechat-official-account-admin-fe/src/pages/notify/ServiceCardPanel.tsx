import React, { useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { api } from '@/lib/api'
import { useAppStore } from '@/stores/app'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import {
  CreditCard, Search, Upload, Info, ShieldAlert, CheckCircle2, AlertCircle, Ban, RefreshCw,
} from 'lucide-react'
import { toast } from 'sonner'
import { formatDate } from '@/lib/utils'

/** code_state 字典 (wxa/get_user_notify) */
const CODE_STATE: Record<number, { label: string; cls: string; icon: any }> = {
  0: { label: '正常', cls: 'bg-emerald-500/10 text-emerald-600 border-emerald-500/30', icon: CheckCircle2 },
  1: { label: '有风险', cls: 'bg-amber-500/10 text-amber-600 border-amber-500/30', icon: AlertCircle },
  2: { label: '异常', cls: 'bg-red-500/10 text-red-600 border-red-500/30', icon: AlertCircle },
  10: { label: '用户拒收本次 code', cls: 'bg-muted text-muted-foreground border-border', icon: Ban },
}

const EXAMPLE_1001 = '{"cur_status":2,"license_plate":"粤A12345A","arrival_time":1679569348,"wxa_path_query":""}'
const EXAMPLE_2001 =
  '{"cur_status":2,"product_count":1,"product_list":{"info_list":[{"product_img":"https://res.wx.qq.com/op_res/demo","product_name":"示例商品","product_path_query":"pages/index/index"}]},"send_time":1696157643,"wxa_path_query":"pages/index/index"}'
const EXAMPLE_CHECK = '{"pay_amount":1005,"pay_time":1683525070}'

/** 服务提醒卡片 (订阅消息·服务卡片) 面板 */
export const ServiceCardPanel: React.FC = () => {
  const { currentApp } = useAppStore()
  const [openid, setOpenid] = useState('')
  const [notifyType, setNotifyType] = useState('1001')
  const [notifyCode, setNotifyCode] = useState('')
  const [contentJson, setContentJson] = useState('')
  const [checkJson, setCheckJson] = useState('')
  const [queried, setQueried] = useState<any>(null)

  const setMutation = useMutation({
    mutationFn: (v: any) => api.post('/api/notify/sub/service-card', v),
    onSuccess: (d: any) => toast.success(d?.message || '服务卡片已激活/更新'),
  })

  const getMutation = useMutation({
    mutationFn: (v: any) => api.post('/api/notify/sub/service-card/query', v),
    onSuccess: (d: any) => {
      setQueried(d)
      toast.success('已获取卡片状态')
    },
  })

  const validate = (): boolean => {
    if (!openid.trim()) {
      toast.error('请填写用户 openid')
      return false
    }
    if (!notifyCode.trim()) {
      toast.error('请填写 notify_code (前端 button 获取的动态令牌，或微信支付订单号)')
      return false
    }
    if (!/^\d+$/.test(notifyType.trim())) {
      toast.error('卡片 id (notify_type) 必须是数字')
      return false
    }
    return true
  }

  if (!currentApp) {
    return (
      <Card>
        <CardContent className="py-12 text-center text-sm text-muted-foreground">
          请先在右上角选择小程序或服务号
        </CardContent>
      </Card>
    )
  }

  const info = queried?.notify_info || null
  const stateDict = info && info.code_state !== undefined ? CODE_STATE[Number(info.code_state)] : null

  return (
    <div className="space-y-5">
      <div className="p-4 rounded-xl bg-orange-500/10 border border-orange-500/20 text-xs text-orange-900 dark:text-orange-200 flex items-start gap-3">
        <ShieldAlert className="h-5 w-5 text-orange-600 shrink-0 mt-0.5" />
        <div className="space-y-1 leading-relaxed">
          <div className="font-semibold text-sm">服务提醒卡片使用要点 (小程序 / 小游戏)</div>
          <div>
            1. 服务卡片是「订阅消息 · 服务提醒」能力：先用 <code>notify_code</code> <strong>激活</strong>卡片，之后每次状态变化都调用同一接口<strong>更新</strong>，微信按卡片定义向用户推送提醒。<br />
            2. <code>notify_code</code> 由小程序前端 <code>button</code> 组件获取，或直接使用<strong>微信支付订单号</strong> (此时需填 <code>check_json</code>，含 pay_amount 分为单位与 pay_time 秒级时间戳)。<br />
            3. <code>content_json</code> 的字段由所选卡片模板决定 (如 1001、2001 两类卡片字段完全不同)，务必按官方「服务提醒」文档中该卡片的定义填写，否则会返回 85434 (invalid notify_type) 等错误。<br />
            4. 第三方代调用需勾选<strong>权限集 id=18</strong>。
          </div>
        </div>
      </div>

      <div className="grid lg:grid-cols-2 gap-5">
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-base flex items-center gap-2">
              <CreditCard className="h-4 w-4 text-orange-600" />激活 / 更新卡片
            </CardTitle>
            <CardDescription className="text-xs">wxa/set_user_notify</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3 text-xs">
            <div className="grid grid-cols-2 gap-2">
              <Input value={openid} onChange={(e) => setOpenid(e.target.value)} placeholder="用户 openid" className="h-9 text-xs font-mono" />
              <Input value={notifyType} onChange={(e) => setNotifyType(e.target.value)} placeholder="卡片 id notify_type" className="h-9 text-xs font-mono" />
            </div>
            <Input value={notifyCode} onChange={(e) => setNotifyCode(e.target.value)}
              placeholder="动态更新令牌 notify_code (或微信支付订单号)" className="h-9 text-xs font-mono" />
            <div className="space-y-1">
              <div className="flex items-center justify-between">
                <span className="text-muted-foreground">content_json (卡片状态)</span>
                <div className="flex gap-1">
                  <Button size="sm" variant="ghost" className="h-6 text-[10px]"
                    onClick={() => { setContentJson(EXAMPLE_1001); setNotifyType('1001'); setCheckJson('') }}>示例 1001</Button>
                  <Button size="sm" variant="ghost" className="h-6 text-[10px]"
                    onClick={() => { setContentJson(EXAMPLE_2001); setNotifyType('2001'); setCheckJson(EXAMPLE_CHECK) }}>示例 2001</Button>
                </div>
              </div>
              <textarea
                className="w-full min-h-[96px] rounded-md border bg-background px-3 py-2 text-xs font-mono focus:outline-none focus:ring-2 focus:ring-ring"
                placeholder='{"cur_status":2,...}'
                value={contentJson}
                onChange={(e) => setContentJson(e.target.value)}
              />
            </div>
            <div className="space-y-1">
              <span className="text-muted-foreground">check_json (仅微信支付订单号作为 code 时填写)</span>
              <textarea
                className="w-full min-h-[52px] rounded-md border bg-background px-3 py-2 text-xs font-mono focus:outline-none focus:ring-2 focus:ring-ring"
                placeholder='{"pay_amount":1005,"pay_time":1683525070}'
                value={checkJson}
                onChange={(e) => setCheckJson(e.target.value)}
              />
            </div>
            <Button size="sm" className="gap-1.5" disabled={setMutation.isPending}
              onClick={() => {
                if (!validate()) return
                if (!contentJson.trim()) {
                  toast.error('请填写卡片内容 content_json')
                  return
                }
                try {
                  JSON.parse(contentJson)
                } catch {
                  toast.error('content_json 不是合法 JSON')
                  return
                }
                if (checkJson.trim()) {
                  try {
                    JSON.parse(checkJson)
                  } catch {
                    toast.error('check_json 不是合法 JSON')
                    return
                  }
                }
                setMutation.mutate({
                  openid: openid.trim(),
                  notify_type: notifyType.trim(),
                  notify_code: notifyCode.trim(),
                  content_json: contentJson,
                  check_json: checkJson.trim(),
                })
              }}>
              <Upload className="h-4 w-4" />{setMutation.isPending ? '提交中...' : '激活 / 更新服务卡片'}
            </Button>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-base flex items-center gap-2">
              <Search className="h-4 w-4 text-orange-600" />查询卡片状态
            </CardTitle>
            <CardDescription className="text-xs">wxa/get_user_notify · 校验 code 是否仍有效</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3 text-xs">
            <Button size="sm" variant="outline" className="gap-1.5" disabled={getMutation.isPending || !notifyCode.trim()}
              onClick={() => {
                if (!validate()) return
                getMutation.mutate({
                  openid: openid.trim(),
                  notify_type: parseInt(notifyType.trim(), 10),
                  notify_code: notifyCode.trim(),
                })
              }}>
              <RefreshCw className={`h-3.5 w-3.5 ${getMutation.isPending ? 'animate-spin' : ''}`} />查询
            </Button>

            {info && (
              <div className="p-3 rounded-lg bg-muted/40 border space-y-2">
                <div className="flex items-center gap-2">
                  <span className="font-mono">notify_type: {String(info.notify_type)}</span>
                  {stateDict ? (
                    <Badge variant="outline" className={`gap-1 text-[10px] ${stateDict.cls}`}>
                      <stateDict.icon className="h-3 w-3" />{stateDict.label}
                    </Badge>
                  ) : (
                    <Badge variant="outline" className="text-[10px]">state: {String(info.code_state)}</Badge>
                  )}
                </div>
                <div className="text-[11px] text-muted-foreground">
                  code 过期时间：{info.code_expire_time ? formatDate(new Date(Number(info.code_expire_time) * 1000)) : '-'}
                </div>
                <div className="space-y-1">
                  <div className="text-[11px] text-muted-foreground">上次有效推送内容 content_json：</div>
                  <pre className="text-[10px] font-mono whitespace-pre-wrap break-all bg-background rounded border p-2 max-h-48 overflow-y-auto">
                    {info.content_json ? prettyJson(info.content_json) : '(该卡片尚未推送过内容)'}
                  </pre>
                </div>
                <div className="flex gap-1.5">
                  <Button size="sm" variant="ghost" className="h-7 text-[11px]"
                    onClick={() => info.content_json && setContentJson(info.content_json)}>
                    回填到更新表单
                  </Button>
                </div>
              </div>
            )}

            {queried && !info && (
              <div className="p-3 rounded-lg bg-muted/40 border text-[11px] flex items-center gap-2 text-muted-foreground">
                <Info className="h-4 w-4" />微信未返回 notify_info，请确认 notify_code 与卡片 id 是否正确。
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}

function prettyJson(s: string) {
  try {
    return JSON.stringify(JSON.parse(s), null, 2)
  } catch {
    return s
  }
}
