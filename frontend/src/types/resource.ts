export type ResourceType = 'pdf' | 'doc' | 'ppt' | 'video' | 'zip' | 'other'

export interface Resource {
  id: string
  courseId: string
  name: string
  type: ResourceType
  size: number
  uploader: string
  uploadedAt: string
}
