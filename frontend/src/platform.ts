import { Capacitor } from '@capacitor/core'

export const isNative = Capacitor.isNativePlatform()

const SERVER_KEY = 'my-pet-server'

function normalizeServer(raw: string): string {
  const u = raw.trim().replace(/\/+$/, '')
  if (!u) return ''
  return /^https?:\/\//i.test(u) ? u : `http://${u}`
}

export function getServerUrl(): string {
  return localStorage.getItem(SERVER_KEY) || ''
}

export function setServerUrl(raw: string): string {
  const url = normalizeServer(raw)
  if (url) localStorage.setItem(SERVER_KEY, url)
  return url
}

export function getApiBase(): string {
  if (!isNative) return '/api'
  const server = getServerUrl()
  return server ? `${server}/api` : '/api'
}

// 后端返回站内绝对路径（如 /api/files/avatars/x.jpg），原生环境下需补上服务器地址
export function resolveUrl(path: string): string {
  if (!path || !isNative || /^https?:\/\//i.test(path)) return path
  const server = getServerUrl()
  return server ? server + path : path
}

export function goLogin() {
  if (isNative) {
    if (!location.hash.startsWith('#/login')) location.hash = '#/login'
  } else if (!location.pathname.startsWith('/login')) {
    location.href = '/login'
  }
}
