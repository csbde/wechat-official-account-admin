import React, { useState, useEffect } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'
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
  Shield,
  Plus,
  Trash2,
  KeyRound,
  ExternalLink,
  Globe,
  Mail,
  Fingerprint,
  Settings,
  Copy,
  CheckCircle2,
  RefreshCw,
  Send,
} from 'lucide-react'
import { toast } from 'sonner'
import { formatDate } from '@/lib/utils'

export const SystemPage: React.FC = () => {
  const queryClient = useQueryClient()

  // 用户管理状态
  const [userModalOpen, setUserModalOpen] = useState(false)
  const [editingUserId, setEditingUserId] = useState<number | null>(null)
  const [username, setUsername] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [nickname, setNickname] = useState('')
  const [role, setRole] = useState('operator')

  // OIDC 配置弹窗状态
  const [oidcModalOpen, setOidcModalOpen] = useState(false)
  const [oidcEnabled, setOidcEnabled] = useState(false)
  const [oidcIssuer, setOidcIssuer] = useState('')
  const [oidcClientId, setOidcClientId] = useState('')
  const [oidcClientSecret, setOidcClientSecret] = useState('')
  const [oidcRedirectUri, setOidcRedirectUri] = useState('')
  const [oidcScopes, setOidcScopes] = useState('openid profile email')
  const [oidcAutoRegister, setOidcAutoRegister] = useState(false)
  const [oidcDefaultRole, setOidcDefaultRole] = useState('operator')
  const [testResult, setTestResult] = useState<any>(null)
  const [testingOIDC, setTestingOIDC] = useState(false)

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

  // 获取 OIDC 配置
  const { data: oidcConfig } = useQuery<any>({
    queryKey: ['system-oidc-setting'],
    queryFn: () => api.get('/api/system/oidc'),
  })

  useEffect(() => {
    if (oidcConfig) {
      setOidcEnabled(!!oidcConfig.enabled)
      setOidcIssuer(oidcConfig.issuer || '')
      setOidcClientId(oidcConfig.client_id || '')
      setOidcClientSecret(oidcConfig.client_secret || '')
      setOidcRedirectUri(oidcConfig.redirect_uri || oidcConfig.suggested_redirect || '')
      setOidcScopes(oidcConfig.scopes || 'openid profile email')
      setOidcAutoRegister(!!oidcConfig.auto_register)
      setOidcDefaultRole(oidcConfig.default_role || 'operator')
    }
  }, [oidcConfig])

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

  // 保存 OIDC 配置
  const saveOidcMutation = useMutation({
    mutationFn: (values: any) => api.post('/api/system/oidc', values),
    onSuccess: () => {
      toast.success('OIDC 单点登录配置保存成功！')
      queryClient.invalidateQueries({ queryKey: ['system-oidc-setting'] })
      setOidcModalOpen(false)
    },
  })

  // 测试 OIDC 连通性
  const handleTestOIDC = async () => {
    if (!oidcIssuer.trim()) {
      toast.error('请先输入 OIDC Issuer 基础地址')
      return
    }
    try {
      setTestingOIDC(true)
      const res: any = await api.post('/api/system/oidc/test', { issuer: oidcIssuer })
      setTestResult(res)
      toast.success('OIDC 端点连通测试成功！')
    } catch (e: any) {
      toast.error(e.message || '连通失败，请检查 Issuer 地址')
    } finally {
      setTestingOIDC(false)
    }
  }

  // 状态启停
  const toggleStatusMutation = useMutation({
    mutationFn: ({ id, status }: { id: number; status: number }) =>
      api.put(`/api/system/users/${id}`, { status }),
    onSuccess: () => {
      toast.success('用户状态已变更')
      queryClient.invalidateQueries({ queryKey: ['system-users'] })
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
    setEmail('')
    setPassword('')
    setNickname('')
    setRole('operator')
    setUserModalOpen(true)
  }

  const openEditModal = (u: any) => {
    setEditingUserId(u.id)
    setUsername(u.username)
    setEmail(u.email || '')
    setPassword('')
    setNickname(u.nickname || '')
    setRole(u.role || 'operator')
    setUserModalOpen(true)
  }

  const copyText = (txt: string, label: string) => {
    navigator.clipboard.writeText(txt)
    toast.success(`已复制 ${label} 到剪贴板`)
  }

  const handleSave = () => {
    if (!editingUserId && (!username.trim() || !password.trim())) {
      toast.error('用户名和密码不能为空')
      return
    }
    saveUserMutation.mutate({
      username,
      email: email.trim() || undefined,
      password: password || undefined,
      nickname,
      role,
    })
  }

  const effectiveRedirect =
    oidcRedirectUri ||
    oidcConfig?.suggested_redirect ||
    `${typeof window !== 'undefined' ? window.location.origin : ''}/api/auth/oidc/callback`

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-bold tracking-tight">系统管理与 RBAC 权限</h2>
          <p className="text-sm text-muted-foreground">
            用户账号维护、OIDC 企业单点登录配置、邮箱绑定管理与 Swagger 接口调试
          </p>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          {/* 配置 OIDC 按钮 */}
          <Button
            variant="outline"
            onClick={() => {
              setTestResult(null)
              setOidcModalOpen(true)
            }}
            className="gap-1.5 text-xs border-emerald-500/40 text-emerald-600 hover:bg-emerald-500/10 font-medium"
          >
            <Fingerprint className="h-4 w-4 text-emerald-600" />
            <span>配置 OIDC 单点登录</span>
          </Button>

          <a
            href="/swagger/index.html"
            target="_blank"
            rel="noreferrer"
            className="inline-flex items-center justify-center rounded-lg text-xs font-medium border border-input bg-background shadow-sm hover:bg-accent h-9 px-3 gap-1.5"
          >
            <ExternalLink className="h-3.5 w-3.5 text-primary" />
            <span>OpenAPI 接口文档</span>
          </a>

          <Button onClick={openCreateModal} className="gap-2 text-xs">
            <Plus className="h-4 w-4" />
            <span>添加管理员/操作员</span>
          </Button>
        </div>
      </div>

      {/* 核心卡片 */}
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
            <p className="text-xs text-muted-foreground mt-1">请配置进微信公众平台后台 IP 白名单</p>
          </CardContent>
        </Card>

        {/* 可点击交互的 OIDC 状态卡片 */}
        <Card
          onClick={() => {
            setTestResult(null)
            setOidcModalOpen(true)
          }}
          className="cursor-pointer hover:border-emerald-500/60 transition-colors group"
        >
          <CardHeader className="pb-2">
            <div className="flex items-center justify-between">
              <CardTitle className="text-sm font-medium text-muted-foreground flex items-center gap-2">
                <Fingerprint className="h-4 w-4 text-primary" />
                <span>OIDC 邮箱自动绑定</span>
              </CardTitle>
              <span className="text-[11px] text-primary group-hover:underline flex items-center gap-0.5">
                <Settings className="h-3 w-3" />
                <span>配置</span>
              </span>
            </div>
          </CardHeader>
          <CardContent>
            <div className="text-xl font-bold flex items-center gap-2">
              <span>{oidcConfig?.enabled ? '已启用' : '未开启'}</span>
              <Badge variant={oidcConfig?.enabled ? 'success' : 'secondary'} className="text-[10px]">
                {oidcConfig?.enabled ? 'ACTIVE' : 'OFF'}
              </Badge>
            </div>
            <p className="text-xs text-muted-foreground mt-1 truncate">
              {oidcConfig?.issuer ? oidcConfig.issuer : '点击在此配置 Keycloak/Authing/Okta 服务'}
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground flex items-center gap-2">
              <Shield className="h-4 w-4 text-purple-600" />
              <span>Casbin RBAC 细粒度授权</span>
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-xl font-bold">多租户行级隔离</div>
            <p className="text-xs text-muted-foreground mt-1">跨租户请求由中间件物理级严格阻断</p>
          </CardContent>
        </Card>
      </div>

      {/* 用户列表 */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">后台用户列表</CardTitle>
          <CardDescription className="text-xs">
            为用户设置「绑定邮箱」后，该用户通过 OIDC 企业单点登录时将自动识别并绑定登录
          </CardDescription>
        </CardHeader>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>用户名</TableHead>
                <TableHead>绑定邮箱 (用于 OIDC 匹配)</TableHead>
                <TableHead>昵称</TableHead>
                <TableHead>角色 (Role)</TableHead>
                <TableHead>账号状态</TableHead>
                <TableHead>创建时间</TableHead>
                <TableHead className="text-right">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoading ? (
                <TableRow>
                  <TableCell colSpan={7} className="h-24 text-center text-muted-foreground">加载中...</TableCell>
                </TableRow>
              ) : users?.map((u) => (
                <TableRow key={u.id} className="text-xs">
                  <TableCell className="font-semibold font-mono">{u.username}</TableCell>
                  <TableCell className="font-mono text-muted-foreground">
                    {u.email ? (
                      <span className="text-foreground flex items-center gap-1 font-medium">
                        <Mail className="h-3.5 w-3.5 text-emerald-600" />
                        <span>{u.email}</span>
                      </span>
                    ) : (
                      <span className="text-muted-foreground/50">未绑定邮箱</span>
                    )}
                  </TableCell>
                  <TableCell>{u.nickname || '-'}</TableCell>
                  <TableCell>
                    <Badge variant={u.role === 'admin' || u.role === 'superadmin' ? 'default' : 'secondary'} className="text-[10px] capitalize">
                      {u.role}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-2">
                      <Switch
                        checked={u.status === 1}
                        disabled={u.id === 1}
                        onCheckedChange={(checked) =>
                          toggleStatusMutation.mutate({ id: u.id, status: checked ? 1 : 0 })
                        }
                      />
                      <span className="text-[11px] text-muted-foreground">{u.status === 1 ? '正常' : '禁用'}</span>
                    </div>
                  </TableCell>
                  <TableCell className="text-muted-foreground">{formatDate(u.created_at)}</TableCell>
                  <TableCell className="text-right space-x-1">
                    <Button variant="ghost" size="icon" onClick={() => openEditModal(u)} className="h-7 w-7" title="修改信息/重置密码">
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
                        title="删除用户"
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

      {/* OIDC 企业单点登录配置 Modal (重点功能！) */}
      <Dialog open={oidcModalOpen} onOpenChange={setOidcModalOpen}>
        <DialogContent className="sm:max-w-xl max-h-[85vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <Fingerprint className="h-5 w-5 text-emerald-600" />
              <span>OIDC 企业单点登录配置</span>
            </DialogTitle>
          </DialogHeader>

          <div className="space-y-4 py-2 text-xs">
            {/* 顶栏开关 */}
            <div className="flex items-center justify-between p-3 rounded-xl bg-muted/60 border">
              <div>
                <div className="font-semibold text-foreground text-sm">启用 OIDC 单点登录</div>
                <div className="text-muted-foreground text-[11px] mt-0.5">
                  开启后登录页将展示「OIDC 企业单点登录」快捷通道，登录后自动按用户邮箱匹配绑定
                </div>
              </div>
              <Switch checked={oidcEnabled} onCheckedChange={setOidcEnabled} />
            </div>

            {/* 回调 URL 提示 */}
            <div className="p-3 bg-muted/60 rounded-xl space-y-1.5 border font-mono">
              <div className="flex items-center justify-between font-sans font-semibold text-foreground">
                <span>OIDC 服务商后台需配置的回调地址 (redirect_uri):</span>
                <button
                  type="button"
                  onClick={() => copyText(effectiveRedirect, 'OIDC 回调地址')}
                  className="text-primary hover:underline text-[11px] inline-flex items-center gap-1 font-semibold"
                >
                  <Copy className="h-3 w-3" />
                  <span>复制地址</span>
                </button>
              </div>
              <div className="bg-background p-2 rounded text-[11px] select-all break-all border text-foreground">
                {effectiveRedirect}
              </div>
            </div>

            <div className="space-y-1.5">
              <label className="font-semibold text-foreground">OIDC Issuer 基础地址 *</label>
              <div className="flex gap-2">
                <Input
                  value={oidcIssuer}
                  onChange={(e) => setOidcIssuer(e.target.value)}
                  placeholder="例如: https://auth.company.com/realms/master"
                  className="font-mono text-xs flex-1"
                />
                <Button
                  type="button"
                  variant="outline"
                  onClick={handleTestOIDC}
                  disabled={testingOIDC}
                  className="gap-1 text-xs shrink-0"
                >
                  <RefreshCw className={`h-3 w-3 ${testingOIDC ? 'animate-spin' : ''}`} />
                  <span>{testingOIDC ? '测试中...' : '测试连通性'}</span>
                </Button>
              </div>
              <p className="text-[11px] text-muted-foreground">
                系统将自动通过 <code>{oidcIssuer || 'https://auth.company.com'}/.well-known/openid-configuration</code> 发现认证端点
              </p>
            </div>

            {/* 测试结果 */}
            {testResult && (
              <div className="p-2.5 bg-emerald-500/10 border border-emerald-500/20 rounded-lg space-y-1 font-mono text-[11px]">
                <div className="font-sans font-semibold text-emerald-800 dark:text-emerald-300 flex items-center gap-1">
                  <CheckCircle2 className="h-3.5 w-3.5" />
                  <span>{testResult.message}</span>
                </div>
                <div className="text-muted-foreground truncate">Auth: {testResult.authorization_endpoint}</div>
                <div className="text-muted-foreground truncate">Token: {testResult.token_endpoint}</div>
              </div>
            )}

            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1.5">
                <label className="font-semibold text-foreground">客户端 ID (Client ID) *</label>
                <Input
                  value={oidcClientId}
                  onChange={(e) => setOidcClientId(e.target.value)}
                  placeholder="woaa_client"
                  className="font-mono text-xs"
                />
              </div>

              <div className="space-y-1.5">
                <label className="font-semibold text-foreground">客户端密钥 (Client Secret) *</label>
                <Input
                  type="password"
                  value={oidcClientSecret}
                  onChange={(e) => setOidcClientSecret(e.target.value)}
                  placeholder="留空则不修改已有密钥"
                  className="font-mono text-xs"
                />
              </div>
            </div>

            <div className="space-y-1.5">
              <label className="font-semibold text-foreground">授权 Scopes</label>
              <Input
                value={oidcScopes}
                onChange={(e) => setOidcScopes(e.target.value)}
                placeholder="openid profile email"
                className="font-mono text-xs"
              />
            </div>

            {/* 自动创建账号安全策略 (重点配置) */}
            <div className="p-3 rounded-xl border bg-muted/40 space-y-3">
              <div className="flex items-center justify-between">
                <div>
                  <div className="font-semibold text-foreground">允许未录入邮箱自动创建账号</div>
                  <div className="text-muted-foreground text-[11px] mt-0.5">
                    若关闭 (推荐)，未在用户列表中录入该邮箱的外部人员将无法登录；若开启，将自动注册新账号
                  </div>
                </div>
                <Switch checked={oidcAutoRegister} onCheckedChange={setOidcAutoRegister} />
              </div>

              {oidcAutoRegister && (
                <div className="space-y-1.5 pt-2 border-t border-border/50">
                  <label className="font-semibold text-foreground">自动创建时的默认权限角色 (严禁直接赋予管理员)</label>
                  <select
                    value={oidcDefaultRole}
                    onChange={(e) => setOidcDefaultRole(e.target.value)}
                    className="w-full h-8 rounded-lg border border-input bg-background px-2.5 text-xs text-foreground"
                  >
                    <option value="operator">Operator (普通操作员 - 仅公众号日常业务权限)</option>
                    <option value="viewer">Viewer (只读访客 - 仅查看数据权限)</option>
                  </select>
                </div>
              )}
            </div>

            <DialogFooter className="pt-2">
              <Button type="button" variant="outline" onClick={() => setOidcModalOpen(false)}>
                取消
              </Button>
              <Button
                type="button"
                onClick={() =>
                  saveOidcMutation.mutate({
                    enabled: oidcEnabled,
                    issuer: oidcIssuer,
                    client_id: oidcClientId,
                    client_secret: oidcClientSecret,
                    redirect_uri: oidcRedirectUri,
                    scopes: oidcScopes,
                    auto_register: oidcAutoRegister,
                    default_role: oidcDefaultRole,
                  })
                }
                disabled={saveOidcMutation.isPending}
                className="bg-emerald-600 hover:bg-emerald-700"
              >
                {saveOidcMutation.isPending ? '保存中...' : '确定保存配置'}
              </Button>
            </DialogFooter>
          </div>
        </DialogContent>
      </Dialog>

      {/* 用户编辑/创建弹窗 */}
      <Dialog open={userModalOpen} onOpenChange={setUserModalOpen}>
        <DialogContent className="sm:max-w-md max-h-[85vh] overflow-y-auto">
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
              <label className="text-xs font-semibold flex items-center justify-between">
                <span>绑定邮箱 (用于 OIDC 单点登录自动匹配)</span>
                <span className="text-[10px] text-primary">推荐配置</span>
              </label>
              <Input
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="例如: user@company.com"
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
                {editingUserId ? '重置密码 (留空则保持不变)' : '登录密码 *'}
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
                <option value="admin">Admin (租户管理员)</option>
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
