<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import api from '@/api'
import type { Pet } from '@/types'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'
import { resolveUrl } from '@/platform'
import { ageText } from '@/utils/time'
import PetFormModal from '@/components/PetFormModal.vue'

const router = useRouter()
const auth = useAuthStore()
const ui = useUiStore()

const loading = ref(true)
const loadError = ref('')
const pets = ref<Pet[]>([])
const showForm = ref(false)

onMounted(async () => {
  ui.setTitle('')
  try {
    const { data } = await api.get('/pets', { params: { page: 1, page_size: 50 } })
    pets.value = data.items || []
  } catch (err) {
    loadError.value = (err as Error).message
  } finally {
    loading.value = false
  }
})

function onSaved(pet: Pet) {
  ui.showBanner('已保存')
  router.push(`/pet/${pet.id}`)
}
</script>

<template>
  <div>
    <!-- 骨架屏 -->
    <div v-if="loading" class="pet-card card" v-for="n in 3" :key="n">
      <div class="skeleton" style="width: 48px; height: 48px; border-radius: 50%"></div>
      <div style="flex: 1">
        <div class="skeleton" style="width: 120px; height: 16px"></div>
        <div class="skeleton" style="width: 180px; height: 12px; margin-top: 10px"></div>
        <div class="skeleton" style="width: 90px; height: 12px; margin-top: 8px"></div>
      </div>
    </div>

    <div v-else-if="loadError" class="empty-state">
      <div class="empty-icon">🐾</div>
      <div class="empty-text">{{ loadError }}</div>
    </div>

    <div v-else-if="pets.length === 0" class="empty-state">
      <div class="empty-icon">🐾</div>
      <div class="empty-text">还没有宠物哦，点下方添加吧</div>
    </div>

    <template v-else>
      <div
        v-for="(p, i) in pets"
        :key="p.id"
        class="pet-card card anim-item"
        :style="{ '--i': Math.min(i, 4) }"
        role="button"
        @click="router.push(`/pet/${p.id}`)"
      >
        <img v-if="p.avatar_url" :src="resolveUrl(p.avatar_url)" :alt="p.name" class="avatar avatar-48" />
        <div v-else class="avatar-fallback avatar-48">{{ p.name.slice(0, 1) }}</div>
        <div class="pet-info">
          <div class="pet-name">{{ p.name }}</div>
          <div class="pet-breed">{{ p.breed || '品种未填写' }}</div>
          <div class="pet-age">{{ ageText(p.birth_date) || '年龄未知' }}</div>
        </div>
        <svg class="pet-arrow" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <polyline points="9 18 15 12 9 6" />
        </svg>
      </div>
    </template>

    <button v-if="auth.isAdmin" class="fab" type="button" @click="showForm = true">＋ 添加宠物</button>

    <pet-form-modal v-model:show="showForm" :pet="null" @saved="onSaved" />
  </div>
</template>

<style scoped>
.pet-card {
  display: flex;
  align-items: center;
  gap: 12px;
  cursor: pointer;
}

.pet-card + .pet-card {
  margin-top: 12px;
}

.pet-card:hover {
  background: #fffdf9;
}

.avatar-48 {
  width: 48px;
  height: 48px;
  font-size: 18px;
}

.pet-info {
  flex: 1;
  min-width: 0;
}

.pet-name {
  font-size: 16px;
  font-weight: 600;
}

.pet-breed {
  font-size: 14px;
  color: var(--text-mid);
  margin-top: 2px;
}

.pet-age {
  font-size: 12px;
  color: var(--text-sub);
  margin-top: 2px;
}

.pet-arrow {
  color: #cccccc;
  flex-shrink: 0;
}

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
</style>
