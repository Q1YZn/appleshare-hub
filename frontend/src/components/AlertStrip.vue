<script setup>
import { Info, Settings2, ShieldAlert } from "@lucide/vue";

defineProps({
  warnings: { type: Array, default: () => [] }
});

const levelClass = {
  danger: "alert-danger",
  warn: "alert-warn",
  info: "alert-info"
};

const levelIcon = {
  danger: ShieldAlert,
  warn: Settings2,
  info: Info
};

function alertClass(level) {
  return levelClass[level] || "alert-info";
}

function alertIcon(level) {
  return levelIcon[level] || Info;
}
</script>

<template>
  <section v-if="warnings.length" class="alert-strip" aria-label="安全提示">
    <div v-for="warning in warnings" :key="warning.title" class="alert-item" :class="alertClass(warning.level)">
      <component :is="alertIcon(warning.level)" :size="22" />
      <div>
        <h2>{{ warning.title }}</h2>
        <p>{{ warning.content }}</p>
      </div>
    </div>
  </section>
</template>
