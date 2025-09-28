<script setup lang="ts">
import ListCompare from "./ListCompare.vue";

defineProps<{
  title?: string;
  description?: string;
  left?: { title?: string; description?: string; items?: Array<string | { title?: string; description?: string }> };
  right?: { title?: string; description?: string; items?: Array<string | { title?: string; description?: string }> };
  leftTitle?: string; // legacy
  rightTitle?: string; // legacy
  leftItems?: Array<string | { title?: string; description?: string }>; // legacy
  rightItems?: Array<string | { title?: string; description?: string }>; // legacy
  leftDescription?: string; // legacy
  rightDescription?: string; // legacy
  note?: string;
}>();
</script>

<template>
  <VPHomeBox>
    <ListCompare
      :model="title || description || left || right ? { title, description, left, right } : undefined"
      :title="title"
      :description="description"
      :left-block="left || { title: leftTitle, description: leftDescription, items: leftItems }"
      :right-block="right || { title: rightTitle, description: rightDescription, items: rightItems }"
      :left="leftItems || []"
      :right="rightItems || []"
      :left-title="leftTitle"
      :right-title="rightTitle"
      :note="note"
    />
  </VPHomeBox>
</template>
