import type { ApiResponse } from '@/types/api'
import type { User } from '@/types/user'

export interface MockContext {
  method: string
  path: string
  query: Record<string, string>
  params: Record<string, string>
  body: Record<string, unknown>
  formData: FormData | null
  token: string | null
  user: User | null
}

export type MockHandler = (
  ctx: MockContext
) => ApiResponse<unknown> | Promise<ApiResponse<unknown>>

export interface MockRoute {
  method: string
  path: string
  handler: MockHandler
}

export function route(method: string, path: string, handler: MockHandler): MockRoute {
  return { method: method.toLowerCase(), path, handler }
}

function matchPath(pattern: string, path: string): Record<string, string> | null {
  const patternParts = pattern.split('/').filter(Boolean)
  const pathParts = path.split('/').filter(Boolean)
  if (patternParts.length !== pathParts.length) return null

  const params: Record<string, string> = {}
  for (let i = 0; i < patternParts.length; i += 1) {
    const patternPart = patternParts[i]
    const pathPart = pathParts[i]
    if (patternPart.startsWith(':')) {
      params[patternPart.slice(1)] = decodeURIComponent(pathPart)
    } else if (patternPart !== pathPart) {
      return null
    }
  }
  return params
}

export function matchRoute(
  routes: MockRoute[],
  method: string,
  path: string
): { route: MockRoute; params: Record<string, string> } | null {
  for (const item of routes) {
    if (item.method !== method) continue
    const params = matchPath(item.path, path)
    if (params) return { route: item, params }
  }
  return null
}
