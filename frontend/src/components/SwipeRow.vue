<script setup lang="ts">
import { ref } from 'vue'

const props = defineProps<{ canDelete: boolean }>()
const emit = defineEmits<{ (e: 'delete'): void }>()

const WIDTH = 72
const offset = ref(0)
const moving = ref(false)

let startX = 0
let startY = 0
let startOffset = 0
let axis: 'x' | 'y' | null = null

function onTouchStart(e: TouchEvent) {
  startX = e.touches[0].clientX
  startY = e.touches[0].clientY
  startOffset = offset.value
  axis = null
  moving.value = true
}

function onTouchMove(e: TouchEvent) {
  const dx = e.touches[0].clientX - startX
  const dy = e.touches[0].clientY - startY
  if (!axis) {
    if (Math.abs(dx) < 8 && Math.abs(dy) < 8) return
    axis = Math.abs(dx) > Math.abs(dy) ? 'x' : 'y'
  }
  if (axis !== 'x') return
  if (e.cancelable) e.preventDefault()
  offset.value = Math.min(0, Math.max(-WIDTH, startOffset + dx))
}

function onTouchEnd() {
  moving.value = false
  offset.value = offset.value < -WIDTH / 2 ? -WIDTH : 0
}

function onDelete() {
  offset.value = 0
  emit('delete')
}
</script>

<template>
  <div class="swipe-row" :class="{ 'swipe-hover': props.canDelete }">
    <div
      class="swipe-content"
      :class="{ settling: !moving }"
      :style="canDelete ? { transform: `translateX(${offset}px)` } : undefined"
      @touchstart="canDelete && onTouchStart($event)"
      @touchmove="canDelete && onTouchMove($event)"
      @touchend="canDelete && onTouchEnd()"
    >
      <slot />
    </div>
    <button v-if="canDelete" class="swipe-delete" type="button" @click="onDelete">删除</button>
  </div>
</template>

<style scoped>
.swipe-row {
  position: relative;
  overflow: hidden;
  border-radius: var(--radius-lg);
}

.swipe-content.settling {
  transition: transform 0.2s cubic-bezier(0.4, 0, 0.2, 1);
}

.swipe-delete {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  width: 72px;
  background: var(--danger);
  color: #fff;
  font-size: 14px;
  display: none;
  align-items: center;
  justify-content: center;
}

/* 桌面端 hover 显示删除按钮；移动端走左滑 */
@media (hover: hover) {
  .swipe-hover:hover .swipe-delete {
    display: flex;
  }
}

.swipe-content:active + .swipe-delete {
  display: flex;
}
</style>
