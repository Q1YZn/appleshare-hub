import { refresh } from "../functions/_lib/refresh.js";

function summary(result) {
  return JSON.stringify({
    generated_at: result.snapshot.generated_at,
    total_count: result.snapshot.total_count,
    available_count: result.snapshot.available_count,
    deleted_count: result.deleted.length
  });
}

export default {
  async scheduled(_event, env) {
    try {
      const result = await refresh(env);
      console.log(`refresh ok ${summary(result)}`);
    } catch (error) {
      console.error(`refresh failed: ${error.message}`);
      throw error;
    }
  },

  async fetch(request, env) {
    const token = env.REFRESH_TOKEN;
    if (token && request.headers.get("authorization") !== `Bearer ${token}`) {
      return new Response(JSON.stringify({ code: 401, message: "unauthorized" }), {
        status: 401,
        headers: { "Content-Type": "application/json; charset=utf-8" }
      });
    }
    try {
      const result = await refresh(env);
      return new Response(
        JSON.stringify({
          code: 200,
          message: "ok",
          ...JSON.parse(summary(result))
        }),
        {
          headers: { "Content-Type": "application/json; charset=utf-8" }
        }
      );
    } catch (error) {
      return new Response(
        JSON.stringify({ code: 500, message: error.message || "refresh failed" }),
        {
          status: 500,
          headers: { "Content-Type": "application/json; charset=utf-8" }
        }
      );
    }
  }
};
