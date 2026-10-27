import React, { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import * as z from 'zod'
import { api } from '@/lib/api'
import { useAuthStore } from '@/stores/auth'
import { Card, CardHeader, CardTitle, CardDescription, CardContent, CardFooter } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Radio, Lock, User, ArrowRight, ShieldCheck, Key } from 'lucide-react'
import { toast } from 'sonner'

const loginSchema = z.object({
  username: z.string().min(1, '请输入登录账号'),
  password: z.string().min(1, '请输入密码'),
})

type LoginFormData = z.infer<typeof loginSchema>

export const LoginPage: React.FC = () => {
  const { setAuth } = useAuthStore()
  const [loading, setLoading] = useState(false)

  const form = useForm<LoginFormData>({
    resolver: zodResolver(loginSchema),
    defaultValues: {
      username: 'admin',
      password: 'admin123',
    },
  })

  const onSubmit = async (values: LoginFormData) => {
    try {
      setLoading(true)
      const res: any = await api.post('/api/auth/login', values)
      if (res.token && res.user) {
        setAuth(res.token, res.user)
        toast.success(`欢迎回来，${res.user.nickname || res.user.username}！`)
      }
    } catch (err: any) {
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  const handleQuickFill = () => {
    form.setValue('username', 'admin')
    form.setValue('password', 'admin123')
    toast.info('已快速填入默认账号: admin / admin123')
  }

  const handleSSO = async () => {
    try {
      setLoading(true)
      const res: any = await api.post('/api/auth/sso/callback', {
        provider: 'wechat_work',
        code: 'sso_demo_user',
      })
      if (res.token && res.user) {
        setAuth(res.token, res.user)
        toast.success('SSO 单点登录验证成功！')
      }
    } catch (err) {
      toast.error('SSO 模拟登录失败')
    } finally {
      setLoading(false)
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
            微信公众号综合管理运营平台 (v2.0)
          </p>
        </div>

        {/* 登录卡片 */}
        <Card className="border-border/60 bg-card/90 backdrop-blur-xl shadow-2xl">
          <CardHeader className="space-y-1 pb-4">
            <CardTitle className="text-xl">账号密码登录</CardTitle>
            <CardDescription className="text-xs">
              输入您的后台管理员或操作员凭据进入系统
            </CardDescription>
          </CardHeader>

          <CardContent>
            <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
              <div className="space-y-1.5">
                <label className="text-xs font-semibold flex items-center gap-1.5">
                  <User className="h-3.5 w-3.5 text-muted-foreground" />
                  <span>用户名</span>
                </label>
                <Input
                  placeholder="admin"
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
                  placeholder="admin123"
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
            <div className="w-full flex items-center justify-between text-xs text-muted-foreground pt-4">
              <button
                type="button"
                onClick={handleQuickFill}
                className="hover:text-primary transition-colors flex items-center gap-1 text-primary font-medium"
              >
                <Key className="h-3.5 w-3.5" />
                <span>填入初始账号 (admin / admin123)</span>
              </button>
            </div>

            <div className="w-full pt-2">
              <Button
                variant="outline"
                type="button"
                onClick={handleSSO}
                className="w-full text-xs h-9 border-dashed gap-1.5"
              >
                <ShieldCheck className="h-4 w-4 text-emerald-500" />
                <span>企业微信 / SSO 单点登录</span>
              </Button>
            </div>
          </CardFooter>
        </Card>

        {/* 底部版权 */}
        <div className="text-center text-xs text-slate-400">
          Powered by Go Gin & React TypeScript · MIT Licensed
        </div>
      </div>
    </div>
  )
}
