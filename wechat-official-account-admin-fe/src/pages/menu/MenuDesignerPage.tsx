import React, { useState, useEffect } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'
import { useAppStore } from '@/stores/app'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog'
import {
  Smartphone,
  Plus,
  Trash2,
  UploadCloud,
  DownloadCloud,
  Save,
  CheckCircle2,
  ArrowLeft,
  ArrowRight,
  ArrowUp,
  ArrowDown,
  Layers,
  Sparkles,
  RotateCcw,
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

  const [activeTab, setActiveTab] = useState<'normal' | 'conditional'>('normal')
  const [buttons, setButtons] = useState<MenuItem[]>([])
  const [selectedPath, setSelectedPath] = useState<number[]>([0])

  // 个性化菜单创建弹窗
  const [condModalOpen, setCondModalOpen] = useState(false)
  const [condName, setCondName] = useState('')
  const [condTagId, setCondTagId] = useState('')
  const [condSex, setCondSex] = useState('0')
  const [condPlatform, setCondPlatform] = useState('0')

  // 获取普通默认菜单
  const { data: menuData } = useQuery<any>({
    queryKey: ['menu', currentApp?.appid],
    queryFn: () => api.get('/api/menu'),
    enabled: !!currentApp,
  })

  // 获取个性化菜单列表
  const { data: conditionalMenus } = useQuery<any[]>({
    queryKey: ['menu-conditional', currentApp?.appid],
    queryFn: () => api.get('/api/menu/conditional'),
    enabled: !!currentApp && activeTab === 'conditional',
  })

  // 获取微信标签 (用于个性化匹配)
  const { data: tags } = useQuery<any[]>({
    queryKey: ['fans-tags', currentApp?.appid],
    queryFn: () => api.get('/api/fans/tags'),
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
    mutationFn: () => api.post('/api/menu', { menu_data: JSON.stringify({ button: buttons }) }),
    onSuccess: () => {
      toast.success('自定义菜单草稿已保存')
      queryClient.invalidateQueries({ queryKey: ['menu', currentApp?.appid] })
    },
  })

  // 发布到微信
  const publishMutation = useMutation({
    mutationFn: async () => {
      await api.post('/api/menu', { menu_data: JSON.stringify({ button: buttons }) })
      return api.post('/api/menu/publish')
    },
    onSuccess: () => {
      toast.success('菜单已成功发布到微信服务器！')
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
          if (Array.isArray(parsed.button)) setButtons(parsed.button)
        } catch {}
      }
    },
  })

  // 保存并发布个性化菜单
  const saveCondMutation = useMutation({
    mutationFn: (payload: any) => api.post('/api/menu/conditional', payload),
    onSuccess: () => {
      toast.success('个性化条件菜单已发布到微信！')
      queryClient.invalidateQueries({ queryKey: ['menu-conditional'] })
      setCondModalOpen(false)
    },
  })

  // 删除个性化菜单
  const deleteCondMutation = useMutation({
    mutationFn: (id: number) => api.delete(`/api/menu/conditional/${id}`),
    onSuccess: () => {
      toast.success('个性化菜单已删除')
      queryClient.invalidateQueries({ queryKey: ['menu-conditional'] })
    },
  })

  const getCurrentItem = (): MenuItem | null => {
    if (buttons.length === 0 || selectedPath.length === 0) return null
    const [p0, p1] = selectedPath
    if (p1 !== undefined && buttons[p0]?.sub_button) {
      return buttons[p0].sub_button![p1] || null
    }
    return buttons[p0] || null
  }

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

  // 一级菜单位置左移/右移
  const moveTopMenu = (index: number, direction: 'left' | 'right') => {
    const target = direction === 'left' ? index - 1 : index + 1
    if (target < 0 || target >= buttons.length) return
    const cloned = [...buttons]
    const temp = cloned[index]
    cloned[index] = cloned[target]
    cloned[target] = temp
    setButtons(cloned)
    setSelectedPath([target])
  }

  // 二级子菜单位置上移/下移
  const moveSubMenu = (p0: number, p1: number, direction: 'up' | 'down') => {
    const target = direction === 'up' ? p1 - 1 : p1 + 1
    const subList = buttons[p0]?.sub_button || []
    if (target < 0 || target >= subList.length) return
    const cloned = JSON.parse(JSON.stringify(buttons))
    const temp = cloned[p0].sub_button[p1]
    cloned[p0].sub_button[p1] = cloned[p0].sub_button[target]
    cloned[p0].sub_button[target] = temp
    setButtons(cloned)
    setSelectedPath([p0, target])
  }

  const addTopMenu = () => {
    if (buttons.length >= 3) {
      toast.error('一级主菜单最多只能添加 3 个')
      return
    }
    const newItem: MenuItem = { name: `菜单${buttons.length + 1}`, type: 'view', url: 'https://' }
    setButtons([...buttons, newItem])
    setSelectedPath([buttons.length])
  }

  const addSubMenu = (p0: number) => {
    const parent = buttons[p0]
    const subList = parent.sub_button || []
    if (subList.length >= 5) {
      toast.error('二级子菜单最多只能添加 5 个')
      return
    }
    const newSub: MenuItem = { name: `子菜单${subList.length + 1}`, type: 'view', url: 'https://' }
    const updated = JSON.parse(JSON.stringify(buttons))
    if (!updated[p0].sub_button) updated[p0].sub_button = []
    updated[p0].sub_button.push(newSub)
    delete updated[p0].type
    delete updated[p0].url
    delete updated[p0].key
    setButtons(updated)
    setSelectedPath([p0, subList.length])
  }

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
      {/* 顶部操作条 */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-bold tracking-tight flex items-center gap-2">
            <span>微信自定义菜单设计器</span>
            {menuData?.published && activeTab === 'normal' && (
              <Badge variant="success" className="text-xs gap-1">
                <CheckCircle2 className="h-3 w-3" />
                已发布 ({formatDate(menuData.published_at)})
              </Badge>
            )}
          </h2>
          <p className="text-sm text-muted-foreground">
            可视化树形编辑一级/二级菜单，支持排序微调、普通菜单发布与个性化条件规则菜单
          </p>
        </div>

        {activeTab === 'normal' ? (
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => {
                if (confirm('确认清空当前所有菜单项吗？')) {
                  setButtons([])
                  setSelectedPath([])
                }
              }}
              className="text-xs text-muted-foreground hover:text-destructive"
            >
              <RotateCcw className="h-3.5 w-3.5 mr-1" />
              <span>清空重置</span>
            </Button>
            <Button variant="outline" size="sm" onClick={() => syncMutation.mutate()} disabled={syncMutation.isPending} className="text-xs">
              <DownloadCloud className="h-3.5 w-3.5 mr-1" />
              <span>从微信同步</span>
            </Button>
            <Button variant="secondary" size="sm" onClick={() => saveDraftMutation.mutate()} disabled={saveDraftMutation.isPending} className="text-xs">
              <Save className="h-3.5 w-3.5 mr-1" />
              <span>保存草稿</span>
            </Button>
            <Button size="sm" onClick={() => publishMutation.mutate()} disabled={publishMutation.isPending} className="text-xs bg-emerald-600 hover:bg-emerald-700 text-white font-semibold">
              <UploadCloud className="h-3.5 w-3.5 mr-1" />
              <span>{publishMutation.isPending ? '发布中...' : '发布到微信'}</span>
            </Button>
          </div>
        ) : (
          <Button onClick={() => setCondModalOpen(true)} className="gap-1.5 text-xs bg-emerald-600 hover:bg-emerald-700">
            <Plus className="h-4 w-4" />
            <span>新建个性化条件菜单</span>
          </Button>
        )}
      </div>

      {/* Tabs 分组 */}
      <Tabs value={activeTab} onValueChange={(val: any) => setActiveTab(val)}>
        <TabsList className="grid grid-cols-2 max-w-xs">
          <TabsTrigger value="normal" className="text-xs">普通默认菜单</TabsTrigger>
          <TabsTrigger value="conditional" className="text-xs">个性化条件菜单</TabsTrigger>
        </TabsList>

        <TabsContent value="normal" className="mt-6">
          {/* 设计器双栏布局：左手机预览，右属性面板 */}
          <div className="grid gap-6 lg:grid-cols-12 items-start">
            {/* 左侧：手机效果模拟器 */}
            <div className="lg:col-span-5 flex justify-center">
              <div className="w-[320px] h-[580px] rounded-[2.5rem] border-8 border-slate-800 dark:border-slate-700 bg-slate-900 shadow-2xl relative flex flex-col overflow-hidden">
                <div className="h-6 w-full flex justify-center items-center">
                  <div className="h-3 w-20 rounded-full bg-slate-800"></div>
                </div>

                <div className="h-10 bg-slate-800 text-white px-4 flex items-center justify-between text-xs font-medium border-b border-slate-700">
                  <span className="truncate max-w-[160px]">{currentApp ? currentApp.name : '微信公众号'}</span>
                  <span className="text-[10px] text-slate-400">···</span>
                </div>

                <div className="flex-1 bg-slate-100 dark:bg-slate-950 p-4 flex flex-col justify-end space-y-3">
                  <div className="max-w-[85%] rounded-2xl rounded-tl-sm bg-white dark:bg-slate-900 border p-3 text-xs shadow-sm">
                    👋 您好！请在下方模拟菜单中点击体验对应功能。
                  </div>
                </div>

                {/* 手机底部自定义菜单区域 */}
                <div className="h-14 bg-white dark:bg-slate-900 border-t border-slate-200 dark:border-slate-800 flex items-center relative select-none">
                  <div className="w-9 flex items-center justify-center text-slate-400">
                    <Smartphone className="h-4 w-4" />
                  </div>

                  <div className="flex-1 h-full flex divide-x divide-slate-200 dark:divide-slate-800 border-l border-slate-200 dark:border-slate-800">
                    {buttons.map((btn, p0) => {
                      const isTopSelected = selectedPath[0] === p0 && selectedPath.length === 1
                      const isAnySubSelected = selectedPath[0] === p0

                      return (
                        <div key={p0} className="flex-1 h-full relative group">
                          {isAnySubSelected && (
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
                      <CardDescription className="text-xs">配置显示名称、顺序与点击动作</CardDescription>
                    </div>

                    {/* 排序微调与删除 */}
                    <div className="flex items-center gap-1">
                      {selectedPath.length === 1 ? (
                        <>
                          <Button
                            variant="ghost"
                            size="icon"
                            disabled={selectedPath[0] <= 0}
                            onClick={() => moveTopMenu(selectedPath[0], 'left')}
                            title="向左移动菜单"
                            className="h-8 w-8"
                          >
                            <ArrowLeft className="h-3.5 w-3.5" />
                          </Button>
                          <Button
                            variant="ghost"
                            size="icon"
                            disabled={selectedPath[0] >= buttons.length - 1}
                            onClick={() => moveTopMenu(selectedPath[0], 'right')}
                            title="向右移动菜单"
                            className="h-8 w-8"
                          >
                            <ArrowRight className="h-3.5 w-3.5" />
                          </Button>
                        </>
                      ) : (
                        <>
                          <Button
                            variant="ghost"
                            size="icon"
                            disabled={selectedPath[1] <= 0}
                            onClick={() => moveSubMenu(selectedPath[0], selectedPath[1], 'up')}
                            title="向上移动"
                            className="h-8 w-8"
                          >
                            <ArrowUp className="h-3.5 w-3.5" />
                          </Button>
                          <Button
                            variant="ghost"
                            size="icon"
                            disabled={selectedPath[1] >= (buttons[selectedPath[0]]?.sub_button?.length || 0) - 1}
                            onClick={() => moveSubMenu(selectedPath[0], selectedPath[1], 'down')}
                            title="向下移动"
                            className="h-8 w-8"
                          >
                            <ArrowDown className="h-3.5 w-3.5" />
                          </Button>
                        </>
                      )}
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={deleteCurrentItem}
                        className="text-destructive hover:bg-destructive/10 h-8 w-8 ml-1"
                        title="删除菜单项"
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                      </Button>
                    </div>
                  </CardHeader>

                  <CardContent className="space-y-4 pt-2 text-sm">
                    <div className="space-y-1.5">
                      <label className="text-xs font-semibold">菜单名称 *</label>
                      <Input
                        value={currentItem.name || ''}
                        maxLength={selectedPath.length === 2 ? 8 : 4}
                        onChange={(e) => updateCurrentItem({ name: e.target.value })}
                        placeholder="输入菜单名称"
                      />
                    </div>

                    {selectedPath.length === 1 && currentItem.sub_button && currentItem.sub_button.length > 0 ? (
                      <div className="p-4 rounded-xl bg-muted/60 text-xs text-muted-foreground space-y-1">
                        <div className="font-semibold text-foreground">💡 提示</div>
                        <div>当前一级主菜单已包含子菜单，微信规范要求点击后仅展开子菜单列表，请点击子菜单配置具体响应行为。</div>
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

                        {currentItem.type === 'view' && (
                          <div className="space-y-1.5">
                            <label className="text-xs font-semibold">网页 URL *</label>
                            <Input
                              value={currentItem.url || ''}
                              onChange={(e) => updateCurrentItem({ url: e.target.value })}
                              placeholder="https://example.com"
                            />
                          </div>
                        )}

                        {currentItem.type === 'click' && (
                          <div className="space-y-1.5">
                            <label className="text-xs font-semibold">事件 Key / 匹配关键词 *</label>
                            <Input
                              value={currentItem.key || ''}
                              onChange={(e) => updateCurrentItem({ key: e.target.value })}
                              placeholder="例如: HELP 或 客服咨询"
                            />
                          </div>
                        )}

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
                              <label className="text-xs font-semibold">备用网页 URL</label>
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
                <Card className="p-12 text-center text-muted-foreground">
                  请在左侧手机预览中点击选中或添加菜单项
                </Card>
              )}
            </div>
          </div>
        </TabsContent>

        {/* 个性化条件菜单列表 */}
        <TabsContent value="conditional" className="mt-6 space-y-4">
          {conditionalMenus && conditionalMenus.length > 0 ? (
            <div className="grid gap-4">
              {conditionalMenus.map((cond) => {
                let rule: any = {}
                try {
                  rule = JSON.parse(cond.match_rule || '{}')
                } catch {}

                return (
                  <Card key={cond.id} className="p-4 flex items-center justify-between">
                    <div className="space-y-1">
                      <div className="flex items-center gap-2">
                        <span className="font-bold text-base">{cond.name}</span>
                        <Badge variant="outline" className="text-xs font-mono">
                          MenuID: {cond.menu_id}
                        </Badge>
                      </div>
                      <div className="text-xs text-muted-foreground flex gap-4">
                        <span>匹配标签: {rule.tag_id ? `Tag ${rule.tag_id}` : '不限'}</span>
                        <span>性别: {rule.sex === '1' ? '男' : rule.sex === '2' ? '女' : '不限'}</span>
                        <span>系统平台: {rule.client_platform_type === '1' ? 'iOS' : rule.client_platform_type === '2' ? 'Android' : '不限'}</span>
                      </div>
                    </div>

                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => {
                        if (confirm(`确认删除个性化菜单 "${cond.name}"？`)) {
                          deleteCondMutation.mutate(cond.id)
                        }
                      }}
                      className="text-destructive hover:bg-destructive/10"
                    >
                      <Trash2 className="h-4 w-4 mr-1" />
                      <span>删除</span>
                    </Button>
                  </Card>
                )
              })}
            </div>
          ) : (
            <Card className="p-12 text-center text-muted-foreground">
              暂未发布任何个性化条件菜单，点击右上角「新建个性化条件菜单」
            </Card>
          )}
        </TabsContent>
      </Tabs>

      {/* 新建个性化菜单弹窗 */}
      <Dialog open={condModalOpen} onOpenChange={setCondModalOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>新建个性化条件菜单</DialogTitle>
          </DialogHeader>

          <div className="space-y-4 py-2 text-sm">
            <div className="space-y-1.5">
              <label className="text-xs font-semibold">条件菜单名称 *</label>
              <Input value={condName} onChange={(e) => setCondName(e.target.value)} placeholder="例如: VIP学员专享菜单" />
            </div>

            <div className="space-y-1.5">
              <label className="text-xs font-semibold">粉丝标签匹配</label>
              <select
                value={condTagId}
                onChange={(e) => setCondTagId(e.target.value)}
                className="w-full h-9 rounded-lg border border-input bg-background px-3 text-sm"
              >
                <option value="">不限粉丝标签</option>
                {tags?.map((t) => (
                  <option key={t.tag_id} value={t.tag_id}>
                    {t.name} ({t.count}人)
                  </option>
                ))}
              </select>
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1.5">
                <label className="text-xs font-semibold">性别匹配</label>
                <select
                  value={condSex}
                  onChange={(e) => setCondSex(e.target.value)}
                  className="w-full h-9 rounded-lg border border-input bg-background px-3 text-sm"
                >
                  <option value="0">不限性别</option>
                  <option value="1">男</option>
                  <option value="2">女</option>
                </select>
              </div>

              <div className="space-y-1.5">
                <label className="text-xs font-semibold">客户端平台</label>
                <select
                  value={condPlatform}
                  onChange={(e) => setCondPlatform(e.target.value)}
                  className="w-full h-9 rounded-lg border border-input bg-background px-3 text-sm"
                >
                  <option value="0">不限系统</option>
                  <option value="1">IOS</option>
                  <option value="2">Android</option>
                </select>
              </div>
            </div>

            <DialogFooter className="pt-2">
              <Button variant="outline" onClick={() => setCondModalOpen(false)}>取消</Button>
              <Button
                onClick={() => {
                  if (!condName.trim()) {
                    toast.error('请输入菜单名称')
                    return
                  }
                  const matchrule: any = {}
                  if (condTagId) matchrule.tag_id = condTagId
                  if (condSex !== '0') matchrule.sex = condSex
                  if (condPlatform !== '0') matchrule.client_platform_type = condPlatform

                  saveCondMutation.mutate({
                    name: condName,
                    menu_data: JSON.stringify({ button: buttons }),
                    match_rule: JSON.stringify(matchrule),
                  })
                }}
                disabled={saveCondMutation.isPending}
              >
                {saveCondMutation.isPending ? '发布中...' : '确定保存并发布'}
              </Button>
            </DialogFooter>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  )
}
