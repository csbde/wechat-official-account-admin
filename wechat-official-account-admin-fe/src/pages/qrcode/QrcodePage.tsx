import React, { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api, PageResult } from '@/lib/api'
import { useAppStore } from '@/stores/app'
import { Button } from '@/components/ui/button'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
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
  QrCode,
  Plus,
  Download,
  Clock,
  Infinity as InfinityIcon,
  Copy,
  ExternalLink,
} from 'lucide-react'
import { toast } from 'sonner'
import { formatDate } from '@/lib/utils'

export const QrcodePage: React.FC = () => {
  const queryClient = useQueryClient()
  const { currentApp } = useAppStore()

  const [dialogOpen, setDialogOpen] = useState(false)
  const [qrcodeType, setQrcodeType] = useState('temp')
  const [sceneStr, setSceneStr] = useState('')
  const [expireDays, setExpireDays] = useState(30)
  const [remark, setRemark] = useState('')

  const { data: qrcodes, isLoading } = useQuery<PageResult<any>>({
    queryKey: ['qrcodes', currentApp?.appid],
    queryFn: () => api.get('/api/qrcode?page=1&page_size=20'),
    enabled: !!currentApp,
  })

  const createMutation = useMutation({
    mutationFn: (values: any) => api.post('/api/qrcode', values),
    onSuccess: () => {
      toast.success('带参二维码生成成功！')
      queryClient.invalidateQueries({ queryKey: ['qrcodes'] })
      setDialogOpen(false)
      setSceneStr('')
      setRemark('')
    },
    onError: (err: any) => {
      toast.error('生成失败: ' + (err.message || '微信接口报错'))
    },
  })

  const copyToClipboard = (text: string, label: string) => {
    navigator.clipboard.writeText(text)
    toast.success(`已复制 ${label} 到剪贴板`)
  }

  const handleCreate = () => {
    if (!sceneStr.trim()) {
      toast.error('请输入场景值标识')
      return
    }

    let days = expireDays
    if (days <= 0 || days > 30) {
      days = 30
    }

    createMutation.mutate({
      qrcode_type: qrcodeType,
      scene_str: sceneStr.trim(),
      expire_seconds: qrcodeType === 'temp' ? days * 86400 : 0,
      remark: remark.trim(),
    })
  }

  // 下载二维码图片
  const handleDownload = (item: any) => {
    const link = document.createElement('a')
    link.href = `/api/qrcode/${item.id}/image`
    link.download = `qrcode_${item.scene_str || item.id}.png`
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    toast.success('已开始下载二维码图片')
  }

  return (
    <div className="space-y-4 sm:space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-bold tracking-tight">带参二维码营销</h2>
          <p className="text-sm text-muted-foreground">
            生成渠道追踪专属二维码，支持临时（最长30天）与永久二维码，可精准统计扫码关注来源
          </p>
        </div>

        <Button onClick={() => setDialogOpen(true)} className="gap-2 bg-emerald-600 hover:bg-emerald-700 w-full sm:w-auto min-h-[38px] justify-center">
          <Plus className="h-4 w-4" />
          <span>生成带参二维码</span>
        </Button>
      </div>

      {isLoading ? (
        <div className="py-12 text-center text-muted-foreground">加载二维码列表中...</div>
      ) : qrcodes?.list && qrcodes.list.length > 0 ? (
        <div className="grid gap-3 sm:gap-4 md:gap-6 grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
          {qrcodes.list.map((item: any) => {
            const isTemp = item.qrcode_type === 'temp'
            const imgSrc = `/api/qrcode/${item.id}/image`

            return (
              <Card key={item.id} className="overflow-hidden group hover:shadow-md transition-shadow">
                <div className="p-6 bg-muted/40 flex flex-col items-center justify-center border-b">
                  <div className="h-44 w-44 bg-white p-2 rounded-xl shadow-sm border flex items-center justify-center overflow-hidden">
                    <img
                      src={imgSrc}
                      referrerPolicy="no-referrer"
                      alt="二维码"
                      className="h-full w-full object-contain"
                    />
                  </div>
                  <div className="mt-3 flex items-center gap-1.5">
                    <Badge variant={isTemp ? 'secondary' : 'default'} className="text-[10px] gap-1">
                      {isTemp ? <Clock className="h-3 w-3" /> : <InfinityIcon className="h-3 w-3" />}
                      <span>{isTemp ? '临时二维码' : '永久二维码'}</span>
                    </Badge>
                  </div>
                </div>

                <CardContent className="p-4 space-y-2 text-xs">
                  <div className="flex items-center justify-between">
                    <span className="text-muted-foreground">场景值 (Scene):</span>
                    <span className="font-semibold font-mono text-foreground">{item.scene_str || item.scene_id}</span>
                  </div>

                  {item.remark && (
                    <div className="flex items-center justify-between">
                      <span className="text-muted-foreground">用途备注:</span>
                      <span className="font-medium truncate max-w-[130px]">{item.remark}</span>
                    </div>
                  )}

                  {isTemp && (
                    <div className="flex items-center justify-between text-muted-foreground">
                      <span>到期时间:</span>
                      <span>{formatDate(item.expire_at)}</span>
                    </div>
                  )}

                  <div className="pt-2 flex items-center justify-between border-t border-border/60">
                    <span className="text-muted-foreground">{formatDate(item.created_at)}</span>
                    <div className="flex items-center gap-2">
                      <button
                        onClick={() => handleDownload(item)}
                        className="text-primary hover:underline flex items-center gap-0.5 font-medium"
                        title="下载二维码图片到本地"
                      >
                        <Download className="h-3.5 w-3.5" />
                        <span>下载</span>
                      </button>
                      <button
                        onClick={() => copyToClipboard(window.location.origin + imgSrc, '二维码图片链接')}
                        className="text-muted-foreground hover:text-foreground inline-flex items-center gap-0.5"
                        title="复制图片直链"
                      >
                        <Copy className="h-3.5 w-3.5" />
                      </button>
                    </div>
                  </div>
                </CardContent>
              </Card>
            )
          })}
        </div>
      ) : (
        <Card className="p-12 text-center text-muted-foreground">
          暂未生成任何带参二维码，点击右上角「生成带参二维码」
        </Card>
      )}

      {/* 创建二维码弹窗 */}
      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="sm:max-w-md max-h-[85vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>新建带参二维码</DialogTitle>
          </DialogHeader>

          <div className="space-y-4 py-2 text-sm">
            <div className="space-y-1.5">
              <label className="text-xs font-semibold">二维码类型 *</label>
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                <Button
                  type="button"
                  variant={qrcodeType === 'temp' ? 'default' : 'outline'}
                  onClick={() => setQrcodeType('temp')}
                  className="text-xs gap-1.5"
                >
                  <Clock className="h-3.5 w-3.5" />
                  <span>临时二维码 (有时效)</span>
                </Button>
                <Button
                  type="button"
                  variant={qrcodeType === 'limit' ? 'default' : 'outline'}
                  onClick={() => setQrcodeType('limit')}
                  className="text-xs gap-1.5"
                >
                  <InfinityIcon className="h-3.5 w-3.5" />
                  <span>永久二维码 (长效)</span>
                </Button>
              </div>
            </div>

            <div className="space-y-1.5">
              <label className="text-xs font-semibold">场景值标识 (Scene Str) *</label>
              <Input
                value={sceneStr}
                onChange={(e) => setSceneStr(e.target.value)}
                placeholder="例如: channel_weibo 或 campaign_2025"
              />
              <p className="text-[11px] text-muted-foreground">
                粉丝扫码关注后微信会把该场景值传给回调服务器用于跟踪转化
              </p>
            </div>

            {qrcodeType === 'temp' && (
              <div className="space-y-1.5">
                <label className="text-xs font-semibold">有效天数 (1 - 30天，微信官方最大支持30天)</label>
                <Input
                  type="number"
                  min={1}
                  max={30}
                  value={expireDays}
                  onChange={(e) => setExpireDays(Math.min(30, Math.max(1, parseInt(e.target.value) || 30)))}
                />
              </div>
            )}

            <div className="space-y-1.5">
              <label className="text-xs font-semibold">用途备注</label>
              <Input
                value={remark}
                onChange={(e) => setRemark(e.target.value)}
                placeholder="例如: 2025线上春季招募展位推广"
              />
            </div>

            <DialogFooter className="pt-2">
              <Button variant="outline" onClick={() => setDialogOpen(false)}>取消</Button>
              <Button onClick={handleCreate} disabled={createMutation.isPending}>
                {createMutation.isPending ? '生成中...' : '立即生成'}
              </Button>
            </DialogFooter>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  )
}
