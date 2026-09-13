import { request } from './http'
import type { Resource } from '@/types/resource'

export function fetchResourcesApi(courseId: string): Promise<Resource[]> {
  return request<Resource[]>({
    url: `/courses/${courseId}/resources`,
    method: 'get'
  })
}

export function uploadResourceApi(
  courseId: string,
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

export function deleteResourceApi(id: string): Promise<null> {
  return request<null>({ url: `/resources/${id}`, method: 'delete' })
}
