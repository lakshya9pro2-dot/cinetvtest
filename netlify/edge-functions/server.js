import worker from "../../worker.js";

export default async function (request, context) {
  const url = new URL(request.url);
  // If visiting the homepage without query parameters, serve static public/index.html
  if (url.pathname === "/" && !url.searchParams.get("type") && !url.searchParams.get("id")) {
    return context.next();
  }
  return worker.fetch(request, {}, context);
}
