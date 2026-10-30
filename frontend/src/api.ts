import axios from 'axios'

// 联调说明：开发环境通过 vite.config.ts 将 /api 代理到 http://localhost:8080，
// 无需 mock，启动后端后前端即可直连真实接口。
const api = axios.create({
  baseURL: '/api',
  timeout: 15000,
})

api.interceptors.request.use((config) => {
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
      if (!location.pathname.startsWith('/login')) {
        location.href = '/login'
      }
    }
    return Promise.reject(new Error(message))
  },
)

export default api
