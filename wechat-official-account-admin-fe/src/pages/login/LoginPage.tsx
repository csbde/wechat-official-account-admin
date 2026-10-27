import React, { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import * as z from 'zod'
import { api } from '@/lib/api'
import { useAuthStore } from '@/stores/auth'
import { Card, CardHeader, CardTitle, CardDescription, CardContent, CardFooter } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Radio, Lock, User, ArrowRight, ShieldCheck, Fingerprint, Mail } from 'lucide-react'
import { toast } from 'sonner'

const loginSchema = z.object({
  username: z.string().min(1, '请输入登录用户名或邮箱'),
  password: z.string().min(1, '请输入密码'),
})

type LoginFormData = z.infer<typeof loginSchema>

export const LoginPage: React.FC = () => {
  const { setAuth } = useAuthStore()
  const [loading, setLoading] = useState(false)
  const [loadingOIDC, setLoadingOIDC] = useState(false)

  const form = useForm<LoginFormData>({
    resolver: zodResolver(loginSchema),
    // 出于安全考虑，登录页不再预填任何默认凭据
    defaultValues: {
      username: '',
      password: '',
    },
  })

  const onSubmit = async (values: LoginFormData) => {
    try {
      setLoading(true)
      const res: any = await api.post('/api/auth/login', values)
      if (res.token && res.user) {
        setAuth(res.token, res.user, res.tenant)
        toast.success(`欢迎回来，${res.user.nickname || res.user.username}！`)
      }
    } catch (err: any) {
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  // 发起 OIDC 单点登录 (自动按邮箱匹配)
  const handleOIDCLogin = async () => {
    try {
      setLoadingOIDC(true)
      const res: any = await api.get('/api/auth/oidc/login')
      if (res.auth_url) {
        toast.loading('正在跳转至 OIDC 身份认证服务...')
        window.location.href = res.auth_url
      }
    } catch (e: any) {
      toast.error('OIDC 发起失败: ' + (e.message || '网络错误'))
    } finally {
      setLoadingOIDC(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center p-4 bg-gradient-to-br from-slate-900 via-slate-800 to-emerald-950 text-foreground">
      <div className="w-full max-w-md space-y-6">
        {/* Logo 区域 */}
        <div className="text-center space-y-2">
          <div className="inline-flex h-14 w-14 rounded-2xl bg-gradient-to-tr from-emerald-500 to-green-400 items-center justify-center text-white shadow-xl shadow-emerald-500/30">
            <Radio className="h-8 w-8" />
          </div>
          <h1 className="text-2xl font-extrabold tracking-tight text-white sm:text-3xl">
            WOAA Admin Pro
          </h1>
          <p className="text-sm text-slate-300">
            微信公众号综合管理平台 · 支持 OIDC 单点登录
          </p>
        </div>

        {/* 登录卡片 */}
        <Card className="border-border/60 bg-card/90 backdrop-blur-xl shadow-2xl">
          <CardHeader className="space-y-1 pb-4">
            <CardTitle className="text-xl">账号密码登录</CardTitle>
            <CardDescription className="text-xs">
              输入您的管理员用户名或邮箱直接登录系统
            </CardDescription>
          </CardHeader>

          <CardContent>
            <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
              <div className="space-y-1.5">
                <label className="text-xs font-semibold flex items-center gap-1.5">
                  <User className="h-3.5 w-3.5 text-muted-foreground" />
                  <span>用户名 / 邮箱</span>
                </label>
                <Input
                  placeholder="admin 或 admin@kkio.com"
                  {...form.register('username')}
                  className="bg-background/60"
                />
                {form.formState.errors.username && (
                  <span className="text-[11px] text-destructive">
                    {form.formState.errors.username.message}
                  </span>
                )}
              </div>

              <div className="space-y-1.5">
                <label className="text-xs font-semibold flex items-center gap-1.5">
                  <Lock className="h-3.5 w-3.5 text-muted-foreground" />
                  <span>密码</span>
                </label>
                <Input
                  type="password"
                  placeholder="请输入登录密码"
                  {...form.register('password')}
                  className="bg-background/60"
                />
                {form.formState.errors.password && (
                  <span className="text-[11px] text-destructive">
                    {form.formState.errors.password.message}
                  </span>
                )}
              </div>

              <Button
                type="submit"
                disabled={loading}
                className="w-full bg-emerald-600 hover:bg-emerald-700 text-white font-semibold h-10 shadow-lg shadow-emerald-600/25 mt-2 gap-2"
              >
                <span>{loading ? '正在验证身份...' : '立即登录'}</span>
                <ArrowRight className="h-4 w-4" />
              </Button>
            </form>
          </CardContent>

          <CardFooter className="flex flex-col gap-3 pt-0 border-t border-border/40 mt-2">
            <div className="w-full flex items-center justify-center text-[11px] text-muted-foreground pt-3">
              <span>请使用管理员分配的账号登录；忘记密码或首次使用请联系平台超级管理员</span>
            </div>

            {/* OIDC 企业单点登录按钮 */}
            <div className="w-full space-y-2 pt-1">
              <Button
                variant="outline"
                type="button"
                onClick={handleOIDCLogin}
                disabled={loadingOIDC}
                className="w-full text-xs h-9 border-dashed border-emerald-500/50 hover:bg-emerald-500/10 gap-1.5 font-medium text-foreground"
              >
                <Fingerprint className="h-4 w-4 text-emerald-500" />
                <span>OIDC 企业单点登录 (自动按邮箱匹配绑定)</span>
              </Button>
            </div>
          </CardFooter>
        </Card>

        {/* 底部版权 */}
        <div className="text-center text-xs text-slate-400">
          Powered by Go 1.26+ & React 18 · MIT Licensed
        </div>
      </div>
    </div>
  )
}
