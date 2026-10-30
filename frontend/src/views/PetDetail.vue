<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useDialog } from 'naive-ui'
import api from '@/api'
import type { FeedGroup, FoodType, LogEntry, Pet } from '@/types'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'
import { formatDate, formatRelative, genderText } from '@/utils/time'
import PetFormModal from '@/components/PetFormModal.vue'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const ui = useUiStore()
const dialog = useDialog()

const petId = route.params.id as string
const pet = ref<Pet | null>(null)
const loadError = ref('')
const loading = ref(true)
const showMenu = ref(false)
const showForm = ref(false)

const FOOD_LABEL: Record<FoodType, string> = { dry: '干粮', wet: '湿粮', treat: '零食' }

const activities = ref<{ time: string; text: string }[]>([])
const hasMore = computed(() => !!pet.value)

const genderAgeLine = computed(() => {
  const p = pet.value
  if (!p) return ''
  const parts = [genderText(p.gender), formatDate(p.birth_date)].filter(Boolean)
  return parts.join(' · ')
})

onMounted(load)

async function load() {
  loading.value = true
  try {
    const { data } = await api.get(`/pets/${petId}`)
    pet.value = data
    ui.setTitle(data.name)
    const [feedsRes, logsRes] = await Promise.all([
      api.get(`/pets/${petId}/feeds`, { params: { page: 1, page_size: 5 } }),
      api.get(`/pets/${petId}/logs`, { params: { page: 1, page_size: 5 } }),
    ])
    const feedActs = (feedsRes.data.groups || []).flatMap((g: FeedGroup) =>
      g.records.map((r) => ({
        time: r.recorded_at,
        text: `吃了${FOOD_LABEL[r.food_type]} ${r.amount}g`,
      })),
    )
    const logActs = (logsRes.data.items || []).map((l: LogEntry) => ({
      time: l.logged_at,
      text: l.content,
    }))
    activities.value = [...feedActs, ...logActs]
      .sort((a, b) => new Date(b.time).getTime() - new Date(a.time).getTime())
      .slice(0, 3)
  } catch (err) {
    loadError.value = (err as Error).message
  } finally {
    loading.value = false
  }
}

function onEdited() {
  ui.showBanner('已保存')
  load()
}

function confirmDelete() {
  showMenu.value = false
  const p = pet.value
  if (!p) return
  dialog.warning({
    title: '删除宠物',
    content: `确定删除"${p.name}"吗？将同时删除 ${p.feed_count} 条喂养记录、${p.log_count} 条日志、${p.reminder_count} 条健康提醒，删除后无法恢复。`,
    positiveText: '确认删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await api.delete(`/pets/${p.id}`)
        ui.showBanner('已删除')
        router.replace('/')
      } catch (err) {
        dialog.error({ title: '删除失败', content: (err as Error).message, positiveText: '知道了' })
      }
    },
  })
}
</script>

<template>
  <div v-if="loading"></div>

  <div v-else-if="loadError || !pet" class="empty-state">
    <div class="empty-icon">🐾</div>
    <div class="empty-text">{{ loadError || '没有找到这只宠物' }}</div>
  </div>

  <div v-else>
    <!-- 头部：毛玻璃 -->
    <div class="hero card">
      <div class="hero-main">
        <img v-if="pet.avatar_url" :src="pet.avatar_url" :alt="pet.name" class="avatar avatar-80" />
        <div v-else class="avatar-fallback avatar-80">{{ pet.name.slice(0, 1) }}</div>
        <div class="hero-info">
          <div class="hero-name">{{ pet.name }}</div>
          <div class="hero-meta">{{ pet.breed || '品种未填写' }}</div>
          <div class="hero-meta">{{ genderAgeLine || '信息未填写' }}</div>
        </div>
        <div v-if="auth.isAdmin" class="more-wrap">
          <button class="icon-btn" aria-label="更多" @click.stop="showMenu = !showMenu">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor">
              <circle cx="12" cy="5" r="1.8" />
              <circle cx="12" cy="12" r="1.8" />
              <circle cx="12" cy="19" r="1.8" />
            </svg>
          </button>
          <div v-if="showMenu" class="menu-overlay" @click="showMenu = false"></div>
          <div v-if="showMenu" class="menu card">
            <button class="menu-item" @click="showMenu = false; showForm = true">编辑宠物</button>
            <button class="menu-item danger" @click="confirmDelete">删除宠物</button>
          </div>
        </div>
      </div>
      <div v-if="pet.notes" class="hero-notes">{{ pet.notes }}</div>
    </div>

    <!-- 快捷操作 -->
    <div class="quick card anim-item" style="--i: 1">
      <router-link :to="`/pet/${petId}/feed`" class="quick-btn">
        <span class="quick-icon">🍚</span>
        <span class="quick-label">喂食记录</span>
      </router-link>
      <router-link :to="`/pet/${petId}/reminder`" class="quick-btn">
        <span class="quick-icon">🔔</span>
        <span class="quick-label">健康提醒</span>
      </router-link>
      <router-link :to="`/pet/${petId}/log`" class="quick-btn">
        <span class="quick-icon">✏️</span>
        <span class="quick-label">日志</span>
      </router-link>
    </div>

    <!-- 最近动态 -->
    <div class="activity card anim-item" style="--i: 2">
      <div class="section-title">{{ pet.name }}的最近动态</div>
      <div v-if="activities.length === 0" class="activity-empty">最近没有新动态</div>
      <div v-for="(a, i) in activities" :key="i" class="activity-row">
        <span class="activity-time">{{ formatRelative(a.time, true) }}</span>
        <span class="activity-text">{{ a.text }}</span>
      </div>
    </div>

    <pet-form-modal v-model:show="showForm" :pet="pet" @saved="onEdited" />
  </div>
</template>

<style scoped>
.empty-state {
  padding: 80px 0 40px;
  text-align: center;
}

.empty-icon {
  font-size: 48px;
  filter: grayscale(1);
  opacity: 0.4;
}

.empty-text {
  margin-top: 16px;
  font-size: 14px;
  color: var(--text-sub);
}

.hero {
  backdrop-filter: blur(10px);
  -webkit-backdrop-filter: blur(10px);
  background: rgba(255, 255, 255, 0.72);
}

.hero-main {
  display: flex;
  align-items: center;
  gap: 16px;
}

.avatar-80 {
  width: 80px;
  height: 80px;
  font-size: 28px;
}

.hero-info {
  flex: 1;
  min-width: 0;
}

.hero-name {
  font-size: 20px;
  font-weight: 600;
}

.hero-meta {
  font-size: 14px;
  color: var(--text-mid);
  margin-top: 2px;
}

.hero-notes {
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid #f0f0f0;
  font-size: 14px;
  color: var(--text-mid);
  word-break: break-word;
}

.more-wrap {
  position: relative;
  flex-shrink: 0;
}

.icon-btn {
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-sm);
  color: var(--text-sub);
}

.icon-btn:hover {
  background: #f5f5f5;
  color: var(--text);
}

.menu-overlay {
  position: fixed;
  inset: 0;
  z-index: 10;
}

.menu {
  position: absolute;
  right: 0;
  top: 42px;
  z-index: 11;
  padding: 6px;
  min-width: 120px;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.08);
}

.menu-item {
  display: block;
  width: 100%;
  text-align: left;
  font-size: 14px;
  padding: 10px 12px;
  border-radius: var(--radius-sm);
}

.menu-item:hover {
  background: #f7f7f7;
}

.menu-item.danger {
  color: var(--danger);
}

.quick {
  margin-top: 16px;
  display: flex;
}

.quick-btn {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  padding: 8px 0;
  border-radius: var(--radius-sm);
}

.quick-btn:hover {
  background: #fafafa;
}

.quick-btn + .quick-btn {
  border-left: 1px solid #f5f5f5;
}

.quick-icon {
  font-size: 24px;
  line-height: 1;
}

.quick-label {
  font-size: 14px;
  color: var(--text-mid);
}

.activity {
  margin-top: 16px;
}

.section-title {
  font-size: 16px;
  font-weight: 600;
  margin-bottom: 12px;
}

.activity-empty {
  font-size: 14px;
  color: var(--text-sub);
  padding: 8px 0;
}

.activity-row {
  display: flex;
  gap: 12px;
  align-items: baseline;
  padding: 8px 0;
}

.activity-row + .activity-row {
  border-top: 1px solid #f7f7f7;
}

.activity-time {
  font-size: 12px;
  color: var(--text-sub);
  flex-shrink: 0;
  width: 96px;
}

.activity-text {
  font-size: 14px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
