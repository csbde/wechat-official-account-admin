import React, { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api, PageResult } from '@/lib/api'
import { useAppStore } from '@/stores/app'
import { Button } from '@/components/ui/button'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import { Badge } from '@/components/ui/badge'
import {
  Upload,
  Image as ImageIcon,
  Music,
  Video,
  FileText,
  Copy,
  Trash2,
  ExternalLink,
  Plus,
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
      toast.success('素材上传成功')
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

  const copyMediaID = (id: string) => {
    navigator.clipboard.writeText(id)
    toast.success('已复制 MediaID 到剪贴板')
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-bold tracking-tight">素材与多媒体管理</h2>
          <p className="text-sm text-muted-foreground">
            管理微信永久素材与临时素材库，包括图片、语音、视频和图文，支持本地直连缓存
          </p>
        </div>

        {/* 上传按钮 */}
        <label className="cursor-pointer">
          <input
            type="file"
            className="hidden"
            onChange={handleFileUpload}
            accept={activeTab === 'image' ? 'image/*' : activeTab === 'voice' ? 'audio/*' : activeTab === 'video' ? 'video/*' : '*'}
          />
          <div className="inline-flex items-center justify-center rounded-lg text-sm font-medium bg-primary text-primary-foreground shadow hover:bg-primary/90 h-9 px-4 gap-2">
            <Upload className="h-4 w-4" />
            <span>上传{activeTab === 'image' ? '图片' : activeTab === 'voice' ? '语音' : '素材'}</span>
          </div>
        </label>
      </div>

      {/* 类型分类 Tabs */}
      <Tabs value={activeTab} onValueChange={(val) => { setActiveTab(val); setPage(1) }}>
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
                  {/* 图片封面展示 */}
                  <div className="h-44 bg-muted/50 relative overflow-hidden flex items-center justify-center">
                    {item.media_type === 'image' ? (
                      <img
                        src={item.url || `/api/materials/${item.id}/preview`}
                        alt={item.title}
                        className="h-full w-full object-cover group-hover:scale-105 transition-transform duration-300"
                        onError={(e) => {
                          // 图片加载失败时展示回退图标
                          ;(e.target as HTMLElement).style.display = 'none'
                        }}
                      />
                    ) : (
                      <div className="flex flex-col items-center gap-2 text-muted-foreground">
                        {item.media_type === 'voice' ? <Music className="h-10 w-10" /> : <Video className="h-10 w-10" />}
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
                      <span className="truncate max-w-[140px]" title={item.media_id}>
                        {item.media_id}
                      </span>
                      <button
                        onClick={() => copyMediaID(item.media_id)}
                        className="hover:text-primary shrink-0"
                        title="复制 Media ID"
                      >
                        <Copy className="h-3.5 w-3.5" />
                      </button>
                    </div>

                    <div className="flex items-center justify-between pt-1 text-[11px] text-muted-foreground border-t">
                      <span>{formatDate(item.created_at)}</span>
                      <button
                        onClick={() => {
                          if (confirm('确认删除此素材？')) {
                            deleteMutation.mutate(item.id)
                          }
                        }}
                        className="text-muted-foreground hover:text-destructive"
                        title="删除素材"
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                      </button>
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
        </div>
      </Tabs>
    </div>
  )
}
