<script setup>
import { CloudOff, Inbox } from "@lucide/vue";
import AccountCard from "./AccountCard.vue";
import FilterControls from "./FilterControls.vue";
import PaginationNav from "./PaginationNav.vue";
import { channelLabel, statusClass } from "../utils/format.js";

defineProps({
  snapshot: { type: Object, default: null },
  accounts: { type: Array, default: () => [] },
  currentPage: { type: Number, default: 1 },
  totalPages: { type: Number, default: 1 },
  selectedCountry: { type: String, default: "" },
  selectedChannel: { type: String, default: "" },
  selectedShadowrocket: { type: String, default: "" },
  onlyAvailable: { type: Boolean, default: false },
  countryOptions: { type: Array, default: () => [] },
  legend: { type: Array, default: () => [] },
  loading: { type: Boolean, default: false },
  error: { type: String, default: "" },
  generatedAt: { type: String, default: "正在获取…" }
});

defineEmits(["update:onlyAvailable", "update:country", "update:shadowrocket", "page"]);
</script>

<template>
  <section class="panel account-panel">
    <div class="panel-header">
      <div class="panel-title-area">
        <h2>账号状态</h2>
        <p class="section-note">{{ generatedAt }}</p>
      </div>
      <div class="legend">
        <span v-for="item in legend" :key="item.status" class="legend-item" :title="item.description || item.label">
          <i class="dot" :class="statusClass(item.status)"></i>
          {{ item.label }}
        </span>
      </div>
    </div>

    <FilterControls
      :only-available="onlyAvailable"
      :selected-country="selectedCountry"
      :selected-shadowrocket="selectedShadowrocket"
      :country-options="countryOptions"
      @update:only-available="$emit('update:onlyAvailable', $event)"
      @update:country="$emit('update:country', $event)"
      @update:shadowrocket="$emit('update:shadowrocket', $event)"
    />

    <div v-if="accounts.length" class="account-list" aria-live="polite">
      <AccountCard
        v-for="account in accounts"
        :key="account.id || `${account.channel}-${account.username}`"
        :account="account"
        :channel-label="channelLabel(snapshot, account.channel)"
      />
    </div>
    <div v-else-if="loading" class="empty-state">
      <p>正在获取账号…</p>
    </div>
    <div v-else-if="error" class="empty-state">
      <CloudOff :size="34" />
      <p>无法连接服务，请稍后重试</p>
    </div>
    <div v-else class="empty-state">
      <Inbox :size="34" />
      <p>暂无可展示的账号</p>
    </div>

    <PaginationNav
      :current-page="currentPage"
      :total-pages="totalPages"
      @page="$emit('page', $event)"
    />
  </section>
</template>
