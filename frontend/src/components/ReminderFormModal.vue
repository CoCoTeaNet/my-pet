<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { NInput, NInputNumber, NButton } from 'naive-ui'
import DateField from '@/components/DateField.vue'
import api from '@/api'
import type { Reminder } from '@/types'
import { useUiStore } from '@/stores/ui'
import { dateKey, fromRFC3339, toRFC3339 } from '@/utils/time'

const props = defineProps<{ show: boolean; petId: string; reminder: Reminder | null }>()
const emit = defineEmits<{
  (e: 'update:show', v: boolean): void
  (e: 'saved'): void
}>()

const ui = useUiStore()
const isEdit = () => !!props.reminder

const TYPES = ['驱虫', '疫苗', '洗澡', '剪指甲', '其他'] as const
const CYCLES = [
  { label: '每月', days: 30 },
  { label: '每季', days: 90 },
  { label: '每年', days: 365 },
]

const type = ref<string>('驱虫')
const intervalDays = ref(30)
const customDays = ref<number | null>(null)
const isCustom = ref(false)
const dueStr = ref('')
const notes = ref('')
const dueError = ref('')
const formError = ref('')
const submitting = ref(false)

function defaultDue(): string {
  const d = new Date()
  d.setDate(d.getDate() + 1)
  d.setHours(9, 0, 0, 0)
  return `${dateKey(d)} 09:00`
}

watch(
  () => props.show,
  (v) => {
    if (!v) return
    dueError.value = ''
    formError.value = ''
    submitting.value = false
    if (props.reminder) {
      const r = props.reminder
      type.value = r.type
      const c = CYCLES.find((x) => x.days === r.interval_days)
      if (c) {
        isCustom.value = false
        intervalDays.value = c.days
      } else {
        isCustom.value = true
        customDays.value = r.interval_days
      }
      dueStr.value = fromRFC3339(r.next_due_at) || defaultDue()
      notes.value = r.notes || ''
    } else {
      type.value = '驱虫'
      isCustom.value = false
      intervalDays.value = 30
      customDays.value = null
      dueStr.value = defaultDue()
      notes.value = ''
    }
  },
)

function pickCycle(days: number) {
  isCustom.value = false
  intervalDays.value = days
}

function pickCustom() {
  isCustom.value = true
  if (!customDays.value) customDays.value = 7
}

function validateDue() {
  dueError.value = dueStr.value ? '' : '请选择首次执行日期'
}

function close() {
  emit('update:show', false)
}

const dueValue = computed({
  get: () => dueStr.value.replace(' ', 'T'),
  set: (v: string) => {
    dueStr.value = v.replace('T', ' ')
  },
})

async function submit() {
  validateDue()
  const days = isCustom.value ? customDays.value : intervalDays.value
  if (dueError.value || submitting.value) return
  if (!days || days <= 0) {
    formError.value = '请输入有效的周期天数'
    return
  }
  submitting.value = true
  formError.value = ''
  try {
    const body = {
      type: type.value,
      interval_days: days,
      next_due_at: toRFC3339(dueStr.value),
      notes: notes.value.trim(),
    }
    if (isEdit() && props.reminder) {
      await api.put(`/reminders/${props.reminder.id}`, body)
    } else {
      await api.post(`/pets/${props.petId}/reminders`, body)
    }
    emit('update:show', false)
    emit('saved')
    ui.showBanner('已保存')
  } catch (err) {
    formError.value = (err as Error).message
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <teleport to="body">
    <div v-if="show" class="sheet-mask" @click.self="close">
      <div class="sheet" role="dialog" aria-modal="true">
        <div class="sheet-header">
          <span class="sheet-title">{{ isEdit() ? '编辑提醒' : '新建提醒' }}</span>
          <button type="button" class="sheet-close" aria-label="关闭" @click="close">
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
              <line x1="18" y1="6" x2="6" y2="18" />
              <line x1="6" y1="6" x2="18" y2="18" />
            </svg>
          </button>
        </div>
        <div class="sheet-body">
      <div class="field-item">
        <label class="field-label">类型</label>
        <div class="chip-group">
          <button
            v-for="t in TYPES"
            :key="t"
            type="button"
            class="chip"
            :class="{ 'chip-active': type === t }"
            @click="type = t"
          >
            {{ t }}
          </button>
        </div>
      </div>

      <div class="field-item">
        <label class="field-label">重复周期</label>
        <div class="chip-group">
          <button
            v-for="c in CYCLES"
            :key="c.days"
            type="button"
            class="chip"
            :class="{ 'chip-active': !isCustom && intervalDays === c.days }"
            @click="pickCycle(c.days)"
          >
            {{ c.label }}
          </button>
          <button type="button" class="chip" :class="{ 'chip-active': isCustom }" @click="pickCustom">自定义</button>
        </div>
        <div v-if="isCustom" class="custom-days">
          <span>每</span>
          <n-input-number v-model:value="customDays" :min="1" :max="3650" style="width: 110px" />
          <span>天一次</span>
        </div>
      </div>

      <div class="field-item">
        <label class="field-label">首次执行日期</label>
        <date-field v-model="dueValue" type="datetime-local" />
        <div v-if="dueError" class="field-error">{{ dueError }}</div>
      </div>

      <div class="field-item">
        <label class="field-label">备注</label>
        <n-input v-model:value="notes" type="textarea" :rows="3" placeholder="选填，如：去宠物医院东门" :maxlength="200" />
      </div>

      <div v-if="formError" class="field-error">{{ formError }}</div>

      <n-button type="primary" block :loading="submitting" class="submit-btn" @click="submit">保存</n-button>
        </div>
      </div>
    </div>
  </teleport>
</template>

<style scoped>
.sheet-mask {
  position: fixed;
  inset: 0;
  z-index: 3000;
  display: flex;
  align-items: flex-end;
  background: rgba(0, 0, 0, 0.45);
  animation: sheet-fade 0.2s ease;
}

.sheet {
  width: 100%;
  max-height: 90vh;
  display: flex;
  flex-direction: column;
  background: #fff;
  border-radius: 16px 16px 0 0;
  padding-bottom: env(safe-area-inset-bottom);
  animation: sheet-up 0.24s cubic-bezier(0.22, 0.61, 0.36, 1);
}

.sheet-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px 4px;
}

.sheet-title {
  font-size: 16px;
  font-weight: 600;
}

.sheet-close {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border-radius: 50%;
  color: var(--text-mid);
}

.sheet-close:hover {
  background: #f5f5f5;
}

.sheet-body {
  padding: 8px 20px 20px;
  overflow-y: auto;
  -webkit-overflow-scrolling: touch;
}

@keyframes sheet-fade {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}

@media (min-width: 640px) {
  .sheet-mask {
    align-items: center;
    justify-content: center;
    padding: 24px;
  }

  .sheet {
    max-width: 520px;
    border-radius: 12px;
  }
}

@keyframes sheet-up {
  from {
    transform: translateY(24px);
    opacity: 0;
  }
  to {
    transform: none;
    opacity: 1;
  }
}

.custom-days {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 12px;
  font-size: 14px;
  color: var(--text-mid);
}

.submit-btn {
  margin-top: 20px;
}
</style>
