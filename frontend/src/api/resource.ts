import http, { request } from './http'
import type { Resource } from '@/types/resource'

export function fetchResourcesApi(courseId: number | string): Promise<Resource[]> {
  return request<Resource[]>({
    url: `/courses/${courseId}/resources`,
    method: 'get'
  })
}

export function uploadResourceApi(
  courseId: number | string,
  file: File,
  onProgress?: (percent: number) => void
): Promise<Resource> {
  const formData = new FormData()
  formData.append('file', file)
  return request<Resource>({
    url: `/courses/${courseId}/resources`,
    method: 'post',
    data: formData,
    headers: { 'Content-Type': 'multipart/form-data' },
    onUploadProgress: (event) => {
      if (!onProgress) return
      const total = event.total ?? 0
      const percent = total > 0 ? Math.round((event.loaded / total) * 100) : 0
      onProgress(Math.min(percent, 100))
    }
  })
}

export function deleteResourceApi(id: number | string): Promise<null> {
  return request<null>({ url: `/resources/${id}`, method: 'delete' })
}

/**
 * 下载资源。下载接口需要 Authorization 头，无法用 <a href> 直链，
 * 因此以 blob 方式取回后由前端触发保存。
 */
export async function downloadResourceApi(id: number | string): Promise<Blob> {
  const response = await http.get(`/resources/${id}/download`, {
    responseType: 'blob'
  })
  return response.data as Blob
}
