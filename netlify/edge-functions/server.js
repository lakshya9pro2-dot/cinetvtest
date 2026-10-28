import worker from "../../worker.js";

export default async function (request, context) {
  const url = new URL(request.url);

  // Handle favicon.ico cleanly
  if (url.pathname === "/favicon.ico") {
    return new Response(null, { status: 204 });
  }

  // If visiting the homepage without query parameters, serve static public/index.html
  if (url.pathname === "/" && !url.searchParams.get("type") && !url.searchParams.get("id")) {
    return context.next();
  }

  try {
    const response = await worker.fetch(request, {}, context);
    return response;
  } catch (err) {
    console.error(`[NETLIFY EDGE ERROR] Uncaught exception:`, err);
    return new Response(JSON.stringify({ error: err.message || "Internal Edge Error" }), {
      status: 500,
      headers: { "Content-Type": "application/json" },
    });
  }
}
