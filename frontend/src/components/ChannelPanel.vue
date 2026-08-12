<script setup>
import { channelLetter, statusClass } from "../utils/format.js";

defineProps({
  channels: { type: Array, default: () => [] },
  selectedChannel: { type: String, default: "" }
});

defineEmits(["toggle"]);

const stateLabel = {
  ok: "正常",
  empty: "无账号",
  error: "获取失败"
};

function channelTitle(channel, index) {
  const label = stateLabel[channel.status] || channel.status;
  const detail = channel.error ? `：${channel.error}` : "";
  return `渠道${channelLetter(index)} ${channel.name}：${label}${detail}`;
}
</script>

<template>
  <section class="panel channel-panel">
    <div class="section-head compact">
      <h2>渠道状态</h2>
    </div>
    <div class="channel-list">
      <button
        v-for="(channel, index) in channels"
        :key="channel.id"
        class="channel-pill"
        :class="{ 'is-active': selectedChannel === channel.id }"
        type="button"
        :title="channelTitle(channel, index)"
        @click="$emit('toggle', channel.id)"
      >
        <i class="dot" :class="statusClass(channel.status)"></i>
        渠道{{ channelLetter(index) }}
      </button>
    </div>
  </section>
</template>
