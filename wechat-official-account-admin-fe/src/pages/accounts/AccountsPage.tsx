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
  Smartphone,
  Copy,
  Trash2,
  Edit,
  Check,
  Star,
  ExternalLink,
} from 'lucide-react'
import { toast } from 'sonner'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import * as z from 'zod'

const accountSchema = z.object({
  name: z.string().min(2, '公众号名称至少2个字符'),
  appid: z.string().min(6, 'AppID 不能为空'),
  appsecret: z.string().min(6, 'AppSecret 不能为空'),
  token: z.string().optional(),
  encoding_aes_key: z.string().optional(),
  description: z.string().optional(),
})

type AccountFormData = z.infer<typeof accountSchema>

export const AccountsPage: React.FC = () => {
  const queryClient = useQueryClient()
  const { currentApp, setCurrentApp } = useAppStore()
  const { user } = useAuthStore()
  const isAdmin = user?.role === 'admin'

  const [dialogOpen, setDialogOpen] = useState(false)
  const [editingId, setEditingId] = useState<number | null>(null)

  const { data: accounts, isLoading } = useQuery<WxAccount[]>({
    queryKey: ['accounts'],
    queryFn: () => api.get('/api/accounts'),
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
    },
  })

  // 创建/更新 Mutation
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

  // 删除 Mutation
  const deleteMutation = useMutation({
    mutationFn: (id: number) => api.delete(`/api/accounts/${id}`),
    onSuccess: (_, id) => {
      toast.success('公众号已删除')
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

  const handleOpenEdit = (acc: WxAccount) => {
    setEditingId(acc.id)
    form.reset({
      name: acc.name,
      appid: acc.appid,
      appsecret: acc.appsecret,
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
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold tracking-tight">微信公众号管理</h2>
          <p className="text-sm text-muted-foreground">
            配置与管理微信服务号/订阅号，支持多账号无缝切换与参数配置
          </p>
        </div>
        <Button onClick={handleOpenCreate} className="gap-2">
          <Plus className="h-4 w-4" />
          <span>添加公众号</span>
        </Button>
      </div>

      {isLoading ? (
        <div className="py-12 text-center text-muted-foreground">正在加载公众号数据...</div>
      ) : accounts && accounts.length > 0 ? (
        <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
          {accounts.map((acc) => {
            const isSelected = currentApp?.id === acc.id
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
                      <div className="h-10 w-10 rounded-xl bg-primary/10 text-primary flex items-center justify-center font-bold text-base">
                        <Smartphone className="h-5 w-5" />
                      </div>
                      <div>
                        <CardTitle className="text-base flex items-center gap-1.5">
                          <span>{acc.name}</span>
                          {acc.is_default && (
                            <Badge variant="success" className="text-[10px]">
                              默认
                            </Badge>
                          )}
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

                  {/* 微信接入服务器信息 */}
                  <div className="p-2.5 rounded-lg bg-muted/60 space-y-1.5 font-mono text-[11px]">
                    <div className="flex items-center justify-between text-muted-foreground">
                      <span>服务器 URL (微信后台填写):</span>
                      <button
                        onClick={() => copyToClipboard(acc.callback_url || '', 'URL')}
                        className="hover:text-foreground text-primary inline-flex items-center gap-1"
                      >
                        <Copy className="h-3 w-3" />
                        <span>复制</span>
                      </button>
                    </div>
                    <div className="truncate text-foreground select-all bg-background/50 px-2 py-1 rounded">
                      {acc.callback_url || `/wxmp/${acc.appid}/handler`}
                    </div>

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
                  </div>

                  {/* 底部操作条 */}
                  <div className="pt-2 flex items-center justify-between border-t border-border/50">
                    <div className="flex items-center gap-1">
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

                    <div className="flex items-center gap-1">
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
              点击下方按钮输入微信公众平台 (mp.weixin.qq.com) 提供的 AppID 和 AppSecret 即可开始管理。
            </p>
          </div>
          <Button onClick={handleOpenCreate} className="gap-2">
            <Plus className="h-4 w-4" />
            <span>立即添加公众号</span>
          </Button>
        </Card>
      )}

      {/* 新增/编辑 Modal */}
      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>{editingId ? '编辑微信公众号配置' : '添加微信公众号'}</DialogTitle>
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
              <label className="text-xs font-semibold">AppSecret (开发者密码) *</label>
              <Input
                type="password"
                placeholder={editingId ? '若不修改请保持原样' : '微信公众平台生成的 AppSecret'}
                {...form.register('appsecret')}
              />
              {form.formState.errors.appsecret && (
                <span className="text-[11px] text-destructive">{form.formState.errors.appsecret.message}</span>
              )}
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1">
                <label className="text-xs font-semibold">令牌 (Token)</label>
                <Input placeholder="自定义字符串" {...form.register('token')} />
              </div>
              <div className="space-y-1">
                <label className="text-xs font-semibold">消息加解密密钥 (AESKey)</label>
                <Input placeholder="43位字符(可选)" {...form.register('encoding_aes_key')} />
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
    </div>
  )
}
