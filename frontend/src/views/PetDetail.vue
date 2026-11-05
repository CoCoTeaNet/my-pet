<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import PetOverview from '@/components/PetOverview.vue'

const route = useRoute()
const petId = computed(() => route.params.id as string)
</script>

<template>
  <pet-overview :pet-id="petId" />
</template>
