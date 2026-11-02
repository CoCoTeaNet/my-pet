<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import api from '@/api'
import type { Pet, Reminder } from '@/types'
import { useUiStore } from '@/stores/ui'
import { formatRelative } from '@/utils/time'

interface Row extends Reminder {
  petName: string
}

const router = useRouter()
const ui = useUiStore()

const loading = ref(true)
const loadError = ref('')
const rows = ref<Row[]>([])
const overdueCount = computed(() => rows.value.filter((r) => r.overdue).length)

onMounted(async () => {
  ui.setTitle('')
  try {
    const { data } = await api.get('/pets', { params: { page: 1, page_size: 50 } })
    const pets: Pet[] = data.items || []
    const groups = await Promise.all(
      pets.map(async (p) => {
        const res = await api.get(`/pets/${p.id}/reminders`)
        return ((res.data.items || []) as Reminder[])
          .filter((r) => r.enabled)
          .map((r) => ({ ...r, petName: p.name }))
      }),
    )
    rows.value = groups.flat().sort((a, b) => {
      if (a.overdue !== b.overdue) return a.overdue ? -1 : 1
      return new Date(a.next_due_at).getTime() - new Date(b.next_due_at).getTime()
    })
  } catch (err) {
    loadError.value = (err as Error).message
  } finally {
    loading.value = false
  }
})

function open(r: Row) {
  router.push(`/pet/${r.pet_id}/reminder`)
}
</script>

<template>
  <div>
    <div v-if="loading">
      <div v-for="n in 3" :key="n" class="reminder-card card" :style="{ marginTop: n > 1 ? '8px' : '0' }">
        <div class="skeleton" style="width: 40px; height: 40px; border-radius: 50%"></div>
        <div style="flex: 1">
          <div class="skeleton" style="width: 100px; height: 15px"></div>
          <div class="skeleton" style="width: 150px; height: 12px; margin-top: 8px"></div>
        </div>
      </div>
    </div>

    <div v-else-if="loadError" class="empty-state">
      <div class="empty-icon">🔔</div>
      <div class="empty-text">{{ loadError }}</div>
    </div>

    <div v-else-if="rows.length === 0" class="empty-state">
      <div class="empty-icon">🔔</div>
      <div class="empty-text">还没有健康提醒</div>
      <div class="empty-hint">进入宠物详情即可添加</div>
    </div>

    <template v-else>
      <div class="summary">
        <span>共 {{ rows.length }} 条提醒</span>
        <span v-if="overdueCount > 0" class="summary-overdue">{{ overdueCount }} 条已到期</span>
      </div>

      <div
        v-for="(r, i) in rows"
        :key="r.id"
        class="reminder-card card anim-item"
        :class="{ overdue: r.overdue }"
        :style="{ '--i': Math.min(i, 4) }"
        role="button"
        @click="open(r)"
      >
        <span class="pet-badge">{{ r.petName.slice(0, 1) }}</span>
        <div class="reminder-main">
          <div class="reminder-top">
            <span class="reminder-type">{{ r.type }}</span>
            <span class="pet-name">{{ r.petName }}</span>
          </div>
          <div class="reminder-due">
            {{ r.overdue ? '已到期 · ' : '下次：' }}{{ formatRelative(r.next_due_at, true) }}
          </div>
        </div>
        <svg class="arrow" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <polyline points="9 18 15 12 9 6" />
        </svg>
      </div>
    </template>
  </div>
</template>

<style scoped>
.summary {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  margin-bottom: 12px;
  padding: 0 4px;
  font-size: 13px;
  color: var(--text-sub);
}

.summary-overdue {
  color: var(--danger);
  font-weight: 600;
}

.reminder-card {
  display: flex;
  align-items: center;
  gap: 12px;
  cursor: pointer;
}

.reminder-card + .reminder-card {
  margin-top: 8px;
}

.pet-badge {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: var(--primary-soft);
  color: var(--primary);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 16px;
  font-weight: 600;
  flex-shrink: 0;
}

.reminder-main {
  flex: 1;
  min-width: 0;
}

.reminder-top {
  display: flex;
  align-items: baseline;
  gap: 8px;
}

.reminder-type {
  font-size: 16px;
  font-weight: 600;
}

.pet-name {
  font-size: 12px;
  color: var(--text-sub);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.reminder-due {
  margin-top: 2px;
  font-size: 13px;
  color: var(--text-mid);
}

.reminder-card.overdue .reminder-type,
.reminder-card.overdue .reminder-due {
  color: var(--danger);
}

.arrow {
  color: var(--text-sub);
  flex-shrink: 0;
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

.empty-hint {
  margin-top: 4px;
  font-size: 12px;
  color: var(--text-sub);
  opacity: 0.7;
}
</style>
