<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { NSlider, NButton, useDialog } from 'naive-ui'
import api from '@/api'
import type { FeedGroup, FeedRecord, FoodType, Pet } from '@/types'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'
import { formatGroupTitle, formatTimeOnly } from '@/utils/time'
import SwipeRow from '@/components/SwipeRow.vue'

const route = useRoute()
const auth = useAuthStore()
const ui = useUiStore()
const dialog = useDialog()

const petId = route.params.id as string
const petName = ref('')
const loadError = ref('')
const loading = ref(true)
const groups = ref<FeedGroup[]>([])

const FOOD: Record<FoodType, { label: string; icon: string }> = {
  dry: { label: '干粮', icon: '🍚' },
  wet: { label: '湿粮', icon: '🥫' },
  treat: { label: '零食', icon: '🍖' },
}
const foodTypes: FoodType[] = ['dry', 'wet', 'treat']

const foodType = ref<FoodType>('dry')
const amount = ref(50)
const submitting = ref(false)

onMounted(async () => {
  try {
    const { data } = await api.get(`/pets/${petId}`)
    petName.value = data.name
    ui.setTitle(`${data.name} · 喂养记录`)
    await load()
  } catch (err) {
    loadError.value = (err as Error).message
  } finally {
    loading.value = false
  }
})

async function load() {
  const { data } = await api.get(`/pets/${petId}/feeds`, { params: { page: 1, page_size: 100 } })
  groups.value = data.groups || []
}

async function submit() {
  if (submitting.value) return
  submitting.value = true
  try {
    await api.post(`/pets/${petId}/feeds`, {
      food_type: foodType.value,
      amount: amount.value,
      notes: '',
    })
    ui.showBanner('已保存')
    amount.value = 50
    await load()
  } catch (err) {
    dialog.error({ title: '保存失败', content: (err as Error).message, positiveText: '知道了' })
  } finally {
    submitting.value = false
  }
}

function confirmDelete(r: FeedRecord) {
  dialog.warning({
    title: '删除喂养信息',
    content: `确定删除${petName.value || '宠物'}的这条喂养信息吗？删除后无法恢复。`,
    positiveText: '确认删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await api.delete(`/feeds/${r.id}`)
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
    <div class="empty-icon">🍚</div>
    <div class="empty-text">{{ loadError }}</div>
  </div>

  <div v-else>
    <!-- 快速添加区 -->
    <div class="card quick-add anim-item" style="--i: 0">
      <div class="chip-group">
        <button
          v-for="t in foodTypes"
          :key="t"
          type="button"
          class="chip"
          :class="{ 'chip-active': foodType === t }"
          @click="foodType = t"
        >
          {{ FOOD[t].label }}
        </button>
      </div>
      <div class="amount-row">
        <n-slider v-model:value="amount" :min="0" :max="200" :step="5" class="amount-slider" />
        <span class="amount-value">{{ amount }}g</span>
      </div>
      <n-button type="primary" block :loading="submitting" @click="submit">记录</n-button>
    </div>

    <!-- 历史列表 -->
    <div v-if="groups.length === 0" class="empty-state">
      <div class="empty-icon">🍚</div>
      <div class="empty-text">还没有喂养记录</div>
    </div>

    <div v-for="(g, gi) in groups" :key="g.date" class="feed-group">
      <div class="group-title">{{ formatGroupTitle(g.date) }}</div>
      <swipe-row
        v-for="(r, ri) in g.records"
        :key="r.id"
        :can-delete="auth.isAdmin"
        class="feed-row-wrap"
        @delete="confirmDelete(r)"
      >
        <div class="feed-row card anim-item" :style="{ '--i': Math.min(ri + 1, 4) }">
          <span class="feed-time">{{ formatTimeOnly(r.recorded_at) }}</span>
          <span class="feed-icon">{{ FOOD[r.food_type]?.icon || '🍚' }}</span>
          <span class="feed-text">{{ FOOD[r.food_type]?.label || '食物' }} {{ r.amount }}g</span>
        </div>
      </swipe-row>
      <div v-if="gi < groups.length - 1" class="group-gap"></div>
    </div>
  </div>
</template>

<style scoped>
.quick-add {
  margin-bottom: 20px;
}

.amount-row {
  display: flex;
  align-items: center;
  gap: 16px;
  margin: 20px 0;
}

.amount-slider {
  flex: 1;
}

.amount-value {
  font-size: 16px;
  font-weight: 600;
  color: var(--primary);
  min-width: 48px;
  text-align: right;
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

.group-title {
  font-size: 14px;
  color: var(--text-sub);
  margin: 0 4px 8px;
}

.group-gap {
  height: 20px;
}

.feed-row-wrap + .feed-row-wrap {
  margin-top: 8px;
}

.feed-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
}

.feed-time {
  font-size: 12px;
  color: var(--text-sub);
  width: 44px;
  flex-shrink: 0;
}

.feed-icon {
  font-size: 18px;
  line-height: 1;
}

.feed-text {
  font-size: 14px;
}
</style>
