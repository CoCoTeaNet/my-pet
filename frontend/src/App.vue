<script setup lang="ts">
import { NConfigProvider, NDialogProvider, zhCN, dateZhCN } from 'naive-ui'
import type { GlobalThemeOverrides } from 'naive-ui'

const themeOverrides: GlobalThemeOverrides = {
  common: {
    primaryColor: '#F0913C',
    primaryColorHover: '#F5A458',
    primaryColorPressed: '#D97F2F',
    primaryColorSuppl: '#F0913C',
    borderRadius: '8px',
  },
}
</script>

<template>
  <n-config-provider :locale="zhCN" :date-locale="dateZhCN" :theme-overrides="themeOverrides">
    <n-dialog-provider>
      <router-view />
    </n-dialog-provider>
  </n-config-provider>
</template>
