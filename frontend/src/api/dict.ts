import { request } from './http'
import type { DictData } from '@/types/dict'

export function fetchDictApi(): Promise<DictData> {
  return request<DictData>({ url: '/dict', method: 'get' })
}
