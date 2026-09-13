import { computed } from 'vue'
import { useAuthStore } from '@/stores/auth'
import type { Role } from '@/types/user'

export function useAuth() {
  const auth = useAuthStore()

  return {
    auth,
    isLoggedIn: computed(() => auth.isLoggedIn),
    user: computed(() => auth.user),
    role: computed(() => auth.role),
    isDirector: computed(() => auth.role === 'director'),
    isTeacher: computed(() => auth.role === 'teacher'),
    isSupervisor: computed(() => auth.role === 'supervisor'),
    hasRole: (...roles: Role[]) => auth.hasRole(...roles)
  }
}
