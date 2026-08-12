function shanghaiDate(now = new Date()) {
  return new Intl.DateTimeFormat("en-CA", {
    timeZone: "Asia/Shanghai",
    year: "numeric",
    month: "2-digit",
    day: "2-digit"
  }).format(now);
}

export function dailyStatsKey(date) {
  return `stats/daily/${date}.json`;
}

export function emptyDailyStats(date) {
  return { date, uv: 0, uids: [] };
}

export async function readDailyStats(bucket, date) {
  const object = await bucket.get(dailyStatsKey(date));
  if (!object) {
    return emptyDailyStats(date);
  }
  try {
    const parsed = JSON.parse(await object.text());
    if (parsed && typeof parsed === "object" && parsed.date === date) {
      return {
        date,
        uv: Number.isFinite(Number(parsed.uv)) ? Math.max(0, Math.floor(Number(parsed.uv))) : 0,
        uids: Array.isArray(parsed.uids)
          ? parsed.uids.filter((uid) => typeof uid === "string" && uid.length <= 128)
          : []
      };
    }
  } catch (error) {
    // fall back to an empty daily record when stored JSON is malformed
  }
  return emptyDailyStats(date);
}

export async function saveDailyStats(bucket, stats) {
  await bucket.put(dailyStatsKey(stats.date), JSON.stringify(stats), {
    httpMetadata: {
      contentType: "application/json; charset=utf-8",
      cacheControl: "no-store"
    }
  });
}

export async function recordVisit(bucket, uid, now = new Date()) {
  const date = shanghaiDate(now);
  const stats = await readDailyStats(bucket, date);
  let created = false;
  if (!stats.uids.includes(uid)) {
    stats.uids.push(uid);
    stats.uv = stats.uids.length;
    await saveDailyStats(bucket, stats);
    created = true;
  }
  return { date, stats, created };
}

export function publicDailyStats(stats) {
  return { date: stats.date, uv: stats.uv };
}

export { shanghaiDate };
