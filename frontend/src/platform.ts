import { Capacitor } from '@capacitor/core'

export const isNative = Capacitor.isNativePlatform()

const SERVER_KEY = 'my-pet-server'

export type ServerScheme = 'http' | 'https'

function normalizeServer(raw: string): string {
  const u = raw.trim().replace(/\/+$/, '')
  if (!u) return ''
  return /^https?:\/\//i.test(u) ? u : `http://${u}`
}

export function parseServerUrl(raw: string): { scheme: ServerScheme; host: string } {
  const u = normalizeServer(raw)
  if (!u) return { scheme: 'http', host: '' }
  return u.toLowerCase().startsWith('https://')
    ? { scheme: 'https', host: u.slice(8) }
    : { scheme: 'http', host: u.slice(7) }
}

export function buildServerUrl(scheme: ServerScheme, host: string): string {
  const h = host.trim().replace(/^https?:\/\//i, '').replace(/\/+$/, '')
  return h ? `${scheme}://${h}` : ''
}

// 用户常直接粘贴完整网址：以输入里的协议为准，并同步回下拉框
export function splitServerInput(scheme: ServerScheme, raw: string): { scheme: ServerScheme; host: string } {
  const value = raw.trim()
  const m = value.match(/^(https?):\/\//i)
  if (!m) return { scheme, host: value }
  return { scheme: m[1].toLowerCase() as ServerScheme, host: value.slice(m[0].length) }
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
