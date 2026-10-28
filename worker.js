/**
 * NanoServer Universal Stream Worker
 * Compatible with: Cloudflare Workers, Vercel Edge, Netlify Edge
 */

const USER_AGENT = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36";
const SECRET = "47Q8tBqO4YqrMHf4";
const AES_KEY_STR = "0123456789123456";
const AES_IV_STR = "2015030120123456";
const WS_SECRET = "00b5f05c40b4f1d91dbc9b3fd8a059ef";
const MAIN_URL = "https://filmin.ajfysu.com";
const HOST_HEADER = "filmin.ajfysu.com";
const APP_ID = "filmin";
const CHANNEL_CODE = "filmin_sh_1000";
const PACKAGE_NAME = "com.dramarush.shortin";
const GAID = "";
const P2P_SALT = "Zox882LYjEn4Rqpa";
const DEFAULT_TMDB_KEY = "e6333b32409e02a4a6eba6fb7ff866bb";

const BRAND_MODELS = {
  Samsung: ["SM-S918B", "SM-A528B", "SM-M336B"],
  Xiaomi: ["2201117TI", "M2012K11AI", "Redmi Note 11"],
  OnePlus: ["LE2111", "CPH2449", "IN2023"],
  Google: ["Pixel 6", "Pixel 7", "Pixel 8"],
  Realme: ["RMX3085", "RMX3360", "RMX3551"],
};

function randomHex(bytesLen) {
  const bytes = new Uint8Array(bytesLen);
  crypto.getRandomValues(bytes);
  return Array.from(bytes, (b) => b.toString(16).padStart(2, "0")).join("");
}

const deviceId = randomHex(16);
const brandKeys = Object.keys(BRAND_MODELS);
const mobMfr = brandKeys[Math.floor(Math.random() * brandKeys.length)];
const models = BRAND_MODELS[mobMfr];
const mobModel = models[Math.floor(Math.random() * models.length)];

console.log(`[BOOT] NanoServer Worker initialized. Emulated Device: ${mobMfr} ${mobModel} (ID: ${deviceId})`);

let cachedToken = "";

// ─── MD5 Implementation (RFC 1321 standard, zero dependency) ─────────────────

function md5(string) {
  function rotateLeft(lValue, iShiftBits) {
    return (lValue << iShiftBits) | (lValue >>> (32 - iShiftBits));
  }
  function addUnsigned(lX, lY) {
    var lX4, lY4, lX8, lY8, lResult;
    lX8 = lX & 0x80000000;
    lY8 = lY & 0x80000000;
    lX4 = lX & 0x40000000;
    lY4 = lY & 0x40000000;
    lResult = (lX & 0x3fffffff) + (lY & 0x3fffffff);
    if (lX4 & lY4) return lResult ^ 0x80000000 ^ lX8 ^ lY8;
    if (lX4 | lY4) {
      if (lResult & 0x40000000) return lResult ^ 0xc0000000 ^ lX8 ^ lY8;
      else return lResult ^ 0x40000000 ^ lX8 ^ lY8;
    } else {
      return lResult ^ lX8 ^ lY8;
    }
  }
  function F(x, y, z) { return (x & y) | (~x & z); }
  function G(x, y, z) { return (x & z) | (y & ~z); }
  function H(x, y, z) { return x ^ y ^ z; }
  function I(x, y, z) { return y ^ (x | ~z); }
  function FF(a, b, c, d, x, s, ac) {
    a = addUnsigned(a, addUnsigned(addUnsigned(F(b, c, d), x), ac));
    return addUnsigned(rotateLeft(a, s), b);
  }
  function GG(a, b, c, d, x, s, ac) {
    a = addUnsigned(a, addUnsigned(addUnsigned(G(b, c, d), x), ac));
    return addUnsigned(rotateLeft(a, s), b);
  }
  function HH(a, b, c, d, x, s, ac) {
    a = addUnsigned(a, addUnsigned(addUnsigned(H(b, c, d), x), ac));
    return addUnsigned(rotateLeft(a, s), b);
  }
  function II(a, b, c, d, x, s, ac) {
    a = addUnsigned(a, addUnsigned(addUnsigned(I(b, c, d), x), ac));
    return addUnsigned(rotateLeft(a, s), b);
  }
  function convertToWordArray(str) {
    var lWordCount;
    var lMessageLength = str.length;
    var lNumberOfWords_temp1 = lMessageLength + 8;
    var lNumberOfWords_temp2 = (lNumberOfWords_temp1 - (lNumberOfWords_temp1 % 64)) / 64;
    var lNumberOfWords = (lNumberOfWords_temp2 + 1) * 16;
    var lWordArray = Array(lNumberOfWords - 1);
    var lBytePosition = 0;
    var lByteCount = 0;
    while (lByteCount < lMessageLength) {
      lWordCount = (lByteCount - (lByteCount % 4)) / 4;
      lBytePosition = (lByteCount % 4) * 8;
      lWordArray[lWordCount] = lWordArray[lWordCount] | (str.charCodeAt(lByteCount) << lBytePosition);
      lByteCount++;
    }
    lWordCount = (lByteCount - (lByteCount % 4)) / 4;
    lBytePosition = (lByteCount % 4) * 8;
    lWordArray[lWordCount] = lWordArray[lWordCount] | (0x80 << lBytePosition);
    lWordArray[lNumberOfWords - 2] = lMessageLength << 3;
    lWordArray[lNumberOfWords - 1] = lMessageLength >>> 29;
    return lWordArray;
  }
  function wordToHex(lValue) {
    var WordToHexValue = "", WordToHexValue_temp = "", lByte, lCount;
    for (lCount = 0; lCount <= 3; lCount++) {
      lByte = (lValue >>> (lCount * 8)) & 255;
      WordToHexValue_temp = "0" + lByte.toString(16);
      WordToHexValue = WordToHexValue + WordToHexValue_temp.substr(WordToHexValue_temp.length - 2, 2);
    }
    return WordToHexValue;
  }

  var utf8Str = unescape(encodeURIComponent(string));
  var x = convertToWordArray(utf8Str);
  var a = 0x67452301, b = 0xefcdab89, c = 0x98badcfe, d = 0x10325476;
  var S11 = 7, S12 = 12, S13 = 17, S14 = 22;
  var S21 = 5, S22 = 9, S23 = 14, S24 = 20;
  var S31 = 4, S32 = 11, S33 = 16, S34 = 23;
  var S41 = 6, S42 = 10, S43 = 15, S44 = 21;

  for (var k = 0; k < x.length; k += 16) {
    var AA = a, BB = b, CC = c, DD = d;
    a = FF(a, b, c, d, x[k + 0], S11, 0xd76aa478);
    d = FF(d, a, b, c, x[k + 1], S12, 0xe8c7b756);
    c = FF(c, d, a, b, x[k + 2], S13, 0x242070db);
    b = FF(b, c, d, a, x[k + 3], S14, 0xc1bdceee);
    a = FF(a, b, c, d, x[k + 4], S11, 0xf57c0faf);
    d = FF(d, a, b, c, x[k + 5], S12, 0x4787c62a);
    c = FF(c, d, a, b, x[k + 6], S13, 0xa8304613);
    b = FF(b, c, d, a, x[k + 7], S14, 0xfd469501);
    a = FF(a, b, c, d, x[k + 8], S11, 0x698098d8);
    d = FF(d, a, b, c, x[k + 9], S12, 0x8b44f7af);
    c = FF(c, d, a, b, x[k + 10], S13, 0xffff5bb1);
    b = FF(b, c, d, a, x[k + 11], S14, 0x895cd7be);
    a = FF(a, b, c, d, x[k + 12], S11, 0x6b901122);
    d = FF(d, a, b, c, x[k + 13], S12, 0xfd987193);
    c = FF(c, d, a, b, x[k + 14], S13, 0xa679438e);
    b = FF(b, c, d, a, x[k + 15], S14, 0x49b40821);
    a = GG(a, b, c, d, x[k + 1], S21, 0xf61e2562);
    d = GG(d, a, b, c, x[k + 6], S22, 0xc040b340);
    c = GG(c, d, a, b, x[k + 11], S23, 0x265e5a51);
    b = GG(b, c, d, a, x[k + 0], S24, 0xe9b6c7aa);
    a = GG(a, b, c, d, x[k + 5], S21, 0xd62f105d);
    d = GG(d, a, b, c, x[k + 10], S22, 0x2441453);
    c = GG(c, d, a, b, x[k + 15], S23, 0xd8a1e681);
    b = GG(b, c, d, a, x[k + 4], S24, 0xe7d3fbc8);
    a = GG(a, b, c, d, x[k + 9], S21, 0x21e1cde6);
    d = GG(d, a, b, c, x[k + 14], S22, 0xc33707d6);
    c = GG(c, d, a, b, x[k + 3], S23, 0xf4d50d87);
    b = GG(b, c, d, a, x[k + 8], S24, 0x455a14ed);
    a = GG(a, b, c, d, x[k + 13], S21, 0xa9e3e905);
    d = GG(d, a, b, c, x[k + 2], S22, 0xfcefa3f8);
    c = GG(c, d, a, b, x[k + 7], S23, 0x676f02d9);
    b = GG(b, c, d, a, x[k + 12], S24, 0x8d2a4c8a);
    a = HH(a, b, c, d, x[k + 5], S31, 0xfffa3942);
    d = HH(d, a, b, c, x[k + 8], S32, 0x8771f681);
    c = HH(c, d, a, b, x[k + 11], S33, 0x6d9d6122);
    b = HH(b, c, d, a, x[k + 14], S34, 0xfde5380c);
    a = HH(a, b, c, d, x[k + 1], S31, 0xa4beea44);
    d = HH(d, a, b, c, x[k + 4], S32, 0x4bdecfa9);
    c = HH(c, d, a, b, x[k + 7], S33, 0xf6bb4b60);
    b = HH(b, c, d, a, x[k + 10], S34, 0xbebfbc70);
    a = HH(a, b, c, d, x[k + 13], S31, 0x289b7ec6);
    d = HH(d, a, b, c, x[k + 0], S32, 0xeaa127fa);
    c = HH(c, d, a, b, x[k + 3], S33, 0xd4ef3085);
    b = HH(b, c, d, a, x[k + 6], S34, 0x4881d05);
    a = HH(a, b, c, d, x[k + 9], S31, 0xd9d4d039);
    d = HH(d, a, b, c, x[k + 12], S32, 0xe6db99e5);
    c = HH(c, d, a, b, x[k + 15], S33, 0x1fa27cf8);
    b = HH(b, c, d, a, x[k + 2], S34, 0xc4ac5665);
    a = II(a, b, c, d, x[k + 0], S41, 0xf4292244);
    d = II(d, a, b, c, x[k + 7], S42, 0x432aff97);
    c = II(c, d, a, b, x[k + 14], S43, 0xab9423a7);
    b = II(b, c, d, a, x[k + 5], S44, 0xfc93a039);
    a = II(a, b, c, d, x[k + 12], S41, 0x655b59c3);
    d = II(d, a, b, c, x[k + 3], S42, 0x8f0ccc92);
    c = II(c, d, a, b, x[k + 10], S43, 0xffeff47d);
    b = II(b, c, d, a, x[k + 1], S44, 0x85845dd1);
    a = II(a, b, c, d, x[k + 8], S41, 0x6fa87e4f);
    d = II(d, a, b, c, x[k + 15], S42, 0xfe2ce6e0);
    c = II(c, d, a, b, x[k + 6], S43, 0xa3014314);
    b = II(b, c, d, a, x[k + 13], S44, 0x4e0811a1);
    a = II(a, b, c, d, x[k + 4], S41, 0xf7537e82);
    d = II(d, a, b, c, x[k + 11], S42, 0xbd3af235);
    c = II(c, d, a, b, x[k + 2], S43, 0x2ad7d2bb);
    b = II(b, c, d, a, x[k + 9], S44, 0xeb86d391);
    a = addUnsigned(a, AA);
    b = addUnsigned(b, BB);
    c = addUnsigned(c, CC);
    d = addUnsigned(d, DD);
  }
  return (wordToHex(a) + wordToHex(b) + wordToHex(c) + wordToHex(d)).toLowerCase();
}

// ─── Cryptographic & Signing Helpers ─────────────────────────────────────────

function generateSign(curTime) {
  return md5(SECRET + deviceId + curTime).toUpperCase();
}

function generateP2pToken(vodId, curTime) {
  return md5(P2P_SALT + deviceId + vodId + curTime).toUpperCase();
}

function signVideoUrl(videoUrl) {
  if (!videoUrl) return "";
  try {
    const u = new URL(videoUrl);
    const nowSec = Math.floor(Date.now() / 1000);
    const wsTime = (nowSec + 60).toString(16);
    const wsSecret = md5(WS_SECRET + u.pathname + wsTime);
    const sep = videoUrl.includes("?") ? "&" : "?";
    return `${videoUrl}${sep}wsSecret=${wsSecret}&wsTime=${wsTime}`;
  } catch (e) {
    return videoUrl;
  }
}

async function aesDecrypt(encryptedBase64) {
  if (!encryptedBase64 || typeof encryptedBase64 !== "string") return "";
  try {
    const cleanStr = encryptedBase64.trim();
    const binaryStr = atob(cleanStr);
    const bytes = new Uint8Array(binaryStr.length);
    for (let i = 0; i < binaryStr.length; i++) {
      bytes[i] = binaryStr.charCodeAt(i);
    }

    const enc = new TextEncoder();
    const keyBytes = enc.encode(AES_KEY_STR);
    const ivBytes = enc.encode(AES_IV_STR);

    const key = await crypto.subtle.importKey(
      "raw",
      keyBytes,
      { name: "AES-CBC" },
      false,
      ["decrypt"]
    );

    const decryptedBuffer = await crypto.subtle.decrypt(
      { name: "AES-CBC", iv: ivBytes },
      key,
      bytes
    );

    const decryptedBytes = new Uint8Array(decryptedBuffer);

    // If GZIP magic bytes (0x1f, 0x8b), decompress
    if (decryptedBytes.length >= 2 && decryptedBytes[0] === 0x1f && decryptedBytes[1] === 0x8b) {
      if (typeof DecompressionStream !== "undefined") {
        const ds = new DecompressionStream("gzip");
        const decompressedStream = new Response(decryptedBytes).body.pipeThrough(ds);
        return await new Response(decompressedStream).text();
      }
    }

    return new TextDecoder().decode(decryptedBytes);
  } catch (err) {
    console.error("[DECRYPT ERROR]", err);
    return "";
  }
}

// ─── Upstream API Helpers ─────────────────────────────────────────────────────

function buildHeaders(curTime, token) {
  return {
    "Accept-Encoding": "identity",
    androidid: deviceId,
    app_id: APP_ID,
    app_language: "en",
    channel_code: CHANNEL_CODE,
    Connection: "Keep-Alive",
    "Content-Type": "application/x-www-form-urlencoded",
    cur_time: curTime,
    device_id: deviceId,
    en_al: "0",
    gaid: GAID,
    Host: HOST_HEADER,
    is_display: "GMT+05:30",
    is_language: "en",
    is_vvv: "0",
    "log-header": "I am the log request header.",
    mob_mfr: mobMfr,
    mobmodel: mobModel,
    package_name: PACKAGE_NAME,
    sign: generateSign(curTime),
    sys_platform: "2",
    sysrelease: "13",
    token: token || "",
    "User-Agent": "okhttp/4.11.0",
    version: "30000",
  };
}

async function httpsPost(endpoint, formData, headers) {
  const startTime = Date.now();
  try {
    const params = new URLSearchParams();
    for (const [k, v] of Object.entries(formData)) {
      params.append(k, v);
    }

    console.log(`[UPSTREAM POST] ${MAIN_URL}${endpoint} (data keys: ${Object.keys(formData).join(", ")})`);
    const res = await fetch(MAIN_URL + endpoint, {
      method: "POST",
      headers: headers,
      body: params.toString(),
    });

    const elapsed = Date.now() - startTime;
    console.log(`[UPSTREAM POST] ${endpoint} -> Status ${res.status} (${elapsed}ms)`);
    if (!res.ok) {
      console.error(`[UPSTREAM POST ERROR] ${endpoint} returned HTTP ${res.status}`);
      return "";
    }
    return await res.text();
  } catch (e) {
    console.error(`[UPSTREAM POST EXCEPTION] ${endpoint}:`, e.message || e);
    return "";
  }
}

async function fetchToken() {
  if (cachedToken) {
    console.log(`[AUTH] Using cached session token (${cachedToken.substring(0, 8)}...)`);
    return cachedToken;
  }

  console.log(`[AUTH] Initializing new session token via /api/public/init...`);
  const curTime = Date.now().toString();
  const headers = buildHeaders(curTime, "");
  const text = await httpsPost("/api/public/init", { invited_by: "", is_install: "1" }, headers);
  if (!text) {
    console.error(`[AUTH ERROR] Upstream /api/public/init returned empty body`);
    return "";
  }

  let jsonStr = text.trim();
  if (!jsonStr.startsWith("{")) {
    jsonStr = await aesDecrypt(jsonStr);
  }

  try {
    const parsed = JSON.parse(jsonStr);
    const token = parsed?.result?.user_info?.token;
    if (token) {
      cachedToken = token;
      console.log(`[AUTH SUCCESS] New session token acquired: ${cachedToken.substring(0, 10)}...`);
      return cachedToken;
    }
    console.error(`[AUTH ERROR] Response missing user_info.token:`, jsonStr.substring(0, 200));
  } catch (e) {
    console.error(`[AUTH ERROR] JSON parse failure:`, e.message);
  }
  return "";
}

async function apiPost(endpoint, formData) {
  let token = await fetchToken();
  let curTime = Date.now().toString();
  let resText = await httpsPost(endpoint, formData, buildHeaders(curTime, token));
  if (!resText) return null;

  let dec = await aesDecrypt(resText.trim());
  try {
    let result = JSON.parse(dec);
    if (result?.code === 401) {
      console.warn(`[API] Token unauthorized (code 401). Retrying with fresh token...`);
      cachedToken = "";
      token = await fetchToken();
      curTime = Date.now().toString();
      resText = await httpsPost(endpoint, formData, buildHeaders(curTime, token));
      if (resText) {
        dec = await aesDecrypt(resText.trim());
        result = JSON.parse(dec);
      }
    }
    return result;
  } catch (e) {
    console.error(`[API ERROR] Decrypt/parse failed for ${endpoint}:`, e.message);
    return null;
  }
}

async function getVodInfo(vodId, audioType = 0) {
  console.log(`[VOD] Requesting vod details for ID: ${vodId}...`);
  let token = await fetchToken();
  let curTime = Date.now().toString();
  let resText = await httpsPost(
    "/api/vod/info_new",
    {
      sign: generateP2pToken(vodId, curTime),
      vod_id: vodId,
      cur_time: curTime,
      audio_type: audioType.toString(),
    },
    buildHeaders(curTime, token)
  );

  if (!resText) {
    console.error(`[VOD ERROR] Empty response from /api/vod/info_new for ${vodId}`);
    return null;
  }
  let dec = await aesDecrypt(resText.trim());
  try {
    let result = JSON.parse(dec);
    if (result?.code === 401) {
      console.warn(`[VOD] Token 401. Refreshing token...`);
      cachedToken = "";
      token = await fetchToken();
      curTime = Date.now().toString();
      resText = await httpsPost(
        "/api/vod/info_new",
        {
          sign: generateP2pToken(vodId, curTime),
          vod_id: vodId,
          cur_time: curTime,
          audio_type: audioType.toString(),
        },
        buildHeaders(curTime, token)
      );
      if (resText) {
        dec = await aesDecrypt(resText.trim());
        result = JSON.parse(dec);
      }
    }

    if (result?.result?.vod_collection) {
      const count = result.result.vod_collection.length;
      console.log(`[VOD SUCCESS] Found ${count} stream item(s) for VOD ID ${vodId}`);
      for (const ep of result.result.vod_collection) {
        const rawUrl = ep.vod_url || ep.down_url || "";
        ep.raw_url = rawUrl;
        ep.signed_url = signVideoUrl(rawUrl);
        if (ep.vod_url) ep.vod_url = signVideoUrl(ep.vod_url);
        if (ep.down_url) ep.down_url = signVideoUrl(ep.down_url);
      }
    } else {
      console.warn(`[VOD WARNING] No vod_collection found in response:`, JSON.stringify(result).substring(0, 200));
    }
    return result;
  } catch (e) {
    console.error(`[VOD ERROR] Parse failure:`, e.message);
    return null;
  }
}

function extractVodId(res) {
  if (!res) return "";
  if (Array.isArray(res.result) && res.result.length > 0) {
    const item = res.result[0];
    return item?.id ? item.id.toString() : "";
  }
  if (res.result?.items && Array.isArray(res.result.items) && res.result.items.length > 0) {
    const item = res.result.items[0];
    return item?.id ? item.id.toString() : "";
  }
  return "";
}

// ─── TMDB Metadata Helpers ───────────────────────────────────────────────────

function parseTmdbMap(data, mediaType) {
  const title = data.title || data.name || data.original_title || data.original_name || "";
  const date = data.release_date || data.first_air_date || "";
  const year = date.includes("-") ? date.split("-")[0] : date;
  return {
    id: data.id,
    title: title,
    year: year,
    type: mediaType,
  };
}

async function fetchTmdbDetails(id, mediaType, apiKey) {
  const key = apiKey || DEFAULT_TMDB_KEY;
  console.log(`[TMDB] Fetching details for ${mediaType} ID: ${id}...`);
  if (id.startsWith("tt")) {
    const findUrl = `https://api.themoviedb.org/3/find/${id}?api_key=${key}&external_source=imdb_id`;
    try {
      const resp = await fetch(findUrl, { headers: { "User-Agent": USER_AGENT } });
      if (resp.ok) {
        const findRes = await resp.json();
        const tvResults = findRes.tv_results || [];
        const movieResults = findRes.movie_results || [];
        if (mediaType === "tv" && tvResults.length > 0) {
          const res = parseTmdbMap(tvResults[0], "tv");
          console.log(`[TMDB SUCCESS] Resolved IMDb ${id} -> TV "${res.title}" (${res.year})`);
          return res;
        } else if (movieResults.length > 0) {
          const res = parseTmdbMap(movieResults[0], "movie");
          console.log(`[TMDB SUCCESS] Resolved IMDb ${id} -> Movie "${res.title}" (${res.year})`);
          return res;
        } else if (tvResults.length > 0) {
          const res = parseTmdbMap(tvResults[0], "tv");
          console.log(`[TMDB SUCCESS] Resolved IMDb ${id} -> TV "${res.title}" (${res.year})`);
          return res;
        }
      }
    } catch (e) {
      console.error(`[TMDB ERROR] IMDb lookup failed:`, e.message);
    }
  }

  if (mediaType === "movie" || mediaType === "tv") {
    const reqUrl = `https://api.themoviedb.org/3/${mediaType}/${id}?api_key=${key}`;
    try {
      const resp = await fetch(reqUrl, { headers: { "User-Agent": USER_AGENT } });
      if (resp.ok) {
        const data = await resp.json();
        const res = parseTmdbMap(data, mediaType);
        console.log(`[TMDB SUCCESS] Resolved TMDB ${id} -> "${res.title}" (${res.year})`);
        return res;
      }
      console.error(`[TMDB ERROR] HTTP ${resp.status} for ${reqUrl}`);
    } catch (e) {
      console.error(`[TMDB ERROR] TMDB API lookup failed:`, e.message);
    }
  }
  return null;
}

// ─── Response Helpers ────────────────────────────────────────────────────────

const corsHeaders = {
  "Access-Control-Allow-Origin": "*",
  "Access-Control-Allow-Methods": "GET, POST, OPTIONS, HEAD",
  "Access-Control-Allow-Headers": "*",
  "Access-Control-Expose-Headers": "*",
};

function jsonResponse(data, status = 200) {
  return new Response(JSON.stringify(data), {
    status: status,
    headers: {
      "Content-Type": "application/json; charset=utf-8",
      ...corsHeaders,
    },
  });
}

function errorResponse(message, status = 400) {
  console.warn(`[HTTP ${status}] Returning error: ${message}`);
  return jsonResponse({ error: message }, status);
}

// ─── Universal Worker Handler ─────────────────────────────────────────────────

export default {
  async fetch(request, env = {}, ctx = {}) {
    const startTime = Date.now();
    const url = new URL(request.url);
    console.log(`[REQ] ${request.method} ${url.pathname}${url.search}`);

    if (request.method === "OPTIONS") {
      return new Response(null, { status: 204, headers: corsHeaders });
    }

    const rawPath = url.pathname;
    let cleanPath = rawPath.replace(/^\/+|\/+$/g, "");
    if (cleanPath.startsWith("api/")) {
      cleanPath = cleanPath.slice(4);
    }

    const tmdbApiKey = env.TMDB_KEY || DEFAULT_TMDB_KEY;
    const reqType = url.searchParams.get("type");
    const queryId = url.searchParams.get("id");

    // Diagnostic health check
    if (cleanPath === "health" || cleanPath === "debug") {
      const token = await fetchToken();
      const tmdbCheck = await fetchTmdbDetails("550", "movie", tmdbApiKey);
      return jsonResponse({
        status: "ok",
        timestamp: new Date().toISOString(),
        emulated_device: `${mobMfr} ${mobModel} (${deviceId})`,
        upstream: {
          host: MAIN_URL,
          token_acquired: Boolean(token),
          token_preview: token ? `${token.substring(0, 8)}...` : null,
        },
        tmdb: {
          connected: Boolean(tmdbCheck),
          sample_title: tmdbCheck?.title || null,
        },
      });
    }

    // 1. Query parameter routing (?type=movie&id=550 or ?type=tv&id=1399&season=1&episode=6)
    if (reqType === "movie" && queryId) {
      return await handleMovieRequest(queryId, tmdbApiKey);
    }
    if (reqType === "tv" && queryId) {
      const season = url.searchParams.get("season") || url.searchParams.get("s") || "1";
      const episode = parseInt(url.searchParams.get("episode") || url.searchParams.get("e") || "1", 10);
      return await handleTvRequest(queryId, season, episode, tmdbApiKey);
    }

    // 2. Path-based routing
    if (cleanPath === "" || cleanPath === "api") {
      const html = `<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>NanoServer Stream Worker</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; padding: 2rem; max-width: 800px; margin: auto; line-height: 1.6; }
        code { background: #f4f4f4; padding: 2px 6px; border-radius: 4px; }
        .badge { display: inline-block; background: #10b981; color: white; padding: 4px 8px; border-radius: 4px; font-weight: bold; }
    </style>
</head>
<body>
    <h1>NanoServer Stream Worker</h1>
    <p><span class="badge">Running</span> Serverless Edge Worker (Cloudflare / Vercel / Netlify)</p>
    <h2>API Endpoints</h2>
    <ul>
        <li><code>GET /api/movie/{tmdb_id}</code> - Resolve movie stream (e.g. <code>/api/movie/550</code>)</li>
        <li><code>GET /api/tv/{tmdb_id}/{season}/{episode}</code> - Resolve TV episode stream (e.g. <code>/api/tv/1399/1/6</code>)</li>
        <li><code>GET /api/tmdb/movie/{tmdb_id}</code> - Fetch TMDB metadata</li>
        <li><code>GET /api/health</code> - Diagnostics & connectivity test</li>
        <li><code>GET /api/test</code> - Upstream connection test</li>
    </ul>
</body>
</html>`;
      return new Response(html, {
        headers: { "Content-Type": "text/html; charset=utf-8", ...corsHeaders },
      });
    }

    if (cleanPath === "test") {
      const vodDetails = await getVodInfo("1466695562", 0);
      if (!vodDetails) return errorResponse("Upstream connection test failed", 502);
      return jsonResponse(vodDetails);
    }

    if (cleanPath.startsWith("tmdb/movie/")) {
      const id = cleanPath.slice("tmdb/movie/".length) || "550";
      const tmdb = await fetchTmdbDetails(id, "movie", tmdbApiKey);
      if (!tmdb) return errorResponse("TMDB info not found", 404);
      return jsonResponse(tmdb);
    }

    if (cleanPath.startsWith("movie/")) {
      const id = cleanPath.slice("movie/".length) || "550";
      return await handleMovieRequest(id, tmdbApiKey);
    }

    if (cleanPath.startsWith("tv/")) {
      const parts = cleanPath.slice("tv/".length).split("/");
      if (parts.length >= 3) {
        const id = parts[0];
        const season = parts[1];
        const episode = parseInt(parts[2], 10);
        return await handleTvRequest(id, season, episode, tmdbApiKey);
      }
      return errorResponse("Invalid format, use /api/tv/{id}/{season}/{episode}", 400);
    }

    // Fallback: check query param id
    if (queryId) {
      return await handleMovieRequest(queryId, tmdbApiKey);
    }

    return errorResponse("Endpoint not found", 404);
  },
};

async function handleMovieRequest(id, tmdbApiKey) {
  console.log(`[MOVIE] Resolving stream for movie ID: ${id}`);
  const tmdb = await fetchTmdbDetails(id, "movie", tmdbApiKey);
  if (!tmdb) {
    console.error(`[MOVIE ERROR] TMDB details not found for ID: ${id}`);
    return errorResponse("TMDB info not found", 404);
  }

  const title = tmdb.Title || tmdb.title;
  console.log(`[MOVIE SEARCH] Querying upstream for: "${title}"`);
  let res = await apiPost("/api/search/result", { kw: title, pn: "1" });
  let vodId = extractVodId(res);

  if (!vodId && (title.includes(":") || title.includes("-"))) {
    const simpleTitle = title.split(":")[0].split("-")[0].trim();
    console.log(`[MOVIE FALLBACK] Retrying search with simplified title: "${simpleTitle}"`);
    res = await apiPost("/api/search/result", { kw: simpleTitle, pn: "1" });
    vodId = extractVodId(res);
  }

  if (!vodId) {
    console.warn(`[MOVIE NOT FOUND] Upstream search returned no results for "${title}"`);
    return errorResponse("Video not found on server", 404);
  }

  console.log(`[MOVIE FOUND] Upstream VOD ID: ${vodId}. Fetching stream URL...`);
  const vodDetails = await getVodInfo(vodId, 0);
  let signedUrl = "";
  if (vodDetails?.result?.vod_collection && vodDetails.result.vod_collection.length > 0) {
    signedUrl = vodDetails.result.vod_collection[0]?.signed_url || "";
  }

  if (!signedUrl) {
    console.error(`[MOVIE ERROR] Stream URL not available for VOD ID ${vodId}`);
    return errorResponse("Stream URL not available", 404);
  }

  console.log(`[MOVIE SUCCESS] Resolved stream URL for "${title}": ${signedUrl.substring(0, 60)}...`);
  return jsonResponse({
    id: id,
    name: title,
    url: signedUrl,
  });
}

async function handleTvRequest(id, season, episode, tmdbApiKey) {
  console.log(`[TV] Resolving stream for TV show ID: ${id}, Season ${season}, Episode ${episode}`);
  const tmdb = await fetchTmdbDetails(id, "tv", tmdbApiKey);
  if (!tmdb) {
    console.error(`[TV ERROR] TMDB details not found for TV ID: ${id}`);
    return errorResponse("TMDB info not found", 404);
  }

  const title = tmdb.Title || tmdb.title;
  const searchQuery = `${title} Season ${season}`;
  console.log(`[TV SEARCH] Querying upstream for: "${searchQuery}"`);
  let res = await apiPost("/api/search/result", { kw: searchQuery, pn: "1" });
  let vodId = extractVodId(res);

  if (!vodId) {
    console.log(`[TV FALLBACK] Retrying search with title only: "${title}"`);
    res = await apiPost("/api/search/result", { kw: title, pn: "1" });
    vodId = extractVodId(res);
  }

  if (!vodId) {
    console.warn(`[TV NOT FOUND] Upstream search returned no results for "${searchQuery}"`);
    return errorResponse("Video not found on server", 404);
  }

  console.log(`[TV FOUND] Upstream VOD ID: ${vodId}. Fetching episode ${episode}...`);
  const vodDetails = await getVodInfo(vodId, 0);
  let signedUrl = "";
  if (vodDetails?.result?.vod_collection) {
    for (const ep of vodDetails.result.vod_collection) {
      const colNum = parseInt(ep.collection, 10);
      if (colNum === episode) {
        signedUrl = ep.signed_url || "";
        break;
      }
    }
  }

  if (!signedUrl) {
    console.error(`[TV ERROR] Episode ${episode} not found in collection for VOD ID ${vodId}`);
    return errorResponse("Episode not found", 404);
  }

  console.log(`[TV SUCCESS] Resolved stream URL for "${title}" S${season}E${episode}: ${signedUrl.substring(0, 60)}...`);
  return jsonResponse({
    id: id,
    name: title,
    season: season,
    episode: episode.toString(),
    url: signedUrl,
  });
}
