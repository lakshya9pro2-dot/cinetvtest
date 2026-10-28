# NanoServer Universal Stream Worker

A lightweight stream resolver backend API that resolves TMDB IDs (for Movies and TV Shows) into signed, playable MP4/M3U8 streaming URLs. The server automatically searches the upstream streaming provider, handles token generation, payload decryption (AES/DES), and extracts signed stream URLs.

This project is structured to deploy seamlessly as a **Serverless Worker** across:
- **Vercel Functions** (Native Go Serverless Function via `api/index.go` with `package handler`)
- **Cloudflare Workers** (Edge JavaScript Worker via `worker.js` and `wrangler.toml`)
- **Netlify Functions** (Go Serverless via `netlify.toml`)
- **Standalone / Docker / Local VPS** (`go run cmd/server/main.go`, listens on `$PORT` or `1937`)

---

## Deploying to Serverless Platforms

### 1. Vercel Functions (Go)
Vercel automatically detects Go Serverless Functions in the `api/` directory when using `package handler`. 
The repository includes `api/index.go` (and root `main.go`) pre-configured with `package handler`, and `vercel.json` rewrites all requests to `/api`.

```bash
# Deploy with Vercel CLI
vercel
```

### 2. Cloudflare Workers (Edge JS)
Cloudflare Workers execute inside V8 isolates. The dedicated `worker.js` provides full zero-dependency feature parity with native Web Crypto and `DecompressionStream`.

```bash
# Deploy with Wrangler CLI
npx wrangler deploy
```

Or copy and paste the contents of `worker.js` directly into the [Cloudflare Workers Web Dashboard](https://dash.cloudflare.com/).

### 3. Netlify Functions (Go)
Configured using `netlify.toml`:

```bash
netlify deploy --prod
```

### 4. Running Locally / Standalone
To run locally on your machine or inside a Docker container:

```bash
go run cmd/server/main.go
# Or with a custom port:
PORT=8080 go run cmd/server/main.go
```

---

## API Endpoints & Usage

### 1. Movie Stream API
Fetch the stream URL for a specific movie using its TMDB ID.

**Endpoints:**
- `GET /api/movie/{tmdb_id}`
- `GET /movie/{tmdb_id}`
- `GET /?type=movie&id={tmdb_id}`

**Example Request:**
```bash
curl -s http://localhost:1937/api/movie/550
```

**Example Response:**
```json
{
  "id": "550",
  "name": "Fight Club",
  "url": "http://filmin.a0p3.com/vod/1/2025/04/03/604be407c1fe/1135529346.mp4?wsSecret=...&wsTime=..."
}
```

---

### 2. TV Show Stream API
Fetch the stream URL for a specific TV show episode using its TMDB ID, Season number, and Episode number.

**Endpoints:**
- `GET /api/tv/{tmdb_id}/{season}/{episode}`
- `GET /tv/{tmdb_id}/{season}/{episode}`
- `GET /?type=tv&id={tmdb_id}&season={season}&episode={episode}`

**Example Request:**
```bash
curl -s http://localhost:1937/api/tv/1399/1/6
```

**Example Response:**
```json
{
  "episode": "6",
  "id": "1399",
  "name": "Game of Thrones",
  "season": "1",
  "url": "http://filmin.a0p3.com/vod/1/2026/06/18/60565ddb818f/534473569.mp4?wsSecret=...&wsTime=..."
}
```

---

### 3. TMDB Metadata
- `GET /api/tmdb/movie/{tmdb_id}`

### 4. Health Check / Status
- `GET /` - HTML dashboard and endpoint list
- `GET /api/test` - Debug test verifying upstream connectivity

---

## Project Structure
```
├── api/
│   └── index.go        # Vercel entrypoint (package handler)
├── cmd/
│   └── server/
│       └── main.go     # Local / Docker standalone server runner
├── main.go             # Serverless handler (package handler)
├── worker.js           # Cloudflare Workers standalone script
├── vercel.json         # Vercel rewrites configuration
├── wrangler.toml       # Cloudflare Workers configuration
├── netlify.toml        # Netlify Functions build configuration
└── go.mod              # Go module definition
```

---

## Environment Variables
- `PORT`: (Optional) Port for local/standalone execution (default: `1937`).
- `TMDB_KEY`: (Optional) Custom The Movie Database API key (default key included).
- `DEVICE_ID`: (Optional) Custom pre-registered emulated device ID.
- `TOKEN`: (Optional) Custom upstream session token.
