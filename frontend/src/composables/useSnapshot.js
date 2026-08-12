import { onBeforeUnmount, onMounted, ref } from "vue";

const MIN_REFRESH_DELAY_MS = 5000;
const FALLBACK_TTL_SECONDS = 300;

export function useSnapshot() {
  const snapshot = ref(null);
  const loading = ref(false);
  const error = ref("");
  let refreshTimer = null;

  function stopAutoRefresh() {
    if (refreshTimer) {
      clearTimeout(refreshTimer);
    }
    refreshTimer = null;
  }

  function scheduleRefresh(ttlSeconds) {
    stopAutoRefresh();
    if (document.visibilityState === "hidden") {
      return;
    }
    const delay = Math.max(
      MIN_REFRESH_DELAY_MS,
      (Number(ttlSeconds) || FALLBACK_TTL_SECONDS) * 1000 + 800
    );
    refreshTimer = setTimeout(() => {
      load(false);
    }, delay);
  }

  async function load(force = false) {
    loading.value = true;
    error.value = "";
    try {
      const options = { headers: { Accept: "application/json" } };
      if (force) {
        options.cache = "no-store";
      }
      const response = await fetch("/api/accounts", options);
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }
      snapshot.value = await response.json();
      scheduleRefresh(snapshot.value.cache_ttl_seconds);
      return snapshot.value;
    } catch (err) {
      error.value = err.message || "网络错误";
      scheduleRefresh(FALLBACK_TTL_SECONDS);
      throw err;
    } finally {
      loading.value = false;
    }
  }

  function handleVisibilityChange() {
    if (document.visibilityState === "visible") {
      if (!refreshTimer && !loading.value) {
        load(false);
      }
    } else {
      stopAutoRefresh();
    }
  }

  onMounted(() => {
    document.addEventListener("visibilitychange", handleVisibilityChange);
    load(false);
  });

  onBeforeUnmount(() => {
    document.removeEventListener("visibilitychange", handleVisibilityChange);
    stopAutoRefresh();
  });

  return { snapshot, loading, error, load };
}
