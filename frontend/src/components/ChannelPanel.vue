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
  let shadowrocket = "，不确定是否 Shadowrocket";
  if (channel.shadowrocket === "certain" || channel.shadowrocket === true) {
    shadowrocket = "，确定有 Shadowrocket";
  } else if (channel.shadowrocket === "possible") {
    shadowrocket = "，可能有 Shadowrocket";
  }
  const name = channel.name || `渠道${channelLetter(index)}`;
  return `${name}：${label}${detail}${shadowrocket}`;
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
        <span>{{ channel.name || `渠道${channelLetter(index)}` }}</span>
      </button>
    </div>
  </section>
</template>
