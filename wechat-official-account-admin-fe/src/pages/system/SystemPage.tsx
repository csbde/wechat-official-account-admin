import React, { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
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
  Shield,
  UserCheck,
  Plus,
  Trash2,
  KeyRound,
  ExternalLink,
  Server,
  Globe,
} from 'lucide-react'
import { toast } from 'sonner'
import { formatDate } from '@/lib/utils'

export const SystemPage: React.FC = () => {
  const queryClient = useQueryClient()
  const [userModalOpen, setUserModalOpen] = useState(false)
  const [editingUserId, setEditingUserId] = useState<number | null>(null)
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [nickname, setNickname] = useState('')
  const [role, setRole] = useState('operator')

  // 获取用户列表
  const { data: users, isLoading } = useQuery<any[]>({
    queryKey: ['system-users'],
    queryFn: () => api.get('/api/system/users'),
  })

  // 获取出口公网 IP
  const { data: ipData } = useQuery<any>({
    queryKey: ['system-ip'],
    queryFn: () => api.get('/api/system/public-ip'),
  })

  // 创建/更新用户
  const saveUserMutation = useMutation({
    mutationFn: (values: any) => {
      if (editingUserId) {
        return api.put(`/api/system/users/${editingUserId}`, values)
      }
      return api.post('/api/system/users', values)
    },
    onSuccess: () => {
      toast.success(editingUserId ? '用户信息已更新' : '新用户创建成功')
      queryClient.invalidateQueries({ queryKey: ['system-users'] })
      setUserModalOpen(false)
    },
  })

  // 删除用户
  const deleteUserMutation = useMutation({
    mutationFn: (id: number) => api.delete(`/api/system/users/${id}`),
    onSuccess: () => {
      toast.success('用户已删除')
      queryClient.invalidateQueries({ queryKey: ['system-users'] })
    },
  })

  const openCreateModal = () => {
    setEditingUserId(null)
    setUsername('')
    setPassword('')
    setNickname('')
    setRole('operator')
    setUserModalOpen(true)
  }

  const openEditModal = (u: any) => {
    setEditingUserId(u.id)
    setUsername(u.username)
    setPassword('')
    setNickname(u.nickname || '')
    setRole(u.role || 'operator')
    setUserModalOpen(true)
  }

  const handleSave = () => {
    if (!editingUserId && (!username.trim() || !password.trim())) {
      toast.error('用户名和密码不能为空')
      return
    }
    saveUserMutation.mutate({
      username,
      password: password || undefined,
      nickname,
      role,
    })
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-bold tracking-tight">系统管理与 RBAC 权限</h2>
          <p className="text-sm text-muted-foreground">
            基于 Casbin 角色访问控制，管理操作员账号、服务器出口 IP 与 OpenAPI 交互文档
          </p>
        </div>

        <div className="flex items-center gap-2">
          <a
            href="/swagger/index.html"
            target="_blank"
            rel="noreferrer"
            className="inline-flex items-center justify-center rounded-lg text-xs font-medium border border-input bg-background shadow-sm hover:bg-accent h-9 px-3 gap-1.5"
          >
            <ExternalLink className="h-3.5 w-3.5 text-primary" />
            <span>OpenAPI / Swagger 接口文档</span>
          </a>
          <Button onClick={openCreateModal} className="gap-2 text-xs">
            <Plus className="h-4 w-4" />
            <span>添加管理员/操作员</span>
          </Button>
        </div>
      </div>

      {/* 服务器环境卡片 */}
      <div className="grid gap-4 md:grid-cols-3">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground flex items-center gap-2">
              <Globe className="h-4 w-4 text-emerald-600" />
              <span>服务器出口公网 IP</span>
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-xl font-bold font-mono">{ipData?.ip || '获取中...'}</div>
            <p className="text-xs text-muted-foreground mt-1">需加入微信公众平台后台 IP 白名单</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground flex items-center gap-2">
              <Shield className="h-4 w-4 text-primary" />
              <span>权限架构 (Casbin RBAC)</span>
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-xl font-bold">Admin / Operator</div>
            <p className="text-xs text-muted-foreground mt-1">PostgreSQL gorm-adapter 持久化策略</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground flex items-center gap-2">
              <Server className="h-4 w-4 text-purple-600" />
              <span>单点登录 (SSO) 支持</span>
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-xl font-bold">JWT + OAuth2 Ready</div>
            <p className="text-xs text-muted-foreground mt-1">支持与企业微信/飞书/OIDC 无缝接入</p>
          </CardContent>
        </Card>
      </div>

      {/* 用户列表 */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">后台用户列表</CardTitle>
        </CardHeader>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>用户名</TableHead>
                <TableHead>昵称</TableHead>
                <TableHead>角色 (Role)</TableHead>
                <TableHead>状态</TableHead>
                <TableHead>创建时间</TableHead>
                <TableHead className="text-right">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoading ? (
                <TableRow>
                  <TableCell colSpan={6} className="h-24 text-center text-muted-foreground">加载中...</TableCell>
                </TableRow>
              ) : users?.map((u) => (
                <TableRow key={u.id} className="text-xs">
                  <TableCell className="font-semibold font-mono">{u.username}</TableCell>
                  <TableCell>{u.nickname || '-'}</TableCell>
                  <TableCell>
                    <Badge variant={u.role === 'admin' ? 'default' : 'secondary'} className="text-[10px] capitalize">
                      {u.role}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <Badge variant={u.status === 1 ? 'success' : 'destructive'} className="text-[10px]">
                      {u.status === 1 ? '正常' : '禁用'}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-muted-foreground">{formatDate(u.created_at)}</TableCell>
                  <TableCell className="text-right space-x-1">
                    <Button variant="ghost" size="icon" onClick={() => openEditModal(u)} className="h-7 w-7">
                      <KeyRound className="h-3.5 w-3.5" />
                    </Button>
                    {u.id !== 1 && (
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => {
                          if (confirm(`确认删除用户 ${u.username}？`)) {
                            deleteUserMutation.mutate(u.id)
                          }
                        }}
                        className="h-7 w-7 text-destructive hover:bg-destructive/10"
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                      </Button>
                    )}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* 用户编辑/创建弹窗 */}
      <Dialog open={userModalOpen} onOpenChange={setUserModalOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>{editingUserId ? '修改用户信息与密码' : '创建新后台用户'}</DialogTitle>
          </DialogHeader>

          <div className="space-y-4 py-2 text-sm">
            <div className="space-y-1.5">
              <label className="text-xs font-semibold">登录用户名 *</label>
              <Input
                value={username}
                disabled={!!editingUserId}
                onChange={(e) => setUsername(e.target.value)}
                placeholder="例如: operator1"
              />
            </div>

            <div className="space-y-1.5">
              <label className="text-xs font-semibold">用户昵称</label>
              <Input
                value={nickname}
                onChange={(e) => setNickname(e.target.value)}
                placeholder="例如: 新媒体运营组"
              />
            </div>

            <div className="space-y-1.5">
              <label className="text-xs font-semibold">
                {editingUserId ? '重置密码 (留空则不修改)' : '登录密码 *'}
              </label>
              <Input
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="输入密码"
              />
            </div>

            <div className="space-y-1.5">
              <label className="text-xs font-semibold">系统角色 (Role)</label>
              <select
                value={role}
                onChange={(e) => setRole(e.target.value)}
                className="w-full h-9 rounded-lg border border-input bg-background px-3 text-sm"
              >
                <option value="operator">Operator (运营操作员)</option>
                <option value="admin">Admin (超级管理员)</option>
              </select>
            </div>

            <DialogFooter className="pt-2">
              <Button variant="outline" onClick={() => setUserModalOpen(false)}>取消</Button>
              <Button onClick={handleSave} disabled={saveUserMutation.isPending}>
                {saveUserMutation.isPending ? '保存中...' : '确定保存'}
              </Button>
            </DialogFooter>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  )
}
