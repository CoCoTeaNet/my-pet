<script setup lang="ts">
import { computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useUiStore } from '@/stores/ui'

const route = useRoute()
const router = useRouter()
const ui = useUiStore()

watch(() => route.path, p => {
  if (!p.startsWith('/pet/')) ui.pageTitle = ''
})

const showBack = computed(() => route.path !== '/')
const title = computed(() => ui.pageTitle || (route.meta.title as string) || '我的宠物')

const reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches
const pageFadeDuration = reducedMotion ? 0 : { enter: 300, leave: 200 }
const bannerDuration = reducedMotion ? 0 : 300

function goBack() {
  if (window.history.length > 1) router.back()
  else router.replace('/')
}
</script>

<template>
  <div class="layout">
    <header class="topbar">
      <div class="topbar-inner">
        <div class="topbar-left">
          <button v-if="showBack" class="icon-btn" aria-label="返回" @click="goBack">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <polyline points="15 18 9 12 15 6" />
            </svg>
          </button>
          <span class="topbar-title">{{ title }}</span>
        </div>
        <router-link v-if="route.path !== '/settings'" to="/settings" class="icon-btn" aria-label="设置">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="12" cy="12" r="3" />
            <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 1 1-4 0v-.09a1.65 1.65 0 0 0-1-1.51 1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 1 1 0-4h.09a1.65 1.65 0 0 0 1.51-1 1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33h.01a1.65 1.65 0 0 0 1-1.51V3a2 2 0 1 1 4 0v.09a1.65 1.65 0 0 0 1 1.51h.01a1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82v.01a1.65 1.65 0 0 0 1.51 1H21a2 2 0 1 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z" />
          </svg>
        </router-link>
      </div>
    </header>

    <transition name="banner" :duration="bannerDuration">
      <div v-if="ui.banner" class="success-banner" role="status">{{ ui.banner }}</div>
    </transition>

    <main class="page">
      <router-view v-slot="{ Component }">
        <transition name="page-fade" mode="out-in" :duration="pageFadeDuration">
          <component :is="Component" :key="route.fullPath" />
        </transition>
      </router-view>
    </main>
  </div>
</template>

<style scoped>
.topbar {
  position: sticky;
  top: 0;
  z-index: 30;
  background: #fff;
  border-bottom: 1px solid #f0f0f0;
}

.topbar-inner {
  max-width: 960px;
  margin: 0 auto;
  height: 52px;
  padding: 0 16px;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

@media (min-width: 640px) {
  .topbar-inner {
    padding: 0 32px;
  }
}

.topbar-left {
  display: flex;
  align-items: center;
  gap: 4px;
  min-width: 0;
}

.topbar-title {
  font-size: 16px;
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.icon-btn {
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-sm);
  color: var(--text);
}

.icon-btn:hover {
  background: #f5f5f5;
}

.success-banner {
  position: fixed;
  top: 60px;
  left: 50%;
  transform: translateX(-50%);
  z-index: 50;
  background: #e8f7ee;
  color: var(--success);
  font-size: 14px;
  padding: 8px 24px;
  border-radius: 999px;
  max-width: calc(100vw - 32px);
}
</style>
