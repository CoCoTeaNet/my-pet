<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { NInput, NButton, useDialog } from 'naive-ui'
import api from '@/api'
import type { LogEntry, LogType, Pet } from '@/types'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'
import { formatRelative } from '@/utils/time'
import SwipeRow from '@/components/SwipeRow.vue'

const route = useRoute()
const auth = useAuthStore()
const ui = useUiStore()
const dialog = useDialog()

const petId = route.params.id as string
const petName = ref('')
const loadError = ref('')
const loading = ref(true)
const logs = ref<LogEntry[]>([])

const LOG_TYPE: Record<LogType, string> = {
  behavior: '行为',
  health: '健康',
  other: '其他',
}
const logTypes: LogType[] = ['behavior', 'health', 'other']

const content = ref('')
const logType = ref<LogType>('behavior')
const contentError = ref('')
const submitting = ref(false)

onMounted(async () => {
  try {
    const { data } = await api.get(`/pets/${petId}`)
    petName.value = data.name
    ui.setTitle(`${data.name} · 日志`)
    await load()
  } catch (err) {
    loadError.value = (err as Error).message
  } finally {
    loading.value = false
  }
})

async function load() {
  const { data } = await api.get(`/pets/${petId}/logs`, { params: { page: 1, page_size: 50 } })
  logs.value = data.items || []
}

function validateContent() {
  contentError.value = content.value.trim() ? '' : '写点什么吧'
}

async function submit() {
  validateContent()
  if (contentError.value || submitting.value) return
  submitting.value = true
  try {
    await api.post(`/pets/${petId}/logs`, {
      content: content.value.trim(),
      log_type: logType.value,
    })
    ui.showBanner('已保存')
    content.value = ''
    await load()
  } catch (err) {
    dialog.error({ title: '保存失败', content: (err as Error).message, positiveText: '知道了' })
  } finally {
    submitting.value = false
  }
}

function confirmDelete(log: LogEntry) {
  dialog.warning({
    title: '删除日志',
    content: '确定删除这条日志吗？删除后无法恢复。',
    positiveText: '确认删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await api.delete(`/logs/${log.id}`)
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
    <div class="empty-icon">✏️</div>
    <div class="empty-text">{{ loadError }}</div>
  </div>

  <div v-else>
    <!-- 添加日志 -->
    <div class="card quick-add anim-item" style="--i: 0">
      <n-input
        v-model:value="content"
        type="textarea"
        :rows="3"
        placeholder="记下今天的小事情，比如：今天精神特别好"
        :maxlength="500"
        @blur="validateContent"
      />
      <div v-if="contentError" class="field-error">{{ contentError }}</div>
      <div class="type-row">
        <div class="chip-group">
          <button
            v-for="t in logTypes"
            :key="t"
            type="button"
            class="chip"
            :class="{ 'chip-active': logType === t }"
            @click="logType = t"
          >
            {{ LOG_TYPE[t] }}
          </button>
        </div>
        <n-button type="primary" :loading="submitting" @click="submit">记录</n-button>
      </div>
    </div>

    <!-- 时间线列表 -->
    <div v-if="logs.length === 0" class="empty-state">
      <div class="empty-icon">✏️</div>
      <div class="empty-text">还没有日志</div>
    </div>

    <div v-else class="log-list">
      <swipe-row
        v-for="(log, i) in logs"
        :key="log.id"
        :can-delete="auth.isAdmin"
        class="log-row-wrap"
        @delete="confirmDelete(log)"
      >
        <div class="log-row card anim-item" :style="{ '--i': Math.min(i + 1, 4) }">
          <div class="log-head">
            <span class="log-time">{{ formatRelative(log.logged_at, true) }}</span>
            <span class="log-tag">{{ LOG_TYPE[log.log_type] || '其他' }}</span>
          </div>
          <div class="log-content">{{ log.content }}</div>
        </div>
      </swipe-row>
    </div>
  </div>
</template>

<style scoped>
.quick-add {
  margin-bottom: 20px;
}

.type-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-top: 12px;
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

.log-row-wrap + .log-row-wrap {
  margin-top: 8px;
}

.log-row {
  padding: 12px 16px;
}

.log-head {
  display: flex;
  align-items: center;
  gap: 8px;
}

.log-time {
  font-size: 12px;
  color: var(--text-sub);
}

.log-tag {
  font-size: 12px;
  color: var(--text-mid);
  background: #f0f0f0;
  border-radius: 4px;
  padding: 1px 8px;
}

.log-content {
  margin-top: 6px;
  font-size: 14px;
  word-break: break-word;
  white-space: pre-wrap;
}
</style>
