<script setup lang="ts">
withDefaults(defineProps<{ modelValue: string; type?: 'date' | 'datetime-local' }>(), {
  type: 'date',
})

const emit = defineEmits<{ (e: 'update:modelValue', v: string): void }>()
</script>

<template>
  <input
    class="date-field"
    :type="type"
    :value="modelValue"
    @input="emit('update:modelValue', ($event.target as HTMLInputElement).value)"
  />
</template>

<style scoped>
.date-field {
  width: 100%;
  min-height: 40px;
  padding: 8px 12px;
  border: 1px solid #e0e0e6;
  border-radius: var(--radius-sm);
  background: #fff;
  color: var(--text);
  font-family: inherit;
  font-size: 14px;
  outline: none;
}

.date-field:focus {
  border-color: var(--primary);
}
</style>
