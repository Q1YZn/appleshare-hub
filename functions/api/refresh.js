import { refresh } from "../_lib/refresh.js";
import { jsonResponse } from "../_lib/r2.js";

function isAuthorized(request, env) {
  const token = env.REFRESH_TOKEN;
  if (!token) {
    return true;
  }
  const authorization = request.headers.get("authorization") || "";
  const url = new URL(request.url);
  return authorization === `Bearer ${token}` || url.searchParams.get("token") === token;
}

async function handleRefresh(context) {
  if (!isAuthorized(context.request, context.env)) {
    return jsonResponse({ code: 401, message: "unauthorized" }, 401);
  }
  try {
    const result = await refresh(context.env);
    return jsonResponse({
      code: 200,
      message: "ok",
      generated_at: result.snapshot.generated_at,
      account_count: result.snapshot.total_count,
      available_count: result.snapshot.available_count,
      deleted: result.deleted
    });
  } catch (error) {
    return jsonResponse(
      {
        code: 500,
        message: error.message || "refresh failed"
      },
      500
    );
  }
}

export async function onRequestGet(context) {
  return handleRefresh(context);
}

export async function onRequestPost(context) {
  return handleRefresh(context);
}
