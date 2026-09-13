import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { loginApi, logoutApi } from '@/api/auth'
import { TOKEN_KEY, USER_KEY } from '@/constants'
import type { LoginPayload, Role, User } from '@/types/user'

function readStoredUser(): User | null {
  try {
    const raw = localStorage.getItem(USER_KEY)
    return raw ? (JSON.parse(raw) as User) : null
  } catch {
    return null
  }
}

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string>(localStorage.getItem(TOKEN_KEY) ?? '')
  const user = ref<User | null>(readStoredUser())

  const isLoggedIn = computed(() => Boolean(token.value))
  const role = computed<Role | undefined>(() => user.value?.role)

  function persist(): void {
    if (token.value) {
      localStorage.setItem(TOKEN_KEY, token.value)
    } else {
      localStorage.removeItem(TOKEN_KEY)
    }
    if (user.value) {
      localStorage.setItem(USER_KEY, JSON.stringify(user.value))
    } else {
      localStorage.removeItem(USER_KEY)
    }
  }

  function hasRole(...roles: Role[]): boolean {
    if (!user.value) return false
    return roles.includes(user.value.role)
  }

  async function login(payload: LoginPayload): Promise<User> {
    const result = await loginApi(payload)
    token.value = result.token
    user.value = result.user
    persist()
    return result.user
  }

  async function logout(): Promise<void> {
    try {
      await logoutApi()
    } finally {
      reset()
    }
  }

  function reset(): void {
    token.value = ''
    user.value = null
    persist()
  }

  function updateUser(next: User): void {
    user.value = next
    persist()
  }

  return {
    token,
    user,
    isLoggedIn,
    role,
    hasRole,
    login,
    logout,
    reset,
    updateUser
  }
})
