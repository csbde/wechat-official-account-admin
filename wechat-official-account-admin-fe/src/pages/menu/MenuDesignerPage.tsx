import React, { useState, useEffect } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'
import { useAppStore } from '@/stores/app'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import {
  Smartphone,
  Plus,
  Trash2,
  UploadCloud,
  DownloadCloud,
  Save,
  CheckCircle2,
  ExternalLink,
  ChevronRight,
  Sparkles,
} from 'lucide-react'
import { toast } from 'sonner'
import { formatDate } from '@/lib/utils'

interface MenuItem {
  name: string
  type?: string
  url?: string
  key?: string
  appid?: string
  pagepath?: string
  sub_button?: MenuItem[]
}

export const MenuDesignerPage: React.FC = () => {
  const queryClient = useQueryClient()
  const { currentApp } = useAppStore()

  // 菜单树数据
  const [buttons, setButtons] = useState<MenuItem[]>([])
  const [selectedPath, setSelectedPath] = useState<number[]>([0]) // [一级索引, 二级索引?]

  // 获取草稿菜单
  const { data: menuData, isLoading } = useQuery<any>({
    queryKey: ['menu', currentApp?.appid],
    queryFn: () => api.get('/api/menu'),
    enabled: !!currentApp,
  })

  useEffect(() => {
    if (menuData?.menu_data) {
      try {
        const parsed = JSON.parse(menuData.menu_data)
        if (Array.isArray(parsed.button)) {
          setButtons(parsed.button)
        }
      } catch (e) {
        console.error(e)
      }
    }
  }, [menuData])

  // 保存草稿
  const saveDraftMutation = useMutation({
    mutationFn: () => {
      const payload = { menu_data: JSON.stringify({ button: buttons }) }
      return api.post('/api/menu', payload)
    },
    onSuccess: () => {
      toast.success('自定义菜单草稿已保存')
      queryClient.invalidateQueries({ queryKey: ['menu', currentApp?.appid] })
    },
  })

  // 发布到微信
  const publishMutation = useMutation({
    mutationFn: async () => {
      // 先保存草稿
      await api.post('/api/menu', { menu_data: JSON.stringify({ button: buttons }) })
      // 再发布
      return api.post('/api/menu/publish')
    },
    onSuccess: () => {
      toast.success('菜单已成功发布到微信服务器，微信客户端通常在24小时内生效')
      queryClient.invalidateQueries({ queryKey: ['menu', currentApp?.appid] })
    },
  })

  // 从微信同步
  const syncMutation = useMutation({
    mutationFn: () => api.post('/api/menu/sync'),
    onSuccess: (data: any) => {
      toast.success('已从微信服务器同步线上菜单')
      queryClient.invalidateQueries({ queryKey: ['menu', currentApp?.appid] })
      if (data?.menu_data) {
        try {
          const parsed = JSON.parse(data.menu_data)
          if (Array.isArray(parsed.button)) {
            setButtons(parsed.button)
          }
        } catch (e) {}
      }
    },
  })

  // 获取当前正在编辑的菜单项
  const getCurrentItem = (): MenuItem | null => {
    if (buttons.length === 0 || selectedPath.length === 0) return null
    const [p0, p1] = selectedPath
    if (p1 !== undefined && buttons[p0]?.sub_button) {
      return buttons[p0].sub_button![p1] || null
    }
    return buttons[p0] || null
  }

  // 更新当前选中的菜单项属性
  const updateCurrentItem = (patch: Partial<MenuItem>) => {
    const [p0, p1] = selectedPath
    setButtons((prev) => {
      const cloned = JSON.parse(JSON.stringify(prev))
      if (p1 !== undefined) {
        cloned[p0].sub_button[p1] = { ...cloned[p0].sub_button[p1], ...patch }
      } else {
        cloned[p0] = { ...cloned[p0], ...patch }
      }
      return cloned
    })
  }

  // 添加一级菜单
  const addTopMenu = () => {
    if (buttons.length >= 3) {
      toast.error('一级菜单最多只能添加 3 个')
      return
    }
    const newItem: MenuItem = {
      name: `菜单${buttons.length + 1}`,
      type: 'view',
      url: 'https://',
    }
    setButtons([...buttons, newItem])
    setSelectedPath([buttons.length])
  }

  // 添加二级子菜单
  const addSubMenu = (parentIndex: number) => {
    const parent = buttons[parentIndex]
    const subList = parent.sub_button || []
    if (subList.length >= 5) {
      toast.error('二级子菜单最多只能添加 5 个')
      return
    }
    const newSub: MenuItem = {
      name: `子菜单${subList.length + 1}`,
      type: 'view',
      url: 'https://',
    }
    const updated = JSON.parse(JSON.stringify(buttons))
    if (!updated[parentIndex].sub_button) {
      updated[parentIndex].sub_button = []
    }
    updated[parentIndex].sub_button.push(newSub)
    // 父级不再需要独立响应动作
    delete updated[parentIndex].type
    delete updated[parentIndex].url
    delete updated[parentIndex].key
    setButtons(updated)
    setSelectedPath([parentIndex, subList.length])
  }

  // 删除菜单
  const deleteCurrentItem = () => {
    const [p0, p1] = selectedPath
    const cloned = JSON.parse(JSON.stringify(buttons))
    if (p1 !== undefined) {
      cloned[p0].sub_button.splice(p1, 1)
      if (cloned[p0].sub_button.length === 0) {
        delete cloned[p0].sub_button
        cloned[p0].type = 'view'
        cloned[p0].url = 'https://'
      }
      setButtons(cloned)
      setSelectedPath([p0])
    } else {
      cloned.splice(p0, 1)
      setButtons(cloned)
      setSelectedPath(cloned.length > 0 ? [0] : [])
    }
  }

  const currentItem = getCurrentItem()

  return (
    <div className="space-y-6">
      {/* 顶部标题栏与发布按钮 */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-bold tracking-tight flex items-center gap-2">
            <span>微信自定义菜单设计器</span>
            {menuData?.published && (
              <Badge variant="success" className="text-xs gap-1">
                <CheckCircle2 className="h-3 w-3" />
                已发布 ({formatDate(menuData.published_at)})
              </Badge>
            )}
          </h2>
          <p className="text-sm text-muted-foreground">
            可视化编辑一级/二级菜单及响应动作，支持一键发布与线上同步
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            onClick={() => syncMutation.mutate()}
            disabled={syncMutation.isPending}
            className="gap-1.5 text-xs"
          >
            <DownloadCloud className="h-4 w-4" />
            <span>从微信同步</span>
          </Button>
          <Button
            variant="secondary"
            onClick={() => saveDraftMutation.mutate()}
            disabled={saveDraftMutation.isPending}
            className="gap-1.5 text-xs"
          >
            <Save className="h-4 w-4" />
            <span>保存草稿</span>
          </Button>
          <Button
            onClick={() => publishMutation.mutate()}
            disabled={publishMutation.isPending}
            className="gap-1.5 text-xs bg-emerald-600 hover:bg-emerald-700 text-white font-semibold"
          >
            <UploadCloud className="h-4 w-4" />
            <span>{publishMutation.isPending ? '发布中...' : '发布到微信'}</span>
          </Button>
        </div>
      </div>

      {/* 设计器双栏布局：左手机预览，右属性面板 */}
      <div className="grid gap-6 lg:grid-cols-12 items-start">
        {/* 左侧：手机效果模拟器 */}
        <div className="lg:col-span-5 flex justify-center">
          <div className="w-[320px] h-[600px] rounded-[2.5rem] border-8 border-slate-800 dark:border-slate-700 bg-slate-900 shadow-2xl relative flex flex-col overflow-hidden">
            {/* 手机顶部刘海 */}
            <div className="h-6 w-full flex justify-center items-center">
              <div className="h-3.5 w-24 rounded-full bg-slate-800"></div>
            </div>

            {/* 微信头部导航栏 */}
            <div className="h-11 bg-slate-800 text-white px-4 flex items-center justify-between text-xs font-medium border-b border-slate-700">
              <span className="truncate max-w-[160px]">{currentApp ? currentApp.name : '微信公众号'}</span>
              <span className="text-[10px] text-slate-400">···</span>
            </div>

            {/* 聊天内容区域 (背景展示) */}
            <div className="flex-1 bg-slate-100 dark:bg-slate-950 p-4 flex flex-col justify-end space-y-3">
              <div className="max-w-[80%] rounded-2xl rounded-tl-sm bg-white dark:bg-slate-900 border p-3 text-xs shadow-sm">
                👋 您好！欢迎关注。请在下方自定义菜单中点击体验对应功能。
              </div>
            </div>

            {/* 手机底部自定义菜单区域 */}
            <div className="h-14 bg-white dark:bg-slate-900 border-t border-slate-200 dark:border-slate-800 flex items-center relative select-none">
              <div className="w-10 flex items-center justify-center text-slate-400 hover:text-slate-600 cursor-pointer">
                <Smartphone className="h-4 w-4" />
              </div>

              {/* 菜单项容器 */}
              <div className="flex-1 h-full flex divide-x divide-slate-200 dark:divide-slate-800 border-l border-slate-200 dark:border-slate-800">
                {buttons.map((btn, p0) => {
                  const isTopSelected = selectedPath[0] === p0 && selectedPath.length === 1
                  const isAnySubSelected = selectedPath[0] === p0
                  const hasSub = btn.sub_button && btn.sub_button.length > 0

                  return (
                    <div key={p0} className="flex-1 h-full relative group">
                      {/* 二级菜单弹出悬浮框 */}
                      {(isAnySubSelected || false) && (
                        <div className="absolute bottom-16 left-0 right-0 bg-white dark:bg-slate-900 border rounded-xl shadow-xl overflow-hidden divide-y text-xs">
                          {btn.sub_button?.map((sub, p1) => {
                            const isSubSelected = selectedPath[0] === p0 && selectedPath[1] === p1
                            return (
                              <button
                                key={p1}
                                onClick={() => setSelectedPath([p0, p1])}
                                className={`w-full py-2.5 px-2 text-center truncate block transition-colors ${
                                  isSubSelected
                                    ? 'bg-primary text-primary-foreground font-semibold'
                                    : 'hover:bg-muted text-foreground'
                                }`}
                              >
                                {sub.name || '子菜单'}
                              </button>
                            )
                          })}
                          {(!btn.sub_button || btn.sub_button.length < 5) && (
                            <button
                              onClick={() => addSubMenu(p0)}
                              className="w-full py-2 px-2 text-center text-primary font-medium hover:bg-primary/10 flex items-center justify-center gap-1"
                            >
                              <Plus className="h-3.5 w-3.5" />
                              <span>添加子菜单</span>
                            </button>
                          )}
                        </div>
                      )}

                      {/* 一级菜单按钮 */}
                      <button
                        onClick={() => setSelectedPath([p0])}
                        className={`w-full h-full flex items-center justify-center text-xs px-1 font-medium transition-colors ${
                          isTopSelected
                            ? 'bg-primary/10 text-primary font-bold'
                            : 'text-slate-700 dark:text-slate-200 hover:bg-slate-50 dark:hover:bg-slate-800'
                        }`}
                      >
                        <span className="truncate">{btn.name || '一级菜单'}</span>
                      </button>
                    </div>
                  )
                })}

                {/* 增加一级菜单按钮 */}
                {buttons.length < 3 && (
                  <button
                    onClick={addTopMenu}
                    className="flex-1 h-full flex items-center justify-center text-slate-400 hover:text-primary hover:bg-primary/5 transition-colors"
                    title="添加一级菜单"
                  >
                    <Plus className="h-4 w-4" />
                  </button>
                )}
              </div>
            </div>
          </div>
        </div>

        {/* 右侧：属性编辑卡片 */}
        <div className="lg:col-span-7">
          {currentItem ? (
            <Card>
              <CardHeader className="flex flex-row items-center justify-between pb-3">
                <div>
                  <CardTitle className="text-base flex items-center gap-2">
                    <span>编辑菜单项: {currentItem.name}</span>
                    <Badge variant="outline">
                      {selectedPath.length === 2 ? '二级子菜单' : '一级主菜单'}
                    </Badge>
                  </CardTitle>
                  <CardDescription className="text-xs">配置该菜单在微信客户端中的点击交互行为</CardDescription>
                </div>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={deleteCurrentItem}
                  className="text-destructive hover:bg-destructive/10 text-xs gap-1"
                >
                  <Trash2 className="h-3.5 w-3.5" />
                  <span>删除本项</span>
                </Button>
              </CardHeader>

              <CardContent className="space-y-4 pt-2 text-sm">
                <div className="space-y-1.5">
                  <label className="text-xs font-semibold">菜单名称 *</label>
                  <Input
                    value={currentItem.name || ''}
                    maxLength={selectedPath.length === 2 ? 8 : 4}
                    onChange={(e) => updateCurrentItem({ name: e.target.value })}
                    placeholder="输入菜单名称 (一级最多4字，二级最多8字)"
                  />
                </div>

                {/* 如果是一级菜单并且有子菜单，则禁止直接绑定响应动作 */}
                {selectedPath.length === 1 && currentItem.sub_button && currentItem.sub_button.length > 0 ? (
                  <div className="p-4 rounded-xl bg-muted/60 text-xs text-muted-foreground space-y-1">
                    <div className="font-semibold text-foreground">💡 提示</div>
                    <div>当前一级菜单已添加子菜单，点击该主菜单将仅展开子菜单列表，请点击子菜单配置具体响应行为。</div>
                  </div>
                ) : (
                  <>
                    <div className="space-y-1.5">
                      <label className="text-xs font-semibold">动作类型 (Type) *</label>
                      <Select
                        value={currentItem.type || 'view'}
                        onValueChange={(val) => updateCurrentItem({ type: val })}
                      >
                        <SelectTrigger>
                          <SelectValue placeholder="选择动作类型" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="view">跳转网页 (view)</SelectItem>
                          <SelectItem value="click">点击触发事件/自动回复 (click)</SelectItem>
                          <SelectItem value="miniprogram">跳转微信小程序 (miniprogram)</SelectItem>
                          <SelectItem value="scancode_push">扫码推事件 (scancode_push)</SelectItem>
                          <SelectItem value="pic_photo_or_album">发图拍照或相册 (pic_photo_or_album)</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>

                    {/* view 类型：输入跳转网页 URL */}
                    {currentItem.type === 'view' && (
                      <div className="space-y-1.5">
                        <label className="text-xs font-semibold">页面 URL *</label>
                        <Input
                          value={currentItem.url || ''}
                          onChange={(e) => updateCurrentItem({ url: e.target.value })}
                          placeholder="https://example.com"
                        />
                        <p className="text-[11px] text-muted-foreground">用户点击菜单后将通过微信内置浏览器打开此链接</p>
                      </div>
                    )}

                    {/* click 类型：输入 EventKey 关键词 */}
                    {currentItem.type === 'click' && (
                      <div className="space-y-1.5">
                        <label className="text-xs font-semibold">事件 Key / 触发关键词 *</label>
                        <Input
                          value={currentItem.key || ''}
                          onChange={(e) => updateCurrentItem({ key: e.target.value })}
                          placeholder="例如: KEY_SERVICE_HELP 或 关键词"
                        />
                        <p className="text-[11px] text-muted-foreground">点击后向开发者服务器发送 click 事件，匹配后自动回复</p>
                      </div>
                    )}

                    {/* 小程序跳转 */}
                    {currentItem.type === 'miniprogram' && (
                      <div className="space-y-3">
                        <div className="space-y-1.5">
                          <label className="text-xs font-semibold">小程序 AppID *</label>
                          <Input
                            value={currentItem.appid || ''}
                            onChange={(e) => updateCurrentItem({ appid: e.target.value })}
                            placeholder="wx987654321..."
                          />
                        </div>
                        <div className="space-y-1.5">
                          <label className="text-xs font-semibold">小程序页面路径 *</label>
                          <Input
                            value={currentItem.pagepath || ''}
                            onChange={(e) => updateCurrentItem({ pagepath: e.target.value })}
                            placeholder="pages/index/index"
                          />
                        </div>
                        <div className="space-y-1.5">
                          <label className="text-xs font-semibold">备用网页 URL (低版本不支持小程序时展示)</label>
                          <Input
                            value={currentItem.url || ''}
                            onChange={(e) => updateCurrentItem({ url: e.target.value })}
                            placeholder="https://example.com"
                          />
                        </div>
                      </div>
                    )}
                  </>
                )}
              </CardContent>
            </Card>
          ) : (
            <Card className="p-8 text-center text-muted-foreground">
              请在左侧手机预览界面点击选中或添加菜单项进行编辑
            </Card>
          )}
        </div>
      </div>
    </div>
  )
}
