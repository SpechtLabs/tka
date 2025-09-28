<template>
  <div class="list-compare">
    <div v-if="computedTitle || computedDescription" class="list-compare__header">
      <h2 v-if="computedTitle" class="list-compare__heading">{{ computedTitle }}</h2>
      <p v-if="computedDescription" class="list-compare__sub">{{ computedDescription }}</p>
    </div>
    <div class="list-compare__column">
      <h3 v-if="computedLeftTitle" class="list-compare__title">{{ computedLeftTitle }}</h3>
      <p v-if="computedLeftDescription" class="list-compare__description">{{ computedLeftDescription }}</p>
      <ul class="list-compare__list">
        <li v-for="(item, idx) in normalizedLeft" :key="'l-' + idx" class="list-compare__item">
          <span class="list-compare__bullet">•</span>
          <div class="list-compare__content">
            <div v-if="item.title" class="list-compare__item-title">{{ item.title }}</div>
            <div v-if="item.description" class="list-compare__item-desc" v-html="item.description"></div>
          </div>
        </li>
      </ul>
    </div>
    <div class="list-compare__divider" aria-hidden="true"></div>
    <div class="list-compare__column">
      <h3 v-if="computedRightTitle" class="list-compare__title">{{ computedRightTitle }}</h3>
      <p v-if="computedRightDescription" class="list-compare__description">{{ computedRightDescription }}</p>
      <ul class="list-compare__list">
        <li v-for="(item, idx) in normalizedRight" :key="'r-' + idx" class="list-compare__item">
          <span class="list-compare__bullet">•</span>
          <div class="list-compare__content">
            <div v-if="item.title" class="list-compare__item-title">{{ item.title }}</div>
            <div v-if="item.description" class="list-compare__item-desc" v-html="item.description"></div>
          </div>
        </li>
      </ul>
    </div>
  </div>
  <p v-if="note" class="list-compare__note">{{ note }}</p>
</template>

<script lang="ts">
import { computed, defineComponent } from "vue";

export default defineComponent({
  name: "ListCompare",
  props: {
    model: {
      type: Object as () => {
        title?: string;
        description?: string;
        left?: { title?: string; description?: string; items?: Array<string | { title?: string; description?: string }> };
        right?: { title?: string; description?: string; items?: Array<string | { title?: string; description?: string }> };
      },
      default: undefined,
    },
    title: {
      type: String,
      default: "",
    },
    description: {
      type: String,
      default: "",
    },
    leftBlock: {
      type: Object as () => {
        title?: string;
        description?: string;
        items?: Array<string | { title?: string; description?: string }>;
      },
      default: () => ({}),
    },
    rightBlock: {
      type: Object as () => {
        title?: string;
        description?: string;
        items?: Array<string | { title?: string; description?: string }>;
      },
      default: () => ({}),
    },
    left: {
      type: Array as () => Array<string | { title?: string; description?: string }>,
      required: false,
      default: () => [],
    },
    right: {
      type: Array as () => Array<string | { title?: string; description?: string }>,
      required: false,
      default: () => [],
    },
    leftTitle: {
      type: String,
      default: "",
    },
    rightTitle: {
      type: String,
      default: "",
    },
    note: {
      type: String,
      default: "",
    },
  },
  setup(props) {
    function normalize(items: Array<string | { title?: string; description?: string }>) {
      return (items || []).map((it) => {
        if (typeof it === "string") {
          return { title: "", description: it };
        }
        return { title: it.title || "", description: it.description || "" };
      });
    }

    const effectiveLeftBlock = computed(() => props.model?.left || props.leftBlock || {});
    const effectiveRightBlock = computed(() => props.model?.right || props.rightBlock || {});

    const normalizedLeft = computed(() =>
      (effectiveLeftBlock.value as any)?.items?.length
        ? normalize((effectiveLeftBlock.value as any).items)
        : normalize(props.left),
    );
    const normalizedRight = computed(() =>
      (effectiveRightBlock.value as any)?.items?.length
        ? normalize((effectiveRightBlock.value as any).items)
        : normalize(props.right),
    );

    const computedLeftTitle = computed(() =>
      (effectiveLeftBlock.value as any)?.title || props.leftTitle || "",
    );
    const computedRightTitle = computed(() =>
      (effectiveRightBlock.value as any)?.title || props.rightTitle || "",
    );
    const computedLeftDescription = computed(() =>
      (effectiveLeftBlock.value as any)?.description || "",
    );
    const computedRightDescription = computed(() =>
      (effectiveRightBlock.value as any)?.description || "",
    );

    const computedTitle = computed(() => props.model?.title || props.title || "");
    const computedDescription = computed(
      () => props.model?.description || props.description || "",
    );

    return {
      normalizedLeft,
      normalizedRight,
      computedLeftTitle,
      computedRightTitle,
      computedLeftDescription,
      computedRightDescription,
      computedTitle,
      computedDescription,
    };
  },
});
</script>

<style scoped>
.list-compare {
  display: grid;
  grid-template-columns: 1fr 1px 1fr;
  gap: 24px;
  align-items: start;
  background-color: var(--vp-c-bg-soft);
  border: 1px solid var(--vp-c-bg-soft);
  border-radius: 12px;
  padding: 24px;
  transition: border-color var(--vp-t-color), background-color var(--vp-t-color);
}

@media (max-width: 768px) {
  .list-compare {
    grid-template-columns: 1fr;
  }
  .list-compare__divider {
    display: none;
  }
}

.list-compare__divider {
  width: 1px;
  height: 100%;
  background: var(--vp-c-default-soft);
}

.list-compare__column {
  min-width: 0;
}

.list-compare__title {
  margin: 0 0 8px 0;
  font-size: 16px;
  font-weight: 600;
  line-height: 24px;
  color: var(--vp-c-text-1);
}

.list-compare__description {
  margin: 0 0 8px 0;
  font-size: 14px;
  font-weight: 500;
  line-height: 24px;
  color: var(--vp-c-text-2);
}

.list-compare__list {
  list-style: none;
  padding: 0;
  margin: 0;
}

.list-compare__item {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 8px 10px;
  border-radius: 8px;
  transition: background-color var(--vp-t-color);
}

.list-compare__item:hover {
  background: var(--vp-c-bg-elv);
}

.list-compare__bullet {
  color: var(--vp-c-accent);
  line-height: 1.4;
}

.list-compare__content {
  color: var(--vp-c-text-1);
}

.list-compare__item-title {
  font-size: 16px;
  font-weight: 600;
  line-height: 24px;
  color: var(--vp-c-text-1);
  margin-bottom: 2px;
}

.list-compare__item-desc {
  font-size: 14px;
  font-weight: 500;
  line-height: 24px;
  color: var(--vp-c-text-2);
}

.list-compare__header {
  grid-column: 1 / -1;
  margin-bottom: 20px;
}

.list-compare__heading {
  margin-bottom: 20px;
  font-size: 20px;
  font-weight: 900;
  color: var(--vp-c-text-1);
  text-align: center;
  transition: color var(--vp-t-color);
}

.list-compare__sub {
  margin-bottom: 20px;
  font-size: 16px;
  line-height: 1.7;
  color: var(--vp-c-text-1);
  text-align: center;
  transition: color var(--vp-t-color);
}

@media (min-width: 768px) {
  .list-compare__heading {
    font-size: 24px;
  }
  .list-compare__sub {
    font-size: 18px;
  }
}

@media (min-width: 960px) {
  .list-compare__heading {
    font-size: 28px;
  }
}

.list-compare__note {
  margin-top: 8px;
  text-align: center;
  color: var(--vp-c-text-2);
  font-size: 12px;
}
</style>
