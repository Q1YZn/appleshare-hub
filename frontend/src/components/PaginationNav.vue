<script setup>
import { ChevronLeft, ChevronRight } from "@lucide/vue";
import { computed } from "vue";

const props = defineProps({
  currentPage: { type: Number, default: 1 },
  totalPages: { type: Number, default: 1 }
});

defineEmits(["page"]);

const pages = computed(() => {
  const total = props.totalPages;
  const current = props.currentPage;
  const items = [];
  let start = Math.max(1, current - 2);
  let end = Math.min(total, start + 4);
  start = Math.max(1, end - 4);

  if (start > 1) {
    items.push(1);
    if (start > 2) {
      items.push("gap");
    }
  }
  for (let page = start; page <= end; page += 1) {
    items.push(page);
  }
  if (end < total) {
    if (end < total - 1) {
      items.push("gap");
    }
    items.push(total);
  }
  return items;
});
</script>

<template>
  <nav v-if="totalPages > 1" class="pagination" aria-label="账号分页">
    <button
      class="pagination-button"
      type="button"
      :disabled="currentPage <= 1"
      @click="$emit('page', currentPage - 1)"
    >
      <ChevronLeft :size="15" />
      <span>上一页</span>
    </button>
    <span class="pagination-summary">{{ currentPage }} / {{ totalPages }}</span>
    <template v-for="(item, index) in pages" :key="item === 'gap' ? `gap-${index}` : item">
      <span v-if="item === 'gap'" class="pagination-gap">...</span>
      <button
        v-else
        class="pagination-button"
        :class="{ 'is-current': item === currentPage }"
        type="button"
        :aria-current="item === currentPage ? 'page' : undefined"
        @click="$emit('page', item)"
      >
        {{ item }}
      </button>
    </template>
    <button
      class="pagination-button"
      type="button"
      :disabled="currentPage >= totalPages"
      @click="$emit('page', currentPage + 1)"
    >
      <span>下一页</span>
      <ChevronRight :size="15" />
    </button>
  </nav>
</template>
