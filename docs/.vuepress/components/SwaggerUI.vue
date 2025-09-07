<template>
  <div class="swagger-ui"></div>
</template>

<script setup>
import SwaggerUI from "swagger-ui-dist/swagger-ui-es-bundle.js";
import "swagger-ui-dist/swagger-ui.css";
import { onMounted } from "vue";

onMounted(() => {
  SwaggerUI({
    url: "/swagger.json", // served from .vuepress/public
    dom_id: ".swagger-ui",
  });
});
</script>

<style scoped>
/* Swagger UI overrides */
.swagger-ui {
    background: transparent !important;
    color: var(--c-text) !important;
}

.swagger-ui .opblock {
    background: var(--c-bg-soft) !important;
    border-radius: 6px;
}

.swagger-ui .scheme-container {
    background: transparent !important;
}
</style>
