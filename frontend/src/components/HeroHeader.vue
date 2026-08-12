<script setup>
import { RefreshCw } from "@lucide/vue";

defineProps({
  availableCount: { type: Number, default: 0 },
  totalCount: { type: Number, default: 0 },
  channelCount: { type: Number, default: 0 },
  loading: { type: Boolean, default: false }
});

defineEmits(["refresh"]);
</script>

<template>
  <header class="hero">
    <div class="hero-copy">
      <p class="eyebrow">AppleShare Hub</p>
      <h1>苹果账号分发状态</h1>
      <p class="hero-sub">账号仅限 App Store 下载使用，使用后请立即按 iOS 26 方式退出。</p>
    </div>
    <div class="hero-metrics" aria-label="账号汇总">
      <div class="metric">
        <strong>{{ availableCount }}</strong>
        <span>可用</span>
      </div>
      <div class="metric">
        <strong>{{ totalCount }}</strong>
        <span>总数</span>
      </div>
      <div class="metric">
        <strong>{{ channelCount }}</strong>
        <span>渠道</span>
      </div>
    </div>
    <button class="button button-primary" type="button" :disabled="loading" @click="$emit('refresh')">
      <RefreshCw :size="17" :class="{ spin: loading }" />
      <span>{{ loading ? "刷新中" : "刷新" }}</span>
    </button>
  </header>
</template>
