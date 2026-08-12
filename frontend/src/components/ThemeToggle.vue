<script setup>
import { Monitor, Moon, Sun } from "@lucide/vue";

defineProps({
  theme: { type: String, default: "auto" }
});

defineEmits(["change"]);

const modes = [
  { value: "auto", label: "自动", icon: Monitor },
  { value: "light", label: "浅色", icon: Sun },
  { value: "dark", label: "深色", icon: Moon }
];
</script>

<template>
  <div class="theme-toggle" role="group" aria-label="外观模式">
    <button
      v-for="mode in modes"
      :key="mode.value"
      class="theme-option"
      :class="{ 'is-active': theme === mode.value }"
      type="button"
      :title="`${mode.label}模式`"
      @click="$emit('change', mode.value)"
    >
      <component :is="mode.icon" :size="15" />
      <span>{{ mode.label }}</span>
    </button>
  </div>
</template>
