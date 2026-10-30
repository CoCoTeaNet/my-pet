export type UserRole = 'admin' | 'member'

export interface UserInfo {
  id: string
  username: string
  role: UserRole
  created_at?: string
}

export interface Pet {
  id: string
  name: string
  breed: string
  gender: '' | 'male' | 'female'
  birth_date: string
  weight: number | null
  avatar_url: string
  notes: string
  feed_count: number
  log_count: number
  reminder_count: number
  created_at: string
}

export type FoodType = 'dry' | 'wet' | 'treat'

export interface FeedRecord {
  id: string
  pet_id: string
  food_type: FoodType
  amount: number
  recorded_at: string
  notes: string
}

export interface FeedGroup {
  date: string
  records: FeedRecord[]
}

export type ReminderType = '驱虫' | '疫苗' | '洗澡' | '剪指甲' | '其他'

export interface Reminder {
  id: string
  pet_id: string
  type: string
  interval_days: number
  next_due_at: string
  last_triggered_at: string
  notes: string
  enabled: boolean
  overdue: boolean
  created_at: string
}

export type LogType = 'behavior' | 'health' | 'other'

export interface LogEntry {
  id: string
  pet_id: string
  content: string
  log_type: LogType
  logged_at: string
  created_at: string
}

export interface AppSettings {
  webhook_url: string
  reminder_interval_minutes: number
  backup_path: string
  last_backup_at: string
}

export interface NotificationLog {
  time: string
  message: string
  success: boolean
}
