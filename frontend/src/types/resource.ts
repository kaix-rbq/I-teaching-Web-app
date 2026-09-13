export type ResourceType = 'pdf' | 'doc' | 'ppt' | 'video' | 'zip' | 'other'

export interface Resource {
  id: number
  courseId: number
  name: string
  type: ResourceType
  size: number
  uploader: string
  uploadedAt: string
}
