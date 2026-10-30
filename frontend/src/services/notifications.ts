import { App } from '@capacitor/app'
import { LocalNotifications } from '@capacitor/local-notifications'
import api from '@/api'
import { isNative } from '@/platform'
import type { Pet, Reminder } from '@/types'

const ENABLE_KEY = 'my-pet-notify'

export function isNotifyEnabled(): boolean {
  return localStorage.getItem(ENABLE_KEY) === '1'
}

// 后端提醒 id 是 UUID，本地通知只接受数字 id，用稳定哈希映射
function numericId(id: string): number {
  let h = 0
  for (let i = 0; i < id.length; i++) h = (Math.imul(31, h) + id.charCodeAt(i)) | 0
  return Math.abs(h) % 2000000000 || 1
}

async function clearScheduled() {
  const { notifications } = await LocalNotifications.getPending()
  if (notifications.length) {
    await LocalNotifications.cancel({ notifications: notifications.map((n) => ({ id: n.id })) })
  }
}

export async function hasPermission(): Promise<boolean> {
  const cur = await LocalNotifications.checkPermissions()
  if (cur.display === 'granted') return true
  if (cur.display === 'prompt' || cur.display === 'prompt-with-rationale') {
    const req = await LocalNotifications.requestPermissions()
    return req.display === 'granted'
  }
  return false
}

export async function syncNotifications(): Promise<number> {
  if (!isNative || !isNotifyEnabled()) return 0
  if (!localStorage.getItem('my-pet-token')) return 0
  if (!(await hasPermission())) return 0

  const { data: petData } = await api.get('/pets', { params: { page: 1, page_size: 50 } })
  const pets: Pet[] = petData.items || []
  const lists = await Promise.all(
    pets.map(async (p) => {
      const { data } = await api.get(`/pets/${p.id}/reminders`)
      return (data.items || []) as Reminder[]
    }),
  )

  const now = Date.now()
  const notifications = lists
    .flat()
    .filter((r) => r.enabled && r.next_due_at)
    .map((r) => {
      const due = new Date(r.next_due_at).getTime()
      const pet = pets.find((p) => p.id === r.pet_id)
      // 已过期的提醒在启动后立即补一条通知
      const at = Number.isFinite(due) && due > now + 60_000 ? due : now + 10_000
      return {
        id: numericId(r.id),
        title: `${pet?.name || '宠物'}：${r.type}`,
        body: r.notes || (due <= now ? '该提醒已到期，请及时处理' : '到期提醒'),
        schedule: { at: new Date(at), allowWhileIdle: true },
      }
    })

  await clearScheduled()
  if (notifications.length) await LocalNotifications.schedule({ notifications })
  return notifications.length
}

export async function setNotifyEnabled(on: boolean): Promise<number> {
  localStorage.setItem(ENABLE_KEY, on ? '1' : '0')
  if (!isNative) return 0
  if (!on) {
    await clearScheduled()
    return 0
  }
  if (!(await hasPermission())) {
    localStorage.setItem(ENABLE_KEY, '0')
    throw new Error('未获得通知权限，请在系统设置中开启')
  }
  return syncNotifications()
}

export function initNotifications() {
  if (!isNative) return
  App.addListener('appStateChange', ({ isActive }) => {
    if (isActive) syncNotifications().catch(() => {})
  })
  syncNotifications().catch(() => {})
}
