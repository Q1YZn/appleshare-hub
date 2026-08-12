const UID_KEY = "ash_uid_v1";
const SENT_DATE_KEY = "ash_stats_date";

function shanghaiDate(now = new Date()) {
  return new Intl.DateTimeFormat("en-CA", {
    timeZone: "Asia/Shanghai",
    year: "numeric",
    month: "2-digit",
    day: "2-digit"
  }).format(now);
}

function safeStorage() {
  try {
    return window.localStorage;
  } catch (error) {
    return null;
  }
}

function generateUid() {
  if (globalThis.crypto && typeof globalThis.crypto.randomUUID === "function") {
    return globalThis.crypto.randomUUID();
  }
  return `u_${Date.now().toString(36)}_${Math.random().toString(36).slice(2, 10)}`;
}

function getOrCreateUid(storage) {
  const existing = storage ? storage.getItem(UID_KEY) : "";
  if (existing && /^[A-Za-z0-9_-]{8,128}$/.test(existing)) {
    return existing;
  }
  const uid = generateUid();
  try {
    storage && storage.setItem(UID_KEY, uid);
  } catch (error) {
    // anonymous counting still works without persistent storage
  }
  return uid;
}

export function useAnalytics() {
  function trackVisit() {
    const storage = safeStorage();
    const today = shanghaiDate();
    if (storage && storage.getItem(SENT_DATE_KEY) === today) {
      return;
    }
    const uid = getOrCreateUid(storage);
    try {
      storage && storage.setItem(SENT_DATE_KEY, today);
    } catch (error) {
      // the beacon below still fires even when storage is unavailable
    }
    const payload = JSON.stringify({ uid });
    if (typeof navigator !== "undefined" && typeof navigator.sendBeacon === "function") {
      navigator.sendBeacon("/api/track", new Blob([payload], { type: "application/json" }));
      return;
    }
    fetch("/api/track", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: payload,
      keepalive: true
    }).catch(() => {});
  }

  return { trackVisit };
}
