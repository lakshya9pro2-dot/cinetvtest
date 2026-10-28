import worker from "../../worker.js";

export default async function (request, context) {
  const url = new URL(request.url);
  console.log(`[NETLIFY EDGE] Incoming: ${request.method} ${url.pathname}${url.search}`);

  // If visiting the homepage without query parameters, serve static public/index.html
  if (url.pathname === "/" && !url.searchParams.get("type") && !url.searchParams.get("id")) {
    console.log(`[NETLIFY EDGE] Serving static root from CDN`);
    return context.next();
  }

  try {
    const response = await worker.fetch(request, {}, context);
    console.log(`[NETLIFY EDGE] Response: ${response.status} for ${url.pathname}`);
    return response;
  } catch (err) {
    console.error(`[NETLIFY EDGE ERROR] Uncaught exception:`, err);
    return new Response(JSON.stringify({ error: err.message || "Internal Edge Error" }), {
      status: 500,
      headers: { "Content-Type": "application/json" },
    });
  }
}
