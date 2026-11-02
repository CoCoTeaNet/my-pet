<script setup lang="ts">
import { useRoute } from 'vue-router'

const route = useRoute()

const tabs = [
  { label: '宠物', to: '/', active: (p: string) => p === '/' || p.startsWith('/pet/') },
  { label: '提醒', to: '/reminders', active: (p: string) => p === '/reminders' },
  { label: '我的', to: '/settings', active: (p: string) => p === '/settings' },
]
</script>

<template>
  <nav class="bottom-nav">
    <router-link
      v-for="t in tabs"
      :key="t.to"
      :to="t.to"
      class="nav-item"
      :class="{ active: t.active(route.path) }"
    >
      <span class="nav-pill">
        <svg v-if="t.label === '宠物'" width="22" height="22" viewBox="0 0 24 24" fill="currentColor">
          <ellipse cx="6" cy="10" rx="2.2" ry="2.8" />
          <ellipse cx="10.2" cy="6.2" rx="2.2" ry="2.9" />
          <ellipse cx="15" cy="6.2" rx="2.2" ry="2.9" />
          <ellipse cx="19" cy="10" rx="2.2" ry="2.8" />
          <path d="M12.5 11.2c2.5 0 4.8 2.2 4.8 4.5 0 1.9-1.6 3.3-3.6 3.3-1 0-1.7-.4-2.9-.4s-1.9.4-2.9.4c-2 0-3.6-1.4-3.6-3.3 0-2.3 2.3-4.5 4.8-4.5.5 0 1 .2 1.7.2s1.2-.2 1.7-.2Z" />
        </svg>
        <svg v-else-if="t.label === '提醒'" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
          <path d="M18 8a6 6 0 1 0-12 0c0 7-3 7-3 9h18c0-2-3-2-3-9" />
          <path d="M10.3 21a1.94 1.94 0 0 0 3.4 0" />
        </svg>
        <svg v-else width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
          <circle cx="12" cy="8" r="4" />
          <path d="M4.5 20.5a7.5 7.5 0 0 1 15 0" />
        </svg>
      </span>
      <span class="nav-label">{{ t.label }}</span>
    </router-link>
  </nav>
</template>

<style scoped>
.bottom-nav {
  position: fixed;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 40;
  display: flex;
  justify-content: space-around;
  align-items: center;
  padding: 6px 8px calc(6px + env(safe-area-inset-bottom));
  background: rgba(255, 255, 255, 0.9);
  backdrop-filter: saturate(180%) blur(16px);
  -webkit-backdrop-filter: saturate(180%) blur(16px);
  border-top: 1px solid rgba(0, 0, 0, 0.06);
}

.nav-item {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
  padding: 4px 0;
  color: var(--text-sub);
  -webkit-tap-highlight-color: transparent;
  transition: color 0.15s cubic-bezier(0.4, 0, 0.2, 1);
}

.nav-pill {
  width: 56px;
  height: 30px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 999px;
  transition: background 0.18s cubic-bezier(0.4, 0, 0.2, 1);
}

.nav-item.active {
  color: var(--primary);
}

.nav-item.active .nav-pill {
  background: var(--primary-soft);
}

.nav-label {
  font-size: 11px;
  line-height: 1.2;
}
</style>
