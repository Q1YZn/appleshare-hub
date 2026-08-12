import { jsonResponse } from "../_lib/r2.js";
import { dailyStatsKey, publicDailyStats, shanghaiDate } from "../_lib/stats.js";

export async function onRequestGet(context) {
  if (!context.env.BUCKET) {
    return jsonResponse({ code: 503, message: "stats unavailable" }, 503);
  }
  const url = new URL(context.request.url);
  const requestedDays = Number(url.searchParams.get("days")) || 7;
  const days = Math.min(90, Math.max(1, Math.floor(requestedDays)));
  const rows = [];
  for (let index = 0; index < days; index += 1) {
    const date = shanghaiDate(new Date(Date.now() - index * 24 * 60 * 60 * 1000));
    const object = await context.env.BUCKET.get(dailyStatsKey(date));
    if (!object) {
      rows.push({ date, uv: 0 });
      continue;
    }
    try {
      const stats = JSON.parse(await object.text());
      rows.push(publicDailyStats(stats));
    } catch (error) {
      rows.push({ date, uv: 0 });
    }
  }
  return jsonResponse({ code: 200, days, rows });
}
