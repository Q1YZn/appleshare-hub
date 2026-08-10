import { loadConfig } from "./config.js";
import { buildProviders } from "./providers.js";
import { buildSnapshot } from "./snapshot.js";

function parseRetainHistory(value) {
  const number = Number(value);
  return Number.isFinite(number) && number > 0 ? Math.floor(number) : 0;
}

export async function refresh(env = {}) {
  const config = loadConfig(env);
  const providers = buildProviders(config);
  const results = await Promise.allSettled(providers.map((provider) => provider.fetch()));
  const snapshot = buildSnapshot(
    providers,
    results,
    new Date(),
    config.server.cache_ttl_seconds
  );

  const bucket = env.BUCKET;
  if (!bucket) {
    throw new Error("R2 binding BUCKET is not configured");
  }

  const body = JSON.stringify(snapshot);
  const snapshotKey = env.SNAPSHOT_KEY || "snapshot/latest.json";
  const httpMetadata = {
    contentType: "application/json; charset=utf-8",
    cacheControl: "no-store"
  };
  await bucket.put(snapshotKey, body, { httpMetadata });

  const retainHistory = parseRetainHistory(env.RETAIN_HISTORY);
  const historyPrefix = env.HISTORY_PREFIX || "snapshots/";
  if (retainHistory > 0) {
    const timestamp = snapshot.generated_at.replace(/[:.]/g, "-");
    await bucket.put(`${historyPrefix}${timestamp}.json`, body, {
      httpMetadata,
      customMetadata: { generatedAt: snapshot.generated_at }
    });
  }

  const listed = await bucket.list({ prefix: historyPrefix });
  const objects = (listed && listed.objects ? listed.objects : []).slice();
  objects.sort((a, b) => String(a.key).localeCompare(String(b.key)));
  const toDelete =
    retainHistory > 0
      ? objects.slice(0, Math.max(0, objects.length - retainHistory))
      : objects;
  const deleted = [];
  for (const object of toDelete) {
    await bucket.delete(object.key);
    deleted.push(object.key);
  }

  return { snapshot, deleted };
}
