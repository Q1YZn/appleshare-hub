<script setup>
import { computed, ref, watch } from "vue";
import AccountPanel from "./components/AccountPanel.vue";
import AlertStrip from "./components/AlertStrip.vue";
import AppFooter from "./components/AppFooter.vue";
import ChannelPanel from "./components/ChannelPanel.vue";
import GuidePanel from "./components/GuidePanel.vue";
import HeroHeader from "./components/HeroHeader.vue";
import { useSnapshot } from "./composables/useSnapshot.js";
import { channelLetter, formatTime } from "./utils/format.js";

const PAGE_SIZE = 7;
const { snapshot, loading, error, load } = useSnapshot();

const onlyAvailable = ref(false);
const selectedCountry = ref("");
const selectedChannel = ref("");
const selectedShadowrocket = ref("");
const currentPage = ref(1);

const warnings = computed(() => snapshot.value?.warnings || []);
const legend = computed(() => snapshot.value?.status_legend || []);
const channels = computed(() => snapshot.value?.channels || []);

const generatedText = computed(() => {
  if (!snapshot.value) {
    return error.value ? `获取失败：${error.value}` : "正在获取…";
  }
  return `更新于 ${formatTime(snapshot.value.generated_at)}，每 ${snapshot.value.cache_ttl_seconds} 秒刷新一次`;
});

const countryOptions = computed(() => {
  const counts = {};
  for (const account of snapshot.value?.accounts || []) {
    const name = account.country || "未知地区";
    counts[name] = (counts[name] || 0) + 1;
  }
  return Object.entries(counts)
    .sort(([a], [b]) => a.localeCompare(b, "zh-CN"))
    .map(([name, count]) => ({ name, count }));
});

const channelOptions = computed(() =>
  (snapshot.value?.channels || []).map((channel, index) => ({
    id: channel.id,
    label: `渠道${channelLetter(index)}`
  }))
);

const filteredAccounts = computed(() => {
  if (!snapshot.value) {
    return [];
  }
  let accounts = snapshot.value.accounts || [];
  if (onlyAvailable.value) {
    accounts = accounts.filter((account) => account.status === "available");
  }
  if (selectedCountry.value) {
    accounts = accounts.filter(
      (account) => (account.country || "未知地区") === selectedCountry.value
    );
  }
  if (selectedChannel.value) {
    accounts = accounts.filter((account) => account.channel === selectedChannel.value);
  }
  if (selectedShadowrocket.value === "yes") {
    accounts = accounts.filter((account) => account.shadowrocket === true);
  } else if (selectedShadowrocket.value === "uncertain") {
    accounts = accounts.filter((account) => account.shadowrocket !== true);
  }
  return accounts;
});

const totalPages = computed(() =>
  Math.max(1, Math.ceil(filteredAccounts.value.length / PAGE_SIZE))
);

const pagedAccounts = computed(() =>
  filteredAccounts.value.slice((currentPage.value - 1) * PAGE_SIZE, currentPage.value * PAGE_SIZE)
);

watch([filteredAccounts, totalPages], () => {
  if (currentPage.value > totalPages.value) {
    currentPage.value = totalPages.value;
  }
});

watch(countryOptions, (options) => {
  if (selectedCountry.value && !options.some((option) => option.name === selectedCountry.value)) {
    selectedCountry.value = "";
  }
});

watch(channelOptions, (options) => {
  if (selectedChannel.value && !options.some((option) => option.id === selectedChannel.value)) {
    selectedChannel.value = "";
  }
});

function changePage(page) {
  currentPage.value = Math.min(Math.max(1, page), totalPages.value);
}

function setOnlyAvailable(value) {
  onlyAvailable.value = value;
  currentPage.value = 1;
}

function setCountry(value) {
  selectedCountry.value = value;
  currentPage.value = 1;
}

function setChannel(value) {
  selectedChannel.value = value;
  currentPage.value = 1;
}

function setShadowrocket(value) {
  selectedShadowrocket.value = value;
  currentPage.value = 1;
}

function toggleChannel(channelId) {
  selectedChannel.value = selectedChannel.value === channelId ? "" : channelId;
  currentPage.value = 1;
}

async function refresh() {
  try {
    await load(true);
  } catch (err) {
    // error state is already rendered by the composable
  }
}
</script>

<template>
  <HeroHeader
    :available-count="snapshot?.available_count || 0"
    :total-count="snapshot?.total_count || 0"
    :channel-count="channels.length"
    :loading="loading"
    @refresh="refresh"
  />

  <AlertStrip :warnings="warnings" />

  <main class="layout">
    <AccountPanel
      :snapshot="snapshot"
      :accounts="pagedAccounts"
      :current-page="currentPage"
      :total-pages="totalPages"
      :selected-country="selectedCountry"
      :selected-channel="selectedChannel"
      :selected-shadowrocket="selectedShadowrocket"
      :only-available="onlyAvailable"
      :country-options="countryOptions"
      :legend="legend"
      :loading="loading"
      :error="error"
      :generated-at="generatedText"
      @update:only-available="setOnlyAvailable"
      @update:country="setCountry"
      @update:shadowrocket="setShadowrocket"
      @page="changePage"
    />

    <aside class="side">
      <ChannelPanel
        :channels="channels"
        :selected-channel="selectedChannel"
        @toggle="toggleChannel"
      />
      <GuidePanel />
    </aside>
  </main>

  <AppFooter />
</template>
