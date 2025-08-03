<script setup lang="ts">
import Contributors from "./Contributors.vue";

defineProps<{
  repo: string;
}>();
</script>

<template>
  <VPHomeBox>
    <Contributors :repo="repo" />
  </VPHomeBox>
</template>
