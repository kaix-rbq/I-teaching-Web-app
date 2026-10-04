import type { Router } from 'vue-router'
import { defaultRouteName } from '@/constants'
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
      return { name: defaultRouteName(auth.role) }
    }

    // 教师端无「质量驾驶舱」：直接落到「我的质量档案」，避免信息重复。
    if (auth.hasRole('teacher') && to.name === 'dashboard') {
      return { name: 'profile-quality' }
    }

    const roles = to.meta.roles
    if (roles && roles.length > 0 && !auth.hasRole(...roles)) {
      return { name: 'forbidden' }
    }

    return true
  })
}
