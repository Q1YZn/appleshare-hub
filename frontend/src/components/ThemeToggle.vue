<script setup>
import { computed } from "vue";
import { Monitor, Moon, Sun } from "@lucide/vue";

const props = defineProps({
  theme: { type: String, default: "auto" }
});

defineEmits(["change"]);

const modes = [
  { value: "auto", label: "自动", icon: Monitor },
  { value: "light", label: "浅色", icon: Sun },
  { value: "dark", label: "深色", icon: Moon }
];

const currentMode = computed(
  () => modes.find((mode) => mode.value === props.theme) || modes[0]
);
</script>

<template>
  <label class="theme-toggle" :title="`${currentMode.label}模式`">
    <span class="theme-toggle-icon">
      <component :is="currentMode.icon" :size="15" />
    </span>
    <select
      class="theme-select"
      :value="theme"
      aria-label="外观模式"
      @change="$emit('change', $event.target.value)"
    >
      <option v-for="mode in modes" :key="mode.value" :value="mode.value">
        {{ mode.label }}
      </option>
    </select>
  </label>
</template>
