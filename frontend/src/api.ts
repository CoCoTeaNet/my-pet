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
    let message: string = err?.response?.data?.error
    if (!message) {
      if (status === 413) message = '上传内容过大（HTTP 413）'
      else if (status) message = `请求失败（HTTP ${status}）`
      else if (/timeout/i.test(err?.message || '')) message = '请求超时，请重试'
      else message = '网络连接失败，请检查服务器地址'
    }
    if (status === 401) {
      localStorage.removeItem('my-pet-token')
      localStorage.removeItem('my-pet-user')
      goLogin()
    }
    return Promise.reject(new Error(message))
  },
)

export default api
