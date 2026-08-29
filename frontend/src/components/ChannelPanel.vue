<script setup>
import { formatChannelName, statusClass } from "../utils/format.js";

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
  let shadowrocket = "，不确定是否 Shadowrocket";
  if (channel.shadowrocket === "certain" || channel.shadowrocket === true) {
    shadowrocket = "，确定有 Shadowrocket";
  } else if (channel.shadowrocket === "possible") {
    shadowrocket = "，可能有 Shadowrocket";
  }
  const name = formatChannelName(channel, index);
  return `${name}：${label}${detail}${shadowrocket}`;
}
</script>

<template>
  <section class="panel channel-panel">
    <div class="panel-header compact">
      <h2>渠道状态</h2>
      <span class="channel-hint">点击快速筛选</span>
    </div>
    <div class="channel-grid">
      <button
        v-for="(channel, index) in channels"
        :key="channel.id"
        class="channel-pill"
        :class="{
          'is-active': selectedChannel === channel.id,
          'is-error': channel.status === 'error'
        }"
        type="button"
        :title="channelTitle(channel, index)"
        @click="$emit('toggle', channel.id)"
      >
        <i class="dot" :class="statusClass(channel.status)"></i>
        <span class="channel-name">{{ formatChannelName(channel, index) }}</span>
      </button>
    </div>
  </section>
</template>
