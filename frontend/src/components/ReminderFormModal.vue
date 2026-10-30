<script setup lang="ts">
import { ref, watch } from 'vue'
import { NDrawer, NDrawerContent, NInput, NInputNumber, NDatePicker, NButton } from 'naive-ui'
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
  <n-drawer :show="show" placement="bottom" height="auto" @update:show="emit('update:show', $event)">
    <n-drawer-content :title="isEdit() ? '编辑提醒' : '新建提醒'" closable body-content-style="padding: 16px 24px 24px;">
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
        <n-date-picker
          v-model:formatted-value="dueStr"
          type="datetime"
          value-format="yyyy-MM-dd HH:mm"
          clearable
          placeholder="选择日期和时间"
          style="width: 100%"
          @blur="validateDue"
        />
        <div v-if="dueError" class="field-error">{{ dueError }}</div>
      </div>

      <div class="field-item">
        <label class="field-label">备注</label>
        <n-input v-model:value="notes" type="textarea" :rows="3" placeholder="选填，如：去宠物医院东门" :maxlength="200" />
      </div>

      <div v-if="formError" class="field-error">{{ formError }}</div>

      <n-button type="primary" block :loading="submitting" class="submit-btn" @click="submit">保存</n-button>
    </n-drawer-content>
  </n-drawer>
</template>

<style scoped>
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
