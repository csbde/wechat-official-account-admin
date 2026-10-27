import React, { useEffect, useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'
import { useAuthStore } from '@/stores/auth'
import { Button } from '@/components/ui/button'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Switch } from '@/components/ui/switch'
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
  Building2,
  Plus,
  Pencil,
  Trash2,
  Users,
  AppWindow,
  Eye,
  ShieldCheck,
  RefreshCw,
} from 'lucide-react'
import { toast } from 'sonner'
import { formatDate } from '@/lib/utils'

interface TenantRow {
  id: number
  name: string
  code: string
  description: string
  status: number
  account_count: number
  user_count: number
  is_current: boolean
  is_home: boolean
  created_at: string
}

const emptyForm = {
  name: '',
  code: '',
  description: '',
  status: 1,
  withAdmin: false,
  adminUsername: '',
  adminPassword: '',
  adminEmail: '',
}

/**
 * 租户(组织)管理面板
 * - 超级管理员：可新建 / 编辑 / 删除租户，并可切换租户视角代运营
 * - 普通管理员：只展示自己所属租户，所有写操作入口隐藏且后端二次拦截
 */
export const TenantPanel: React.FC = () => {
  const queryClient = useQueryClient()
  const { user, activeTenantId, setActiveTenantId } = useAuthStore()
  const isSuperAdmin = user?.role === 'superadmin'

  const [dialogOpen, setDialogOpen] = useState(false)
  const [editingId, setEditingId] = useState<number | null>(null)
  const [form, setForm] = useState({ ...emptyForm })

  const { data: tenants, isLoading, isError, error, refetch } = useQuery<TenantRow[]>({
    queryKey: ['system-tenants'],
    queryFn: () => api.get('/api/system/tenants'),
    retry: 0,
  })

  const setField = (patch: Partial<typeof emptyForm>) => setForm((prev) => ({ ...prev, ...patch }))

  const openCreate = () => {
    setEditingId(null)
    setForm({ ...emptyForm })
    setDialogOpen(true)
  }

  const openEdit = (t: TenantRow) => {
    setEditingId(t.id)
    setForm({
      ...emptyForm,
      name: t.name,
      code: t.code,
      description: t.description || '',
      status: t.status,
    })
    setDialogOpen(true)
  }

  const saveMutation = useMutation({
    mutationFn: (values: any) => {
      if (editingId) {
        return api.put(`/api/system/tenants/${editingId}`, {
          name: values.name,
          description: values.description,
          status: values.status,
        })
      }
      return api.post('/api/system/tenants', values)
    },
    onSuccess: () => {
      toast.success(editingId ? '租户信息已更新' : '租户创建成功')
      queryClient.invalidateQueries({ queryKey: ['system-tenants'] })
      queryClient.invalidateQueries({ queryKey: ['system-tenant-options'] })
      queryClient.invalidateQueries({ queryKey: ['accounts'] })
      setDialogOpen(false)
    },
  })

  const deleteMutation = useMutation({
    mutationFn: (id: number) => api.delete(`/api/system/tenants/${id}`),
    onSuccess: () => {
      toast.success('租户已删除')
      queryClient.invalidateQueries({ queryKey: ['system-tenants'] })
      queryClient.invalidateQueries({ queryKey: ['system-tenant-options'] })
    },
  })

  const handleDelete = (t: TenantRow) => {
    if (t.account_count > 0 || t.user_count > 0) {
      toast.error(
        `租户「${t.name}」下仍有 ${t.account_count} 个微信账号 / ${t.user_count} 个用户，请先迁移或删除，避免数据被孤立`,
      )
      return
    }
    if (!window.confirm(`确认删除租户「${t.name} (${t.code})」？删除后该标识可被重新启用。`)) return
    deleteMutation.mutate(t.id)
  }

  const handleSubmit = () => {
    if (!form.name.trim()) {
      toast.error('租户名称不能为空')
      return
    }
    if (!editingId) {
      if (!/^[a-zA-Z][a-zA-Z0-9_-]{1,63}$/.test(form.code.trim())) {
        toast.error('租户标识需字母开头，仅允许字母、数字、下划线与中划线，长度 2-64')
        return
      }
      if (form.withAdmin) {
        if (!form.adminUsername.trim()) {
          toast.error('请填写该租户初始管理员的用户名')
          return
        }
        if (form.adminPassword.length < 6) {
          toast.error('初始管理员密码不能少于 6 位')
          return
        }
      }
    }
    saveMutation.mutate({ ...form, code: form.code.trim(), name: form.name.trim() })
  }

  // 切换到该租户视角（仅超级管理员）
  const switchView = (t: TenantRow) => {
    setActiveTenantId(t.id === (user?.tenant_id ?? 1) ? null : t.id)
    queryClient.invalidateQueries()
    toast.success(t.id === (user?.tenant_id ?? 1) ? '已回到本租户视角' : `已切换到租户「${t.name}」视角`)
  }

  useEffect(() => {
    if (!dialogOpen) return
    // 弹窗打开后强制滚动到顶部，避免内容较长时视口错位
    const el = document.querySelector('[data-tenant-dialog-body]')
    if (el) el.scrollTop = 0
  }, [dialogOpen, editingId])

  return (
    <Card>
      <CardHeader>
        <div className="flex items-start justify-between gap-3">
          <div>
            <CardTitle className="text-base flex items-center gap-2">
              <Building2 className="h-4 w-4 text-primary" />
              <span>租户与组织 (多租户隔离)</span>
              {!isSuperAdmin && (
                <Badge variant="secondary" className="text-[10px]">
                  只读
                </Badge>
              )}
            </CardTitle>
            <CardDescription className="text-xs mt-1">
              {isSuperAdmin
                ? '每个租户的数据(公众号/小程序/菜单/素材/粉丝/日志)完全物理隔离；切换“视角”后可直接为该租户录入账号与配置'
                : '你只能看到自己所属租户；如需新增组织，请联系超级管理员在「系统与权限」中创建'}
            </CardDescription>
          </div>
          {isSuperAdmin && (
            <div className="flex shrink-0 gap-2">
              <Button variant="outline" size="sm" onClick={() => refetch()} className="gap-1.5 text-xs">
                <RefreshCw className="h-3.5 w-3.5" />
                <span>刷新</span>
              </Button>
              <Button size="sm" onClick={openCreate} className="gap-1.5 text-xs">
                <Plus className="h-3.5 w-3.5" />
                <span>新建租户</span>
              </Button>
            </div>
          )}
        </div>
      </CardHeader>
      <CardContent className="p-0">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-14">ID</TableHead>
              <TableHead>组织名称</TableHead>
              <TableHead>唯一标识 (Code)</TableHead>
              <TableHead className="text-center">
                <span className="inline-flex items-center gap-1">
                  <AppWindow className="h-3.5 w-3.5" />
                  <span>微信账号</span>
                </span>
              </TableHead>
              <TableHead className="text-center">
                <span className="inline-flex items-center gap-1">
                  <Users className="h-3.5 w-3.5" />
                  <span>后台用户</span>
                </span>
              </TableHead>
              <TableHead>状态</TableHead>
              <TableHead>创建时间</TableHead>
              <TableHead className="text-right">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <TableRow>
                <TableCell colSpan={8} className="h-20 text-center text-muted-foreground">
                  加载中...
                </TableCell>
              </TableRow>
            ) : isError ? (
              <TableRow>
                <TableCell colSpan={8} className="h-20 text-center text-destructive">
                  加载失败：{(error as Error)?.message || '无权限访问租户列表'}
                </TableCell>
              </TableRow>
            ) : !tenants || tenants.length === 0 ? (
              <TableRow>
                <TableCell colSpan={8} className="h-20 text-center text-muted-foreground">
                  暂无租户数据
                </TableCell>
              </TableRow>
            ) : (
              tenants.map((t) => (
                <TableRow key={t.id} className="text-xs">
                  <TableCell className="font-mono text-muted-foreground">{t.id}</TableCell>
                  <TableCell>
                    <div className="flex flex-wrap items-center gap-1.5">
                      <span className="font-semibold text-foreground">{t.name}</span>
                      {t.is_home && (
                        <Badge variant="outline" className="text-[10px]">
                          本租户
                        </Badge>
                      )}
                      {t.is_current && !t.is_home && (
                        <Badge variant="success" className="text-[10px]">
                          当前视角
                        </Badge>
                      )}
                    </div>
                    {t.description && (
                      <div className="text-[11px] text-muted-foreground mt-0.5 max-w-[280px] truncate">
                        {t.description}
                      </div>
                    )}
                  </TableCell>
                  <TableCell className="font-mono text-muted-foreground">{t.code}</TableCell>
                  <TableCell className="text-center font-mono">{t.account_count}</TableCell>
                  <TableCell className="text-center font-mono">{t.user_count}</TableCell>
                  <TableCell>
                    <Badge variant={t.status === 1 ? 'success' : 'destructive'} className="text-[10px]">
                      {t.status === 1 ? '启用' : '停用'}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-muted-foreground">{formatDate(t.created_at)}</TableCell>
                  <TableCell className="text-right">
                    <div className="inline-flex items-center justify-end gap-1">
                      {isSuperAdmin && !t.is_current && (
                        <Button
                          variant="ghost"
                          size="icon"
                          className="h-7 w-7"
                          title={t.status === 1 ? '切换到该租户视角 (代运营)' : '该租户已停用，无法切换'}
                          disabled={t.status !== 1}
                          onClick={() => switchView(t)}
                        >
                          <Eye className="h-3.5 w-3.5" />
                        </Button>
                      )}
                      {isSuperAdmin && t.is_current && (
                        <Button
                          variant="ghost"
                          size="icon"
                          className="h-7 w-7 text-primary"
                          title="返回本租户视角"
                          onClick={() => switchView(t)}
                        >
                          <ShieldCheck className="h-3.5 w-3.5" />
                        </Button>
                      )}
                      {isSuperAdmin && (
                        <>
                          <Button
                            variant="ghost"
                            size="icon"
                            className="h-7 w-7"
                            title="编辑租户"
                            onClick={() => openEdit(t)}
                          >
                            <Pencil className="h-3.5 w-3.5" />
                          </Button>
                          <Button
                            variant="ghost"
                            size="icon"
                            className="h-7 w-7 text-destructive hover:bg-destructive/10"
                            title="删除租户 (需先清空账号与用户)"
                            disabled={t.is_home}
                            onClick={() => handleDelete(t)}
                          >
                            <Trash2 className="h-3.5 w-3.5" />
                          </Button>
                        </>
                      )}
                    </div>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>

        {activeTenantId && (
          <div className="mx-6 mb-4 mt-1 p-2.5 rounded-lg bg-amber-500/10 border border-amber-500/25 text-[11px] text-amber-700 dark:text-amber-300">
            你正处于「代管租户」视角，所有新增的公众号、菜单、素材等数据都会归属该租户；操作完成后请及时切回本租户。
          </div>
        )}
      </CardContent>

      {/* 租户新建 / 编辑弹窗 */}
      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="sm:max-w-lg max-h-[85vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <Building2 className="h-5 w-5 text-primary" />
              <span>{editingId ? '编辑租户组织' : '新建租户组织'}</span>
            </DialogTitle>
          </DialogHeader>

          <div data-tenant-dialog-body className="space-y-4 py-2 text-sm">
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1.5">
                <label className="text-xs font-semibold">组织名称 *</label>
                <Input
                  value={form.name}
                  onChange={(e) => setField({ name: e.target.value })}
                  placeholder="例如: 华东分公司"
                />
              </div>
              <div className="space-y-1.5">
                <label className="text-xs font-semibold">唯一标识 (Code) *</label>
                <Input
                  value={form.code}
                  disabled={!!editingId}
                  onChange={(e) => setField({ code: e.target.value })}
                  placeholder="例如: east-branch"
                  className="font-mono"
                />
                <p className="text-[11px] text-muted-foreground">
                  {editingId ? '唯一标识创建后不可修改' : '字母开头，允许字母/数字/下划线/中划线'}
                </p>
              </div>
            </div>

            <div className="space-y-1.5">
              <label className="text-xs font-semibold">备注说明</label>
              <Input
                value={form.description}
                onChange={(e) => setField({ description: e.target.value })}
                placeholder="例如: 负责华东区公众号代运营"
              />
            </div>

            <div className="flex items-center justify-between p-3 rounded-xl bg-muted/60 border">
              <div>
                <div className="font-semibold text-sm">租户启用状态</div>
                <div className="text-[11px] text-muted-foreground mt-0.5">
                  停用后该租户成员将无法登录与切换视角
                </div>
              </div>
              <Switch
                checked={form.status === 1}
                onCheckedChange={(v) => setField({ status: v ? 1 : 0 })}
              />
            </div>

            {!editingId && (
              <div className="space-y-3 rounded-xl border bg-muted/30 p-3">
                <div className="flex items-center justify-between">
                  <div>
                    <div className="font-semibold text-sm">同时创建该租户的初始管理员</div>
                    <div className="text-[11px] text-muted-foreground mt-0.5">
                      建议开启，否则新租户无人可登录 (角色固定为 admin，不能自建超级管理员)
                    </div>
                  </div>
                  <Switch
                    checked={form.withAdmin}
                    onCheckedChange={(v) => setField({ withAdmin: v })}
                  />
                </div>

                {form.withAdmin && (
                  <div className="grid grid-cols-2 gap-3 pt-1 border-t border-border/50">
                    <div className="space-y-1.5">
                      <label className="text-xs font-semibold">管理员用户名 *</label>
                      <Input
                        value={form.adminUsername}
                        onChange={(e) => setField({ adminUsername: e.target.value })}
                        placeholder="例如: east_admin"
                      />
                    </div>
                    <div className="space-y-1.5">
                      <label className="text-xs font-semibold">登录密码 *</label>
                      <Input
                        type="password"
                        value={form.adminPassword}
                        onChange={(e) => setField({ adminPassword: e.target.value })}
                        placeholder="不少于 6 位"
                      />
                    </div>
                    <div className="space-y-1.5 col-span-2">
                      <label className="text-xs font-semibold">绑定邮箱 (用于 OIDC 单点登录匹配)</label>
                      <Input
                        type="email"
                        value={form.adminEmail}
                        onChange={(e) => setField({ adminEmail: e.target.value })}
                        placeholder="例如: admin@company.com"
                      />
                    </div>
                  </div>
                )}
              </div>
            )}

            <DialogFooter className="pt-1">
              <Button variant="outline" onClick={() => setDialogOpen(false)}>
                取消
              </Button>
              <Button onClick={handleSubmit} disabled={saveMutation.isPending}>
                {saveMutation.isPending ? '保存中...' : '确定保存'}
              </Button>
            </DialogFooter>
          </div>
        </DialogContent>
      </Dialog>
    </Card>
  )
}
