import type { Router } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

export function setupGuards(router: Router): void {
  router.beforeEach((to) => {
    const auth = useAuthStore()

    document.title = to.meta.title ? `${to.meta.title} · 爱教学` : '爱教学'

    const isPublic = to.matched.some((record) => record.meta.public === true)

    if (!auth.isLoggedIn && !isPublic) {
      return { name: 'login', query: { redirect: to.fullPath } }
    }

    if (auth.isLoggedIn && to.name === 'login') {
      return { name: 'dashboard' }
    }

    const roles = to.meta.roles
    if (roles && roles.length > 0 && !auth.hasRole(...roles)) {
      return { name: 'forbidden' }
    }

    return true
  })
}
