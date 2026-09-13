import axios, {
  type AxiosError,
  type AxiosRequestConfig,
  type AxiosResponse
} from 'axios'
import { ElMessage } from 'element-plus'
import type { ApiResponse } from '@/types/api'
import { TOKEN_KEY, USER_KEY } from '@/constants'
import { mockAdapter } from '@/mocks'

export const USE_MOCK = import.meta.env.VITE_USE_MOCK === 'true'

const http = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '/api/v1',
  timeout: 20000
})

if (USE_MOCK) {
  http.defaults.adapter = mockAdapter
}

http.interceptors.request.use((config) => {
  const token = localStorage.getItem(TOKEN_KEY)
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

function handleUnauthorized(): void {
  localStorage.removeItem(TOKEN_KEY)
  localStorage.removeItem(USER_KEY)
  const { pathname, search, href } = window.location
  if (!pathname.startsWith('/login')) {
    const redirect = encodeURIComponent(pathname + search)
    window.location.replace(`/login?redirect=${redirect}`)
  } else if (!href.includes('redirect')) {
    window.location.replace('/login')
  }
}

http.interceptors.response.use(
  (response: AxiosResponse<ApiResponse<unknown>>) => {
    const body = response.data
    if (body && typeof body === 'object' && 'code' in body) {
      if (body.code === 0) return response
      if (body.code === 401) {
        handleUnauthorized()
      }
      ElMessage.error(body.message || '请求失败')
      return Promise.reject(new Error(body.message || '请求失败'))
    }
    return response
  },
  (error: AxiosError<ApiResponse<unknown>>) => {
    const status = error.response?.status
    if (status === 401) {
      handleUnauthorized()
    }
    const message = error.response?.data?.message || error.message || '网络异常，请稍后重试'
    ElMessage.error(message)
    return Promise.reject(error)
  }
)

export async function request<T>(config: AxiosRequestConfig): Promise<T> {
  const response = await http.request<ApiResponse<T>>(config)
  return response.data.data
}

export default http
