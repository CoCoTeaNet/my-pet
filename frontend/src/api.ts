import axios from 'axios'
import { getApiBase, goLogin } from '@/platform'

// Web 端走 vite 代理的同源 /api；原生 App 端走用户在登录页填写的服务器地址
const api = axios.create({ timeout: 15000 })

api.interceptors.request.use((config) => {
  config.baseURL = getApiBase()
  const token = localStorage.getItem('my-pet-token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

api.interceptors.response.use(
  (res) => res,
  (err) => {
    const status = err?.response?.status
    const message: string = err?.response?.data?.error || '网络超时，请重试'
    if (status === 401) {
      localStorage.removeItem('my-pet-token')
      localStorage.removeItem('my-pet-user')
      goLogin()
    }
    return Promise.reject(new Error(message))
  },
)

export default api
