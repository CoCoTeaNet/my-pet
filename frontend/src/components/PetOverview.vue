<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useDialog } from 'naive-ui'
import api from '@/api'
import type { FeedGroup, FoodType, LogEntry, Pet } from '@/types'
import { useUiStore } from '@/stores/ui'
import { resolveUrl } from '@/platform'
import { ageText, formatDate, formatRelative, genderText } from '@/utils/time'
import PetFormModal from '@/components/PetFormModal.vue'

const props = defineProps<{ petId: string; embedded?: boolean }>()
const emit = defineEmits<{
  (e: 'updated'): void
  (e: 'deleted'): void
}>()

const router = useRouter()
const ui = useUiStore()
const dialog = useDialog()

const pet = ref<Pet | null>(null)
const loadError = ref('')
const loading = ref(true)
const showMenu = ref(false)
const showForm = ref(false)

const FOOD_LABEL: Record<FoodType, string> = { dry: '干粮', wet: '湿粮', treat: '零食' }

const activities = ref<{ time: string; text: string }[]>([])

const genderAgeLine = computed(() => {
  const p = pet.value
  if (!p) return ''
  const parts = [genderText(p.gender)]
  parts.push(p.birth_date ? ageText(p.birth_date) : '年龄未知')
  return parts.filter(Boolean).join(' · ')
})

const datesLine = computed(() => {
  const p = pet.value
  if (!p) return ''
  const parts: string[] = []
  if (p.birth_date) parts.push(`生日 ${formatDate(p.birth_date)}`)
  if (p.adoption_date) parts.push(`领养 ${formatDate(p.adoption_date)}`)
  return parts.join(' · ')
})

watch(() => props.petId, load, { immediate: true })

async function load() {
  const id = props.petId
  if (!id) {
    loading.value = false
    return
  }
  loading.value = true
  loadError.value = ''
  pet.value = null
  activities.value = []
  try {
    const { data } = await api.get(`/pets/${id}`)
    if (id !== props.petId) return
    pet.value = data
    if (!props.embedded) ui.setTitle(data.name)
    const [feedsRes, logsRes] = await Promise.all([
      api.get(`/pets/${id}/feeds`, { params: { page: 1, page_size: 5 } }),
      api.get(`/pets/${id}/logs`, { params: { page: 1, page_size: 5 } }),
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
    if (id !== props.petId) return
    activities.value = [...feedActs, ...logActs]
      .sort((a, b) => new Date(b.time).getTime() - new Date(a.time).getTime())
      .slice(0, 3)
  } catch (err) {
    if (id !== props.petId) return
    loadError.value = (err as Error).message
  } finally {
    if (id === props.petId) loading.value = false
  }
}

function onEdited() {
  ui.showBanner('已保存')
  load()
  emit('updated')
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
        if (props.embedded) emit('deleted')
        else router.replace('/')
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
        <img v-if="pet.avatar_url" :src="resolveUrl(pet.avatar_url)" :alt="pet.name" class="avatar avatar-80" />
        <div v-else class="avatar-fallback avatar-80">{{ pet.name.slice(0, 1) }}</div>
        <div class="hero-info">
          <div class="hero-name">{{ pet.name }}</div>
          <div class="hero-meta">{{ pet.breed || '品种未填写' }}</div>
          <div class="hero-meta">
            {{ genderAgeLine }}
            <button v-if="!pet.birth_date" type="button" class="inline-link" @click="showForm = true">设置生日</button>
          </div>
          <div v-if="datesLine" class="hero-meta">{{ datesLine }}</div>
        </div>
        <div class="more-wrap">
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
      <router-link :to="`/pet/${props.petId}/feed`" class="quick-btn">
        <span class="quick-icon">🍚</span>
        <span class="quick-label">喂食记录</span>
      </router-link>
      <router-link :to="`/pet/${props.petId}/reminder`" class="quick-btn">
        <span class="quick-icon">🔔</span>
        <span class="quick-label">健康提醒</span>
      </router-link>
      <router-link :to="`/pet/${props.petId}/log`" class="quick-btn">
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

    <button type="button" class="delete-btn anim-item" style="--i: 3" @click="confirmDelete">
      删除宠物
    </button>

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

.inline-link {
  color: var(--primary);
  font-size: 12px;
  margin-left: 6px;
  padding: 2px 0;
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

.delete-btn {
  display: block;
  width: 100%;
  margin-top: 24px;
  padding: 12px;
  border-radius: var(--radius-lg);
  background: #fff;
  border: 1px solid rgba(224, 86, 76, 0.24);
  color: var(--danger);
  font-size: 15px;
  transition: background 0.15s cubic-bezier(0.4, 0, 0.2, 1);
}

.delete-btn:active {
  background: #fff5f4;
}
</style>
