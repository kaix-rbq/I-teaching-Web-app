import 'vue-router'
import type { Role } from './user'

declare module 'vue-router' {
  interface RouteMeta {
    title?: string
    public?: boolean
    roles?: Role[]
  }
}
