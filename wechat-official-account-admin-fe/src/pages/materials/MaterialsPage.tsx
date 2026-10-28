import React, { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api, PageResult } from '@/lib/api'
import { useAppStore } from '@/stores/app'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog'
import {
  Upload,
  Image as ImageIcon,
  Music,
  Video,
  FileText,
  Copy,
  Trash2,
  Eye,
  Plus,
  BookOpen,
  Send,
  ExternalLink,
} from 'lucide-react'
import { toast } from 'sonner'
import { formatDate } from '@/lib/utils'

export const MaterialsPage: React.FC = () => {
  const queryClient = useQueryClient()
  const { currentApp } = useAppStore()
  const [activeTab, setActiveTab] = useState('image')
  const [page, setPage] = useState(1)

  // 新建图文文章弹窗状态
  const [newsModalOpen, setNewsModalOpen] = useState(false)
  const [newsTitle, setNewsTitle] = useState('')
  const [newsAuthor, setNewsAuthor] = useState('')
  const [newsDigest, setNewsDigest] = useState('')
  const [newsContent, setNewsContent] = useState('')
  const [newsThumbMediaId, setNewsThumbMediaId] = useState('')
  const [newsThumbUrl, setNewsThumbUrl] = useState('')
  const [newsSourceUrl, setNewsSourceUrl] = useState('')

  // 图文正文查看弹窗
  const [viewArticle, setViewArticle] = useState<any>(null)

  // 获取素材列表
  const { data: pageData, isLoading } = useQuery<PageResult<any>>({
    queryKey: ['materials', currentApp?.appid, activeTab, page],
    queryFn: () => api.get(`/api/materials?type=${activeTab}&page=${page}&page_size=12`),
    enabled: !!currentApp,
  })

  // 获取图片素材列表 (用于图文封面选取)
  const { data: imageMaterials } = useQuery<PageResult<any>>({
    queryKey: ['materials-images-for-thumb', currentApp?.appid],
    queryFn: () => api.get(`/api/materials?type=image&page=1&page_size=20`),
    enabled: !!currentApp && newsModalOpen,
  })

  // 上传多媒体文件 (图片、语音、视频)
  const uploadMutation = useMutation({
    mutationFn: (formData: FormData) => {
      return api.post('/api/materials/upload', formData, {
        headers: { 'Content-Type': 'multipart/form-data' },
      })
    },
    onSuccess: () => {
      toast.success('素材文件上传成功！')
      queryClient.invalidateQueries({ queryKey: ['materials'] })
    },
  })

  // 创建图文文章 (结构化图文草稿)
  const createNewsMutation = useMutation({
    mutationFn: (values: any) => api.post('/api/materials/news', values),
    onSuccess: () => {
      toast.success('图文素材文章已成功创建并保存！')
      queryClient.invalidateQueries({ queryKey: ['materials'] })
      setNewsModalOpen(false)
      setNewsTitle('')
      setNewsAuthor('')
      setNewsDigest('')
      setNewsContent('')
      setNewsThumbMediaId('')
      setNewsThumbUrl('')
      setNewsSourceUrl('')
    },
    onError: (err: any) => {
      toast.error('创建失败: ' + (err.message || '网络错误'))
    },
  })

  // 删除素材
  const deleteMutation = useMutation({
    mutationFn: (id: number) => api.delete(`/api/materials/${id}`),
    onSuccess: () => {
      toast.success('素材已删除')
      queryClient.invalidateQueries({ queryKey: ['materials'] })
    },
  })

  const handleFileUpload = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return

    const formData = new FormData()
    formData.append('file', file)
    formData.append('media_type', activeTab)
    formData.append('title', file.name)

    uploadMutation.mutate(formData)
    e.target.value = ''
  }

  const copyToClipboard = (text: string, label: string) => {
    navigator.clipboard.writeText(text)
    toast.success(`已复制 ${label} 到剪贴板`)
  }

  const handleSaveNews = () => {
    if (!newsTitle.trim()) {
      toast.error('请输入图文文章标题')
      return
    }
    if (!newsContent.trim()) {
      toast.error('请输入图文文章正文内容')
      return
    }

    createNewsMutation.mutate({
      title: newsTitle.trim(),
      author: newsAuthor.trim(),
      digest: newsDigest.trim(),
      content: newsContent.trim(),
      thumb_media_id: newsThumbMediaId.trim(),
      thumb_url: newsThumbUrl.trim(),
      content_source_url: newsSourceUrl.trim(),
    })
  }

  return (
    <div className="space-y-4 sm:space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-bold tracking-tight">素材与多媒体管理</h2>
          <p className="text-sm text-muted-foreground">
            图片、语音、视频多媒体文件上传，以及结构化微信图文文章 (News) 编辑与发布
          </p>
        </div>

        {/* 右上角操作按钮：区分多媒体文件上传与图文文章新建 */}
        {activeTab === 'news' ? (
          <Button onClick={() => setNewsModalOpen(true)} className="w-full sm:w-auto min-h-[38px] gap-2 bg-emerald-600 hover:bg-emerald-700 justify-center">
            <Plus className="h-4 w-4" />
            <span>新建图文文章</span>
          </Button>
        ) : (
          <label className="cursor-pointer">
            <input
              type="file"
              className="hidden"
              onChange={handleFileUpload}
              accept={
                activeTab === 'image'
                  ? 'image/*'
                  : activeTab === 'voice'
                  ? 'audio/*'
                  : activeTab === 'video'
                  ? 'video/*'
                  : '*'
              }
            />
            <div className="inline-flex items-center justify-center rounded-lg text-sm font-medium bg-primary text-primary-foreground shadow hover:bg-primary/90 h-9 min-h-[38px] px-4 gap-2 w-full sm:w-auto">
              <Upload className="h-4 w-4" />
              <span>
                上传{activeTab === 'image' ? '图片' : activeTab === 'voice' ? '语音' : '视频'}文件
              </span>
            </div>
          </label>
        )}
      </div>

      {/* 分类 Tabs */}
      <Tabs
        value={activeTab}
        onValueChange={(val) => {
          setActiveTab(val)
          setPage(1)
        }}
      >
        <TabsList className="grid grid-cols-4 max-w-md w-full h-auto p-1">
          <TabsTrigger value="image" className="gap-1 sm:gap-1.5 text-[11px] sm:text-xs py-1.5 px-1">
            <ImageIcon className="h-3.5 w-3.5" />
            <span>图片素材</span>
          </TabsTrigger>
          <TabsTrigger value="news" className="gap-1 sm:gap-1.5 text-[11px] sm:text-xs py-1.5 px-1">
            <FileText className="h-3.5 w-3.5 text-blue-500" />
            <span>图文文章</span>
          </TabsTrigger>
          <TabsTrigger value="voice" className="gap-1 sm:gap-1.5 text-[11px] sm:text-xs py-1.5 px-1">
            <Music className="h-3.5 w-3.5" />
            <span>语音</span>
          </TabsTrigger>
          <TabsTrigger value="video" className="gap-1 sm:gap-1.5 text-[11px] sm:text-xs py-1.5 px-1">
            <Video className="h-3.5 w-3.5" />
            <span>视频</span>
          </TabsTrigger>
        </TabsList>

        <div className="mt-6">
          {isLoading ? (
            <div className="py-12 text-center text-muted-foreground">正在加载素材列表...</div>
          ) : pageData?.list && pageData.list.length > 0 ? (
            <div className="grid gap-3 sm:gap-4 md:gap-6 grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4">
              {pageData.list.map((item: any) => {
                const isNews = item.media_type === 'news'

                return (
                  <Card key={item.id} className="overflow-hidden group hover:shadow-md transition-shadow flex flex-col justify-between">
                    <div>
                      {/* 媒体封面展示 */}
                      <div className="h-44 bg-muted/50 relative overflow-hidden flex items-center justify-center">
                        {item.media_type === 'image' ? (
                          <img
                            src={item.url || `/api/materials/${item.id}/preview`}
                            referrerPolicy="no-referrer"
                            alt={item.title}
                            className="h-full w-full object-cover group-hover:scale-105 transition-transform duration-300"
                            onError={(e) => {
                              ;(e.target as HTMLElement).style.display = 'none'
                            }}
                          />
                        ) : isNews ? (
                          item.url ? (
                            <img
                              src={item.url}
                              referrerPolicy="no-referrer"
                              alt={item.title}
                              className="h-full w-full object-cover group-hover:scale-105 transition-transform duration-300"
                            />
                          ) : (
                            <div className="flex flex-col items-center gap-2 text-blue-600">
                              <BookOpen className="h-12 w-12" />
                              <span className="text-xs font-semibold">微信图文文章</span>
                            </div>
                          )
                        ) : (
                          <div className="flex flex-col items-center gap-2 text-muted-foreground">
                            {item.media_type === 'voice' ? (
                              <Music className="h-10 w-10 text-emerald-600" />
                            ) : (
                              <Video className="h-10 w-10 text-purple-600" />
                            )}
                            <span className="text-xs uppercase">{item.media_type}</span>
                          </div>
                        )}

                        <Badge variant="secondary" className="absolute top-2 right-2 text-[10px] bg-background/80 backdrop-blur-sm">
                          {isNews ? '图文草稿' : item.media_cat === 'permanent' ? '永久素材' : '临时素材'}
                        </Badge>
                      </div>

                      <CardContent className="p-4 space-y-2 text-xs">
                        <div className="font-semibold text-sm truncate text-foreground" title={item.title}>
                          {item.title || '未命名素材'}
                        </div>

                        {item.description && (
                          <p className="text-muted-foreground line-clamp-2 text-[11px] leading-relaxed">
                            {item.description}
                          </p>
                        )}

                        <div className="flex items-center justify-between text-muted-foreground font-mono text-[11px] bg-muted/50 p-1.5 rounded">
                          <span className="truncate max-w-[130px]" title={item.media_id}>
                            {item.media_id}
                          </span>
                          <button
                            onClick={() => copyToClipboard(item.media_id, 'MediaID')}
                            className="hover:text-primary shrink-0"
                            title="复制 Media ID"
                          >
                            <Copy className="h-3.5 w-3.5" />
                          </button>
                        </div>
                      </CardContent>
                    </div>

                    {/* 底部操作条 */}
                    <div className="px-4 pb-3 flex items-center justify-between text-[11px] text-muted-foreground border-t pt-2">
                      <span>{formatDate(item.created_at)}</span>
                      <div className="flex items-center gap-2">
                        {isNews ? (
                          <button
                            onClick={() => setViewArticle(item)}
                            className="text-primary hover:underline flex items-center gap-0.5"
                            title="查看文章正文详情"
                          >
                            <Eye className="h-3.5 w-3.5" />
                            <span>阅读正文</span>
                          </button>
                        ) : (
                          <a
                            href={item.url || `/api/materials/${item.id}/preview`}
                            target="_blank"
                            rel="noreferrer"
                            className="hover:text-primary p-0.5"
                            title="查看原文件/下载"
                          >
                            <Eye className="h-3.5 w-3.5" />
                          </a>
                        )}

                        <button
                          onClick={() => {
                            if (confirm('确认删除此素材？')) {
                              deleteMutation.mutate(item.id)
                            }
                          }}
                          className="hover:text-destructive p-0.5"
                          title="删除素材"
                        >
                          <Trash2 className="h-3.5 w-3.5" />
                        </button>
                      </div>
                    </div>
                  </Card>
                )
              })}
            </div>
          ) : (
            <Card className="p-12 text-center text-muted-foreground space-y-3">
              <div>当前分类下暂无素材</div>
              {activeTab === 'news' ? (
                <Button onClick={() => setNewsModalOpen(true)} size="sm" className="gap-1.5">
                  <Plus className="h-4 w-4" />
                  <span>新建第一篇图文文章</span>
                </Button>
              ) : (
                <p className="text-xs">点击右上角按钮上传多媒体文件</p>
              )}
            </Card>
          )}

          {/* 分页 */}
          {pageData?.pagination && pageData.pagination.total > 0 && (
            <div className="flex items-center justify-between text-xs text-muted-foreground pt-6">
              <span>共 {pageData.pagination.total} 项素材</span>
              <div className="flex items-center gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  disabled={page <= 1}
                  onClick={() => setPage(page - 1)}
                  className="h-8 text-xs"
                >
                  上一页
                </Button>
                <span>
                  第 {page} 页 / 共 {Math.ceil(pageData.pagination.total / pageData.pagination.page_size) || 1} 页
                </span>
                <Button
                  variant="outline"
                  size="sm"
                  disabled={page >= Math.ceil(pageData.pagination.total / pageData.pagination.page_size)}
                  onClick={() => setPage(page + 1)}
                  className="h-8 text-xs"
                >
                  下一页
                </Button>
              </div>
            </div>
          )}
        </div>
      </Tabs>

      {/* 新建图文文章编辑弹窗 (结构化微信文章) */}
      <Dialog open={newsModalOpen} onOpenChange={setNewsModalOpen}>
        <DialogContent className="sm:max-w-xl max-h-[85vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <FileText className="h-5 w-5 text-blue-600" />
              <span>新建微信图文文章 (News)</span>
            </DialogTitle>
          </DialogHeader>

          <div className="space-y-4 py-2 text-xs">
            <div className="space-y-1.5">
              <label className="font-semibold text-foreground">文章标题 *</label>
              <Input
                value={newsTitle}
                onChange={(e) => setNewsTitle(e.target.value)}
                placeholder="例如: 2026春季开学通知与校园指南"
              />
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <div className="space-y-1.5">
                <label className="font-semibold text-foreground">作者 (选填)</label>
                <Input
                  value={newsAuthor}
                  onChange={(e) => setNewsAuthor(e.target.value)}
                  placeholder="例如: 官方运营团队"
                />
              </div>

              <div className="space-y-1.5">
                <label className="font-semibold text-foreground">原文阅读跳转链接 (选填)</label>
                <Input
                  value={newsSourceUrl}
                  onChange={(e) => setNewsSourceUrl(e.target.value)}
                  placeholder="https://your-domain.com/full-article"
                />
              </div>
            </div>

            {/* 封面图片关联 */}
            <div className="space-y-2 p-3 bg-muted/40 rounded-xl border">
              <label className="font-semibold text-foreground flex items-center justify-between">
                <span>封面图片设置</span>
                <span className="text-[10px] text-muted-foreground">推荐比例 16:9 或 2.35:1</span>
              </label>

              <div className="flex gap-2">
                <Input
                  value={newsThumbUrl}
                  onChange={(e) => setNewsThumbUrl(e.target.value)}
                  placeholder="粘贴封面图片 URL 或在下方图片素材库中快速选取"
                  className="font-mono text-xs flex-1"
                />
              </div>

              {/* 从已上传的图片库快速选取 */}
              {imageMaterials?.list && imageMaterials.list.length > 0 && (
                <div className="space-y-1 pt-1">
                  <div className="text-[10.5px] text-muted-foreground">从图片库快速选取封面:</div>
                  <div className="flex gap-2 overflow-x-auto py-1">
                    {imageMaterials.list.slice(0, 6).map((img: any) => (
                      <button
                        key={img.id}
                        type="button"
                        onClick={() => {
                          setNewsThumbMediaId(img.media_id)
                          setNewsThumbUrl(img.url || `/api/materials/${img.id}/preview`)
                          toast.success('已选择该图片作为封面')
                        }}
                        className={`h-12 w-16 rounded border overflow-hidden shrink-0 transition-all ${
                          newsThumbMediaId === img.media_id ? 'ring-2 ring-primary border-primary' : 'opacity-70 hover:opacity-100'
                        }`}
                      >
                        <img src={img.url || `/api/materials/${img.id}/preview`} alt="" className="h-full w-full object-cover" />
                      </button>
                    ))}
                  </div>
                </div>
              )}
            </div>

            <div className="space-y-1.5">
              <label className="font-semibold text-foreground">图文摘要 (在微信聊天消息卡片中展示)</label>
              <textarea
                value={newsDigest}
                onChange={(e) => setNewsDigest(e.target.value)}
                rows={2}
                className="w-full rounded-lg border border-input bg-background p-2 text-xs focus:outline-none focus:ring-1 focus:ring-ring"
                placeholder="简要概括文章主旨..."
              />
            </div>

            <div className="space-y-1.5">
              <label className="font-semibold text-foreground">文章正文内容 (支持富文本排版/文字) *</label>
              <textarea
                value={newsContent}
                onChange={(e) => setNewsContent(e.target.value)}
                rows={6}
                className="w-full rounded-lg border border-input bg-background p-2.5 text-xs focus:outline-none focus:ring-1 focus:ring-ring font-sans"
                placeholder="输入图文正文内容..."
              />
            </div>

            <DialogFooter className="pt-2">
              <Button type="button" variant="outline" onClick={() => setNewsModalOpen(false)}>
                取消
              </Button>
              <Button
                type="button"
                onClick={handleSaveNews}
                disabled={createNewsMutation.isPending}
                className="bg-emerald-600 hover:bg-emerald-700"
              >
                {createNewsMutation.isPending ? '保存中...' : '确定保存图文'}
              </Button>
            </DialogFooter>
          </div>
        </DialogContent>
      </Dialog>

      {/* 查看文章正文详情弹窗 */}
      <Dialog open={!!viewArticle} onOpenChange={() => setViewArticle(null)}>
        <DialogContent className="sm:max-w-lg max-h-[85vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle className="text-base">{viewArticle?.title}</DialogTitle>
          </DialogHeader>

          {viewArticle && (
            <div className="space-y-3 py-2 text-xs leading-relaxed">
              {viewArticle.url && (
                <div className="rounded-xl overflow-hidden max-h-48 border">
                  <img src={viewArticle.url} alt="" className="h-full w-full object-cover" />
                </div>
              )}

              {viewArticle.description && (
                <div className="p-3 bg-muted/60 rounded-lg text-muted-foreground border italic">
                  摘要: {viewArticle.description}
                </div>
              )}

              <div className="p-3 bg-background rounded-lg border whitespace-pre-wrap text-foreground">
                {viewArticle.local_path || '无正文内容'}
              </div>

              <div className="flex items-center justify-between text-muted-foreground pt-2 border-t text-[11px] font-mono">
                <span>MediaID: {viewArticle.media_id}</span>
                <span>创建时间: {formatDate(viewArticle.created_at)}</span>
              </div>
            </div>
          )}
        </DialogContent>
      </Dialog>
    </div>
  )
}
