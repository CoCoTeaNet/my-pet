<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { NCheckbox, useDialog } from 'naive-ui'
import api from '@/api'
import type { Reminder } from '@/types'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'
import { syncNotifications } from '@/services/notifications'
import { dateKey, formatRelative, formatTimeOnly } from '@/utils/time'
import ReminderFormModal from '@/components/ReminderFormModal.vue'

const route = useRoute()
const auth = useAuthStore()
const ui = useUiStore()
const dialog = useDialog()

const petId = route.params.id as string
const loading = ref(true)
const loadError = ref('')
const items = ref<Reminder[]>([])

const showForm = ref(false)
const editing = ref<Reminder | null>(null)

const view = ref(new Date())
const selectedDate = ref('')

const remindersByDate = computed(() => {
  const map = new Map<string, Reminder[]>()
  for (const r of items.value) {
    const d = new Date(r.next_due_at)
    if (Number.isNaN(d.getTime())) continue
    const k = dateKey(d)
    if (!map.has(k)) map.set(k, [])
    map.get(k)!.push(r)
  }
  return map
})

const viewYear = computed(() => view.value.getFullYear())
const viewMonth = computed(() => view.value.getMonth())
const weekLabels = ['一', '二', '三', '四', '五', '六', '日']

const calendarCells = computed(() => {
  const y = viewYear.value
  const m = viewMonth.value
  const first = new Date(y, m, 1)
  const startOffset = (first.getDay() + 6) % 7
  const daysInMonth = new Date(y, m + 1, 0).getDate()
  const cells: { day: number; key: string }[] = []
  for (let d = 1; d <= daysInMonth; d++) {
    cells.push({ day: d, key: dateKey(new Date(y, m, d)) })
  }
  return { blanks: Array.from({ length: startOffset }), cells }
})

function prevMonth() {
  view.value = new Date(viewYear.value, viewMonth.value - 1, 1)
}

function nextMonth() {
  view.value = new Date(viewYear.value, viewMonth.value + 1, 1)
}

onMounted(async () => {
  try {
    const { data } = await api.get(`/pets/${petId}`)
    ui.setTitle(`${data.name} · 健康提醒`)
    await load()
  } catch (err) {
    loadError.value = (err as Error).message
  } finally {
    loading.value = false
  }
})

async function load() {
  const { data } = await api.get(`/pets/${petId}/reminders`)
  items.value = data.items || []
  syncNotifications().catch(() => {})
}

function openCreate() {
  editing.value = null
  showForm.value = true
}

function openEdit(r: Reminder) {
  editing.value = r
  showForm.value = true
}

async function complete(r: Reminder) {
  try {
    await api.post(`/reminders/${r.id}/complete`)
    ui.showBanner('已完成')
    await load()
  } catch (err) {
    dialog.error({ title: '操作失败', content: (err as Error).message, positiveText: '知道了' })
  }
}

function confirmDelete(r: Reminder) {
  dialog.warning({
    title: '删除提醒',
    content: `确定删除"${r.type}"这条提醒吗？删除后不再收到通知，无法恢复。`,
    positiveText: '确认删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await api.delete(`/reminders/${r.id}`)
        ui.showBanner('已删除')
        await load()
      } catch (err) {
        dialog.error({ title: '删除失败', content: (err as Error).message, positiveText: '知道了' })
      }
    },
  })
}
</script>

<template>
  <div v-if="loading"></div>

  <div v-else-if="loadError" class="empty-state">
    <div class="empty-icon">🔔</div>
    <div class="empty-text">{{ loadError }}</div>
  </div>

  <div v-else>
    <!-- 当月日历 -->
    <div class="card calendar anim-item" style="--i: 0">
      <div class="cal-header">
        <button class="cal-nav" aria-label="上个月" @click="prevMonth">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="15 18 9 12 15 6" /></svg>
        </button>
        <span class="cal-title">{{ viewYear }} 年 {{ viewMonth + 1 }} 月</span>
        <button class="cal-nav" aria-label="下个月" @click="nextMonth">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="9 18 15 12 9 6" /></svg>
        </button>
      </div>
      <div class="cal-week">
        <span v-for="w in weekLabels" :key="w" class="cal-week-label">{{ w }}</span>
      </div>
      <div class="cal-grid">
        <span v-for="(b, i) in calendarCells.blanks" :key="'b' + i" class="cal-cell cal-blank"></span>
        <button
          v-for="c in calendarCells.cells"
          :key="c.key"
          type="button"
          class="cal-cell"
          :class="{ 'cal-has': remindersByDate.has(c.key), 'cal-selected': selectedDate === c.key }"
          @click="remindersByDate.has(c.key) && (selectedDate = c.key)"
        >
          {{ c.day }}
          <span v-if="remindersByDate.has(c.key)" class="cal-dot"></span>
        </button>
      </div>
    </div>

    <!-- 提醒列表 -->
    <div v-if="items.length === 0" class="empty-state">
      <div class="empty-icon">🔔</div>
      <div class="empty-text">还没有健康提醒</div>
    </div>

    <div v-else class="reminder-list">
      <div
        v-for="(r, i) in items"
        :key="r.id"
        class="reminder-row card anim-item"
        :class="{ overdue: r.overdue }"
        :style="{ '--i': Math.min(i + 1, 4) }"
      >
        <div class="reminder-main">
          <div class="reminder-top">
            <span class="reminder-type">{{ r.type }}</span>
            <span class="reminder-due">下次：{{ formatRelative(r.next_due_at, true) }}</span>
          </div>
          <div v-if="r.notes" class="reminder-notes">{{ r.notes }}</div>
        </div>
        <div class="reminder-side">
          <n-checkbox @update:checked="() => complete(r)">完成</n-checkbox>
          <div v-if="auth.isAdmin" class="reminder-ops">
            <button class="text-btn" @click="openEdit(r)">编辑</button>
            <button class="text-btn danger" @click="confirmDelete(r)">删除</button>
          </div>
        </div>
      </div>
    </div>

    <button class="fab" type="button" @click="openCreate">＋ 新建提醒</button>

    <!-- 点击日期的浮层 -->
    <div v-if="selectedDate" class="day-overlay" @click.self="selectedDate = ''">
      <div class="day-pop card">
        <div class="day-pop-header">
          <span class="day-pop-title">{{ formatRelative(selectedDate + 'T00:00:00') }}的提醒</span>
          <button class="day-pop-close" aria-label="关闭" @click="selectedDate = ''">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><line x1="18" y1="6" x2="6" y2="18" /><line x1="6" y1="6" x2="18" y2="18" /></svg>
          </button>
        </div>
        <div v-for="r in remindersByDate.get(selectedDate) || []" :key="r.id" class="day-pop-item">
          <span class="reminder-type">{{ r.type }}</span>
          <span class="day-pop-time">{{ formatTimeOnly(r.next_due_at) }}</span>
          <span v-if="r.notes" class="day-pop-notes">{{ r.notes }}</span>
        </div>
      </div>
    </div>

    <reminder-form-modal v-model:show="showForm" :pet-id="petId" :reminder="editing" @saved="load" />
  </div>
</template>

<style scoped>
.calendar {
  padding: 16px;
}

.cal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}

.cal-title {
  font-size: 16px;
  font-weight: 600;
}

.cal-nav {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-sm);
  color: var(--text-mid);
}

.cal-nav:hover {
  background: #f5f5f5;
}

.cal-week,
.cal-grid {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  gap: 2px;
}

.cal-week-label {
  text-align: center;
  font-size: 12px;
  color: var(--text-sub);
  padding: 4px 0;
}

.cal-cell {
  position: relative;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  border-radius: var(--radius-sm);
  color: var(--text);
}

.cal-cell:hover {
  background: #fafafa;
}

.cal-cell.cal-blank {
  pointer-events: none;
}

.cal-cell.cal-has {
  font-weight: 600;
}

.cal-cell.cal-selected {
  background: var(--primary-soft);
}

.cal-dot {
  position: absolute;
  bottom: 5px;
  left: 50%;
  transform: translateX(-50%);
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: var(--primary);
}

.empty-state {
  padding: 60px 0 20px;
  text-align: center;
}

.empty-icon {
  font-size: 40px;
  filter: grayscale(1);
  opacity: 0.4;
}

.empty-text {
  margin-top: 12px;
  font-size: 14px;
  color: var(--text-sub);
}

.reminder-list {
  margin-top: 20px;
}

.reminder-row {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}

.reminder-row + .reminder-row {
  margin-top: 8px;
}

.reminder-main {
  flex: 1;
  min-width: 0;
}

.reminder-top {
  display: flex;
  align-items: baseline;
  gap: 10px;
  flex-wrap: wrap;
}

.reminder-type {
  font-size: 16px;
  font-weight: 600;
}

.reminder-due {
  font-size: 14px;
  color: var(--text-mid);
}

.reminder-notes {
  margin-top: 4px;
  font-size: 12px;
  color: var(--text-sub);
  word-break: break-word;
}

.reminder-row.overdue .reminder-type,
.reminder-row.overdue .reminder-due,
.reminder-row.overdue .reminder-notes {
  color: var(--danger);
}

.reminder-side {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 4px;
  flex-shrink: 0;
}

.reminder-ops {
  display: flex;
  gap: 2px;
}

.day-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.35);
  z-index: 40;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
}

.day-pop {
  width: 100%;
  max-width: 360px;
  max-height: 60vh;
  overflow: auto;
}

.day-pop-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}

.day-pop-title {
  font-size: 16px;
  font-weight: 600;
}

.day-pop-close {
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 6px;
  color: var(--text-sub);
}

.day-pop-item {
  display: flex;
  align-items: baseline;
  gap: 10px;
  padding: 10px 0;
  border-top: 1px solid #f5f5f5;
}

.day-pop-time {
  font-size: 12px;
  color: var(--text-sub);
}

.day-pop-notes {
  font-size: 12px;
  color: var(--text-sub);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
