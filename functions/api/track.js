import { jsonResponse } from "../_lib/r2.js";
import { recordVisit } from "../_lib/stats.js";

const UID_PATTERN = /^[A-Za-z0-9_-]{8,128}$/;

async function readUid(request) {
  const url = new URL(request.url);
  const queryUid = url.searchParams.get("uid");
  let bodyUid = "";
  if (request.body) {
    try {
      const body = await request.json();
      if (body && typeof body.uid === "string") {
        bodyUid = body.uid;
      }
    } catch (error) {
      // query fallback is fine for simple beacons
    }
  }
  return String(bodyUid || queryUid || "").trim();
}

export async function onRequestPost(context) {
  if (!context.env.BUCKET) {
    return jsonResponse({ code: 503, message: "tracking unavailable" }, 503);
  }
  const uid = await readUid(context.request);
  if (!UID_PATTERN.test(uid)) {
    return jsonResponse({ code: 400, message: "invalid uid" }, 400);
  }
  try {
    const result = await recordVisit(context.env.BUCKET, uid);
    return jsonResponse({
      code: 200,
      message: "ok",
      date: result.date,
      created: result.created
    });
  } catch (error) {
    return jsonResponse(
      {
        code: 500,
        message: error.message || "tracking failed"
      },
      500
    );
  }
}
