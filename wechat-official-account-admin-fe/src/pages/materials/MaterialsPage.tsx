import React, { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api, PageResult } from '@/lib/api'
import { useAppStore } from '@/stores/app'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Badge } from '@/components/ui/badge'
import {
  Upload,
  Image as ImageIcon,
  Music,
  Video,
  FileText,
  Copy,
  Trash2,
  Download,
  Eye,
} from 'lucide-react'
import { toast } from 'sonner'
import { formatDate } from '@/lib/utils'

export const MaterialsPage: React.FC = () => {
  const queryClient = useQueryClient()
  const { currentApp } = useAppStore()
  const [activeTab, setActiveTab] = useState('image')
  const [page, setPage] = useState(1)

  // 获取素材列表
  const { data: pageData, isLoading } = useQuery<PageResult<any>>({
    queryKey: ['materials', currentApp?.appid, activeTab, page],
    queryFn: () => api.get(`/api/materials?type=${activeTab}&page=${page}&page_size=12`),
    enabled: !!currentApp,
  })

  // 上传素材
  const uploadMutation = useMutation({
    mutationFn: (formData: FormData) => {
      return api.post('/api/materials/upload', formData, {
        headers: { 'Content-Type': 'multipart/form-data' },
      })
    },
    onSuccess: () => {
      toast.success('素材上传成功！')
      queryClient.invalidateQueries({ queryKey: ['materials'] })
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

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-bold tracking-tight">素材与多媒体管理</h2>
          <p className="text-sm text-muted-foreground">
            图片、语音、视频与图文永久/临时素材库，支持本地文件直传、预览与 MediaID 快速调用
          </p>
        </div>

        {/* 上传按钮 */}
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
          <div className="inline-flex items-center justify-center rounded-lg text-sm font-medium bg-primary text-primary-foreground shadow hover:bg-primary/90 h-9 px-4 gap-2">
            <Upload className="h-4 w-4" />
            <span>
              上传{activeTab === 'image' ? '图片' : activeTab === 'voice' ? '语音' : activeTab === 'video' ? '视频' : '素材'}
            </span>
          </div>
        </label>
      </div>

      {/* 分类 Tabs */}
      <Tabs
        value={activeTab}
        onValueChange={(val) => {
          setActiveTab(val)
          setPage(1)
        }}
      >
        <TabsList className="grid grid-cols-4 max-w-md">
          <TabsTrigger value="image" className="gap-1.5 text-xs">
            <ImageIcon className="h-3.5 w-3.5" />
            <span>图片</span>
          </TabsTrigger>
          <TabsTrigger value="voice" className="gap-1.5 text-xs">
            <Music className="h-3.5 w-3.5" />
            <span>语音</span>
          </TabsTrigger>
          <TabsTrigger value="video" className="gap-1.5 text-xs">
            <Video className="h-3.5 w-3.5" />
            <span>视频</span>
          </TabsTrigger>
          <TabsTrigger value="news" className="gap-1.5 text-xs">
            <FileText className="h-3.5 w-3.5" />
            <span>图文</span>
          </TabsTrigger>
        </TabsList>

        <div className="mt-6">
          {isLoading ? (
            <div className="py-12 text-center text-muted-foreground">正在加载素材列表...</div>
          ) : pageData?.list && pageData.list.length > 0 ? (
            <div className="grid gap-6 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4">
              {pageData.list.map((item: any) => (
                <Card key={item.id} className="overflow-hidden group hover:shadow-md transition-shadow">
                  {/* 媒体封面展示 */}
                  <div className="h-44 bg-muted/50 relative overflow-hidden flex items-center justify-center">
                    {item.media_type === 'image' ? (
                      <img
                        src={item.url || `/api/materials/${item.id}/preview`}
                        alt={item.title}
                        className="h-full w-full object-cover group-hover:scale-105 transition-transform duration-300"
                        onError={(e) => {
                          ;(e.target as HTMLElement).style.display = 'none'
                        }}
                      />
                    ) : (
                      <div className="flex flex-col items-center gap-2 text-muted-foreground">
                        {item.media_type === 'voice' ? (
                          <Music className="h-10 w-10 text-emerald-600" />
                        ) : item.media_type === 'video' ? (
                          <Video className="h-10 w-10 text-purple-600" />
                        ) : (
                          <FileText className="h-10 w-10 text-blue-600" />
                        )}
                        <span className="text-xs uppercase">{item.media_type}</span>
                      </div>
                    )}
                    <Badge variant="secondary" className="absolute top-2 right-2 text-[10px] bg-background/80 backdrop-blur-sm">
                      {item.media_cat === 'permanent' ? '永久' : '临时'}
                    </Badge>
                  </div>

                  <CardContent className="p-4 space-y-2 text-xs">
                    <div className="font-semibold truncate text-foreground" title={item.title}>
                      {item.title || '未命名素材'}
                    </div>

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

                    <div className="flex items-center justify-between pt-1 text-[11px] text-muted-foreground border-t">
                      <span>{formatDate(item.created_at)}</span>
                      <div className="flex items-center gap-1">
                        <a
                          href={item.url || `/api/materials/${item.id}/preview`}
                          target="_blank"
                          rel="noreferrer"
                          className="hover:text-primary p-1"
                          title="查看原文件/下载"
                        >
                          <Eye className="h-3.5 w-3.5" />
                        </a>
                        <button
                          onClick={() => {
                            if (confirm('确认删除此素材？')) {
                              deleteMutation.mutate(item.id)
                            }
                          }}
                          className="hover:text-destructive p-1"
                          title="删除素材"
                        >
                          <Trash2 className="h-3.5 w-3.5" />
                        </button>
                      </div>
                    </div>
                  </CardContent>
                </Card>
              ))}
            </div>
          ) : (
            <Card className="p-12 text-center text-muted-foreground">
              当前分类下暂无素材，点击右上角上传
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
    </div>
  )
}
