import { jsonResponse, readSnapshotText } from "../_lib/r2.js";

export async function onRequestGet(context) {
  try {
    const text = await readSnapshotText(context.env);
    return new Response(text, {
      headers: {
        "Content-Type": "application/json; charset=utf-8",
        "Cache-Control": "no-store"
      }
    });
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
