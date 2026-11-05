<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import api from '@/api'
import type { Pet } from '@/types'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'
import { resolveUrl } from '@/platform'
import PetFormModal from '@/components/PetFormModal.vue'
import PetOverview from '@/components/PetOverview.vue'

const auth = useAuthStore()
const ui = useUiStore()

const loading = ref(true)
const loadError = ref('')
const pets = ref<Pet[]>([])
const selectedId = ref('')
const showForm = ref(false)

const showSwitcher = computed(() => pets.value.length > 1)

onMounted(async () => {
  ui.setTitle('')
  try {
    const { data } = await api.get('/auth/me')
    auth.setAuth(auth.token, data)
  } catch {
    // 刷新用户信息失败不影响页面
  }
  await loadPets()
})

async function loadPets(keepId?: string) {
  loading.value = pets.value.length === 0
  loadError.value = ''
  try {
    const { data } = await api.get('/pets', { params: { page: 1, page_size: 100 } })
    pets.value = data.items || []
  } catch (err) {
    loadError.value = (err as Error).message
    return
  } finally {
    loading.value = false
  }

  if (pets.value.length === 0) {
    selectedId.value = ''
    return
  }
  const featured = auth.user?.featured_pet_id || ''
  const keep = keepId && pets.value.some((p) => p.id === keepId) ? keepId : ''
  selectedId.value = keep || pets.value.find((p) => p.id === featured)?.id || pets.value[0].id
  if (selectedId.value !== featured) persistFeatured(selectedId.value)
}

function selectPet(p: Pet) {
  if (p.id === selectedId.value) return
  selectedId.value = p.id
  persistFeatured(p.id)
}

async function persistFeatured(id: string) {
  auth.setFeaturedPet(id)
  try {
    await api.put('/auth/featured-pet', { pet_id: id })
  } catch {
    // 本地选择已生效，下次进入再同步
  }
}

function onSaved(pet: Pet) {
  ui.showBanner('已保存')
  if (!pets.value.some((p) => p.id === pet.id)) pets.value.push(pet)
  selectedId.value = pet.id
  persistFeatured(pet.id)
  loadPets(pet.id)
}

function onUpdated() {
  loadPets(selectedId.value)
}

function onDeleted() {
  auth.setFeaturedPet('')
  loadPets()
}
</script>

<template>
  <div>
    <div v-if="loading" class="home-skeleton">
      <div class="skeleton" style="height: 112px; border-radius: var(--radius-lg)"></div>
      <div class="skeleton" style="height: 72px; margin-top: 16px; border-radius: var(--radius-lg)"></div>
      <div class="skeleton" style="height: 120px; margin-top: 16px; border-radius: var(--radius-lg)"></div>
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
      <div v-if="showSwitcher" class="pet-switcher card anim-item">
        <button
          v-for="p in pets"
          :key="p.id"
          type="button"
          class="pet-chip"
          :class="{ active: p.id === selectedId }"
          @click="selectPet(p)"
        >
          <img v-if="p.avatar_url" :src="resolveUrl(p.avatar_url)" :alt="p.name" class="avatar avatar-32" />
          <span v-else class="avatar-fallback avatar-32">{{ p.name.slice(0, 1) }}</span>
          <span class="pet-chip-name">{{ p.name }}</span>
        </button>
      </div>

      <pet-overview
        :pet-id="selectedId"
        embedded
        @updated="onUpdated"
        @deleted="onDeleted"
      />
    </template>

    <button class="fab" type="button" @click="showForm = true">＋ 添加宠物</button>

    <pet-form-modal v-model:show="showForm" :pet="null" @saved="onSaved" />
  </div>
</template>

<style scoped>
.pet-switcher {
  display: flex;
  gap: 8px;
  padding: 8px;
  margin-bottom: 12px;
  overflow-x: auto;
  scrollbar-width: none;
}

.pet-switcher::-webkit-scrollbar {
  display: none;
}

.pet-chip {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 5px 12px 5px 5px;
  border-radius: 999px;
  background: #f5f5f5;
  color: var(--text-mid);
  font-size: 14px;
  transition: background 0.15s cubic-bezier(0.4, 0, 0.2, 1),
    color 0.15s cubic-bezier(0.4, 0, 0.2, 1),
    box-shadow 0.15s cubic-bezier(0.4, 0, 0.2, 1);
}

.pet-chip.active {
  background: var(--primary-soft);
  color: var(--primary);
  font-weight: 600;
  box-shadow: inset 0 0 0 1.5px var(--primary);
}

.avatar-32 {
  width: 32px;
  height: 32px;
  font-size: 13px;
}

.pet-chip-name {
  max-width: 96px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
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
