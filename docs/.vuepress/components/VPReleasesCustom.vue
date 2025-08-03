<script setup lang="ts">
import Releases from "./Releases.vue";

defineProps<{
  repo: string;
}>();
</script>

<template>
  <VPHomeBox>
    <Releases :repo="repo" />
  </VPHomeBox>
</template>
