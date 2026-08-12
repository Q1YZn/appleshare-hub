import { jsonResponse, readSnapshotText } from "../_lib/r2.js";

export async function onRequestGet(context) {
  const url = new URL(context.request.url);
  url.pathname = "/api/accounts";
  url.search = "";
  const cacheKey = new Request(url.toString(), { method: "GET" });
  const cache = typeof globalThis.caches !== "undefined" ? globalThis.caches.default : null;

  try {
    let cached = null;
    if (cache) {
      try {
        cached = await cache.match(cacheKey);
      } catch (error) {
        cached = null;
      }
      if (cached) {
        return cached;
      }
    }

    const text = await readSnapshotText(context.env);
    const snapshot = JSON.parse(text);
    const ttl = Number(snapshot.cache_ttl_seconds) || 300;
    const response = new Response(text, {
      headers: {
        "Content-Type": "application/json; charset=utf-8",
        "Cache-Control": `public, max-age=${ttl}, s-maxage=${ttl}`
      }
    });
    if (cache) {
      try {
        await cache.put(cacheKey, response.clone());
      } catch (error) {
        // cache is an optimization; never fail the API because of it
      }
    }
    return response;
  } catch (error) {
    return jsonResponse(
      {
        code: 500,
        message: error.message || "failed to read snapshot"
      },
      500
    );
  }
}
