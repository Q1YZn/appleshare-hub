import { emptySnapshot } from "./snapshot.js";
import { loadConfig } from "./config.js";

export function jsonResponse(data, status = 200) {
  return new Response(JSON.stringify(data), {
    status,
    headers: {
      "Content-Type": "application/json; charset=utf-8",
      "Cache-Control": "no-store"
    }
  });
}

export async function readSnapshotText(env) {
  const bucket = env.BUCKET;
  if (!bucket) {
    throw new Error("R2 binding BUCKET is not configured");
  }
  const key = env.SNAPSHOT_KEY || "snapshot/latest.json";
  const object = await bucket.get(key);
  if (!object) {
    const config = loadConfig(env);
    return JSON.stringify(emptySnapshot(config.server.cache_ttl_seconds));
  }
  return object.text();
}
