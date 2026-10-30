const pad = (n: number) => String(n).padStart(2, '0')

export function parseDate(s?: string | null): Date | null {
  if (!s) return null
  const d = new Date(s)
  return Number.isNaN(d.getTime()) ? null : d
}

export function dateKey(d: Date): string {
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

export function formatClock(d: Date): string {
  return `${pad(d.getHours())}:${pad(d.getMinutes())}`
}

export function formatTimeOnly(s?: string | null): string {
  const d = parseDate(s)
  return d ? formatClock(d) : ''
}

function isSameDay(a: Date, b: Date): boolean {
  return a.getFullYear() === b.getFullYear() && a.getMonth() === b.getMonth() && a.getDate() === b.getDate()
}

/**
 * S7 时间表述：今天 08:30 / 昨天 14:00 / 3 天前 / 10月25日（跨年带年份）
 */
export function formatRelative(s?: string | null, withTime = false): string {
  const d = parseDate(s)
  if (!d) return ''
  const now = new Date()
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate())
  const day = new Date(d.getFullYear(), d.getMonth(), d.getDate())
  const diff = Math.round((today.getTime() - day.getTime()) / 86400000)
  const md = `${d.getMonth() + 1}月${d.getDate()}日`
  if (isSameDay(d, now)) return withTime ? `今天 ${formatClock(d)}` : '今天'
  if (diff === 1) return withTime ? `昨天 ${formatClock(d)}` : '昨天'
  if (diff > 1 && diff < 7) return withTime ? `${diff} 天前 ${formatClock(d)}` : `${diff} 天前`
  if (d.getFullYear() === now.getFullYear()) return withTime ? `${md} ${formatClock(d)}` : md
  return withTime ? `${d.getFullYear()}年${md} ${formatClock(d)}` : `${d.getFullYear()}年${md}`
}

/** 分组标题，入参为 "2026-10-30" */
export function formatGroupTitle(dateStr: string): string {
  const [y, m, day] = dateStr.split('-').map(Number)
  if (!y || !m || !day) return dateStr
  const d = new Date(y, m - 1, day)
  const now = new Date()
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate())
  const diff = Math.round((today.getTime() - d.getTime()) / 86400000)
  if (diff === 0) return '今天'
  if (diff === 1) return '昨天'
  if (y === now.getFullYear()) return `${m}月${day}日`
  return `${y}年${m}月${day}日`
}

/** 出生日期 yyyy-MM-dd → 2020年5月1日 */
export function formatDate(s?: string | null): string {
  if (!s) return ''
  const [y, m, day] = s.split('-').map(Number)
  if (!y || !m || !day) return s
  return `${y}年${m}月${day}日`
}

export function ageText(s?: string | null): string {
  const d = parseDate(s)
  if (!d) return ''
  const now = new Date()
  let months = (now.getFullYear() - d.getFullYear()) * 12 + (now.getMonth() - d.getMonth())
  if (now.getDate() < d.getDate()) months -= 1
  if (months < 0) return '未满月'
  const years = Math.floor(months / 12)
  const rest = months % 12
  if (years === 0) return rest === 0 ? '未满月' : `${rest} 个月`
  return rest === 0 ? `${years} 岁` : `${years} 岁 ${rest} 个月`
}

export function genderText(g?: string | null): string {
  if (g === 'male') return '公'
  if (g === 'female') return '母'
  return ''
}

export function passwordValid(pw: string): boolean {
  return pw.length >= 8 && /[a-zA-Z]/.test(pw) && /\d/.test(pw)
}

/** 0 无 1 弱 2 中 3 强 */
export function passwordStrength(pw: string): 0 | 1 | 2 | 3 {
  if (!pw) return 0
  let score = 0
  if (/[a-zA-Z]/.test(pw) && /\d/.test(pw)) score += 1
  if (pw.length >= 10) score += 1
  if (/[^a-zA-Z0-9]/.test(pw) || pw.length >= 12) score += 1
  return Math.min(3, score) as 0 | 1 | 2 | 3
}

/** "yyyy-MM-dd HH:mm"（本地时间）→ RFC3339 */
export function toRFC3339(localStr: string): string {
  const d = new Date(localStr.replace(' ', 'T'))
  return d.toISOString()
}

/** RFC3339 → "yyyy-MM-dd HH:mm"（本地时间，用于日期选择器回填） */
export function fromRFC3339(s?: string | null): string {
  const d = parseDate(s)
  if (!d) return ''
  return `${dateKey(d)} ${formatClock(d)}`
}
