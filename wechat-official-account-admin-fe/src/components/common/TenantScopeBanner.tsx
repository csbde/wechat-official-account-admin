import React from 'react'
import { useQuery } from '@tanstack/react-query'
import { Building2, ShieldAlert } from 'lucide-react'
import { api } from '@/lib/api'
import { useAuthStore } from '@/stores/auth'
import { Badge } from '@/components/ui/badge'

/**
 * 租户视角提示条：明确告知当前数据归属哪个租户组织，避免“为什么新建的账号都在本租户”的困惑。
 * 超级管理员处于代管视角时会高亮警示。
 */
export const TenantScopeBanner: React.FC<{ extra?: string }> = ({ extra }) => {
  const { user, tenant, activeTenantId } = useAuthStore()
  const isSuperAdmin = user?.role === 'superadmin'

  const { data: tenants } = useQuery<any[]>({
    queryKey: ['system-tenants'],
    queryFn: () => api.get('/api/system/tenants'),
    enabled: isSuperAdmin,
    retry: 0,
  })
  const view = activeTenantId ? tenants?.find((t) => t.id === activeTenantId) : null
  const name = view?.name || tenant?.name || '默认组织'

  return (
    <div
      className={`flex flex-wrap items-center gap-2 text-xs px-3 py-2 rounded-lg border ${
        view
          ? 'bg-amber-500/10 border-amber-500/25 text-amber-700 dark:text-amber-300'
          : 'bg-muted/50 border-border/60 text-muted-foreground'
      }`}
    >
      {view ? (
        <ShieldAlert className="h-3.5 w-3.5 shrink-0" />
      ) : (
        <Building2 className="h-3.5 w-3.5 shrink-0 text-primary" />
      )}
      <span>
        当前租户视角：
        <span className="font-semibold text-foreground"> {name}</span>
        <span className="font-mono opacity-70"> ({view?.code || tenant?.code || 'default'})</span>
      </span>
      <Badge variant="outline" className="text-[10px]">
        {view ? '代管视角' : '本租户'}
      </Badge>
      <span className="opacity-80">
        {extra || '此处新增的公众号、菜单、素材、粉丝等数据都将归属该租户，与其他租户完全隔离'}
      </span>
    </div>
  )
}
