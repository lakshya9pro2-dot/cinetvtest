package handler

import (
	"bytes"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"crypto/des"
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	mathrand "math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	TAG                  = "NanoServer"
	DefaultPort          = 1937
	UserAgent            = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	SECRET_KEY_ENCRYPTED = "MxASAkl/yHTGg+/Tw1R7u96nGqkWsOZ2"
	DES_KEY              = "dsawdf634eebGFHITR5UT9kS0"
	DES_IV               = "32456738"
	AES_KEY_STR          = "0123456789123456"
	AES_IV_STR           = "2015030120123456"
	WS_SECRET            = "00b5f05c40b4f1d91dbc9b3fd8a059ef"
	MAIN_URL             = "https://filmin.ajfysu.com"
	HOST_HEADER          = "filmin.ajfysu.com"
	APP_ID               = "filmin"
	CHANNEL_CODE         = "filmin_sh_1000"
	PACKAGE_NAME         = "com.dramarush.shortin"
	GAID                 = ""
	P2P_SALT             = "Zox882LYjEn4Rqpa"
	TMDB_KEY             = "e6333b32409e02a4a6eba6fb7ff866bb"
)

var (
	brandModels = map[string][]string{
		"Samsung": {"SM-S918B", "SM-A528B", "SM-M336B"},
		"Xiaomi":  {"2201117TI", "M2012K11AI", "Redmi Note 11"},
		"OnePlus": {"LE2111", "CPH2449", "IN2023"},
		"Google":  {"Pixel 6", "Pixel 7", "Pixel 8"},
		"Realme":  {"RMX3085", "RMX3360", "RMX3551"},
	}
	deviceId    string
	mobMfr      string
	mobModel    string
	cachedToken string
	tokenLock   sync.Mutex
)

func init() {
	b := make([]byte, 16)
	rand.Read(b)
	deviceId = hex.EncodeToString(b)

	var keys []string
	for k := range brandModels {
		keys = append(keys, k)
	}
	mobMfr = keys[mathrand.Intn(len(keys))]
	models := brandModels[mobMfr]
	mobModel = models[mathrand.Intn(len(models))]
}

type TmdbInfo struct {
	ID      int    `json:"id"`
	Title   string `json:"title"`
	Year    string `json:"year"`
	Type    string `json:"type"`
	Season  *int   `json:"season,omitempty"`
	Episode *int   `json:"episode,omitempty"`
}

func md5Hex(text string) string {
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}

func des3Decrypt(encryptedBase64 string) string {
	data, err := base64.StdEncoding.DecodeString(encryptedBase64)
	if err != nil {
		return ""
	}
	keyBytes := make([]byte, 24)
	desKeyBytes := []byte(DES_KEY)
	copy(keyBytes, desKeyBytes)

	block, err := des.NewTripleDESCipher(keyBytes)
	if err != nil {
		return ""
	}

	mode := cipher.NewCBCDecrypter(block, []byte(DES_IV))
	decrypted := make([]byte, len(data))
	mode.CryptBlocks(decrypted, data)

	// Unpad PKCS5
	if len(decrypted) == 0 {
		return ""
	}
	padding := int(decrypted[len(decrypted)-1])
	if padding > 0 && padding <= 8 && len(decrypted) >= padding {
		decrypted = decrypted[:len(decrypted)-padding]
	}

	return string(decrypted)
}

func generateSign(curTime string) string {
	secret := des3Decrypt(SECRET_KEY_ENCRYPTED)
	return strings.ToUpper(md5Hex(secret + deviceId + curTime))
}

func generateP2pToken(vodId, timestamp string) string {
	return strings.ToUpper(md5Hex(P2P_SALT + deviceId + vodId + timestamp))
}

func signVideoUrl(videoUrl string) string {
	if videoUrl == "" {
		return ""
	}
	u, err := url.Parse(videoUrl)
	if err != nil {
		return videoUrl
	}
	nowSec := time.Now().Unix()
	wsTime := strconv.FormatInt(nowSec+60, 16)
	wsSecret := md5Hex(WS_SECRET + u.Path + wsTime)
	sep := "?"
	if strings.Contains(videoUrl, "?") {
		sep = "&"
	}
	return fmt.Sprintf("%s%swsSecret=%s&wsTime=%s", videoUrl, sep, wsSecret, wsTime)
}

func aesDecrypt(encryptedBase64 string) string {
	data, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encryptedBase64))
	if err != nil {
		return ""
	}

	block, err := aes.NewCipher([]byte(AES_KEY_STR))
	if err != nil {
		return ""
	}

	if len(data)%aes.BlockSize != 0 || len(data) == 0 {
		return ""
	}

	mode := cipher.NewCBCDecrypter(block, []byte(AES_IV_STR))
	unpadded := make([]byte, len(data))
	mode.CryptBlocks(unpadded, data)

	// Unpad PKCS5
	padding := int(unpadded[len(unpadded)-1])
	if padding > 0 && padding <= 16 && len(unpadded) >= padding {
		unpadded = unpadded[:len(unpadded)-padding]
	}

	if len(unpadded) >= 2 && unpadded[0] == 0x1f && unpadded[1] == 0x8b {
		gzipReader, err := gzip.NewReader(bytes.NewReader(unpadded))
		if err == nil {
			defer gzipReader.Close()
			decompressed, err := io.ReadAll(gzipReader)
			if err == nil {
				return string(decompressed)
			}
		}
	}

	return string(unpadded)
}

func buildHeaders(curTime, token string) map[string]string {
	return map[string]string{
		"Accept-Encoding": "identity",
		"androidid":       deviceId,
		"app_id":          APP_ID,
		"app_language":    "en",
		"channel_code":    CHANNEL_CODE,
		"Connection":      "Keep-Alive",
		"Content-Type":    "application/x-www-form-urlencoded",
		"cur_time":        curTime,
		"device_id":       deviceId,
		"en_al":           "0",
		"gaid":            GAID,
		"Host":            HOST_HEADER,
		"is_display":      "GMT+05:30",
		"is_language":     "en",
		"is_vvv":          "0",
		"log-header":      "I am the log request header.",
		"mob_mfr":         mobMfr,
		"mobmodel":        mobModel,
		"package_name":    PACKAGE_NAME,
		"sign":            generateSign(curTime),
		"sys_platform":    "2",
		"sysrelease":      "13",
		"token":           token,
		"User-Agent":      "okhttp/4.11.0",
		"version":         "30000",
	}
}

func httpsPost(endpoint string, formData, headers map[string]string) []byte {
	u := MAIN_URL + endpoint
	data := url.Values{}
	for k, v := range formData {
		data.Set(k, v)
	}

	req, err := http.NewRequest("POST", u, strings.NewReader(data.Encode()))
	if err != nil {
		return nil
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	return body
}

func invalidateToken() {
	tokenLock.Lock()
	defer tokenLock.Unlock()
	cachedToken = ""
}

func fetchToken() string {
	tokenLock.Lock()
	defer tokenLock.Unlock()

	if cachedToken != "" {
		return cachedToken
	}

	curTime := strconv.FormatInt(time.Now().UnixMilli(), 10)
	headers := buildHeaders(curTime, "")
	buf := httpsPost("/api/public/init", map[string]string{"invited_by": "", "is_install": "1"}, headers)
	if len(buf) == 0 {
		return ""
	}

	text := strings.TrimSpace(string(buf))
	jsonStr := text
	if !strings.HasPrefix(text, "{") {
		jsonStr = aesDecrypt(text)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err == nil {
		if result, ok := parsed["result"].(map[string]interface{}); ok {
			if userInfo, ok := result["user_info"].(map[string]interface{}); ok {
				if t, ok := userInfo["token"].(string); ok && t != "" {
					cachedToken = t
					return cachedToken
				}
			}
		}
	}
	return ""
}

func apiPost(endpoint string, formData map[string]string) map[string]interface{} {
	token := fetchToken()
	curTime := strconv.FormatInt(time.Now().UnixMilli(), 10)
	buf := httpsPost(endpoint, formData, buildHeaders(curTime, token))
	if len(buf) == 0 {
		return nil
	}

	dec := aesDecrypt(strings.TrimSpace(string(buf)))
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(dec), &result); err != nil {
		return nil
	}

	// Retry once on unauthorized token
	if code, ok := result["code"].(float64); ok && code == 401 {
		invalidateToken()
		token = fetchToken()
		curTime = strconv.FormatInt(time.Now().UnixMilli(), 10)
		buf = httpsPost(endpoint, formData, buildHeaders(curTime, token))
		if len(buf) > 0 {
			dec = aesDecrypt(strings.TrimSpace(string(buf)))
			json.Unmarshal([]byte(dec), &result)
		}
	}

	return result
}

func getVodInfo(vodId string, audioType int) map[string]interface{} {
	token := fetchToken()
	curTime := strconv.FormatInt(time.Now().UnixMilli(), 10)
	buf := httpsPost(
		"/api/vod/info_new",
		map[string]string{
			"sign":       generateP2pToken(vodId, curTime),
			"vod_id":     vodId,
			"cur_time":   curTime,
			"audio_type": strconv.Itoa(audioType),
		},
		buildHeaders(curTime, token),
	)
	if len(buf) == 0 {
		return nil
	}

	dec := aesDecrypt(strings.TrimSpace(string(buf)))
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(dec), &result); err != nil {
		return nil
	}

	// Retry once on unauthorized token
	if code, ok := result["code"].(float64); ok && code == 401 {
		invalidateToken()
		token = fetchToken()
		curTime = strconv.FormatInt(time.Now().UnixMilli(), 10)
		buf = httpsPost(
			"/api/vod/info_new",
			map[string]string{
				"sign":       generateP2pToken(vodId, curTime),
				"vod_id":     vodId,
				"cur_time":   curTime,
				"audio_type": strconv.Itoa(audioType),
			},
			buildHeaders(curTime, token),
		)
		if len(buf) > 0 {
			dec = aesDecrypt(strings.TrimSpace(string(buf)))
			json.Unmarshal([]byte(dec), &result)
		}
	}

	if resObj, ok := result["result"].(map[string]interface{}); ok {
		if collections, ok := resObj["vod_collection"].([]interface{}); ok {
			for _, epIntf := range collections {
				if ep, ok := epIntf.(map[string]interface{}); ok {
					rawUrl, _ := ep["vod_url"].(string)
					if rawUrl == "" {
						rawUrl, _ = ep["down_url"].(string)
					}
					ep["raw_url"] = rawUrl
					ep["signed_url"] = signVideoUrl(rawUrl)
					if vUrl, ok := ep["vod_url"].(string); ok && vUrl != "" {
						ep["vod_url"] = signVideoUrl(vUrl)
					}
					if dUrl, ok := ep["down_url"].(string); ok && dUrl != "" {
						ep["down_url"] = signVideoUrl(dUrl)
					}
				}
			}
		}
	}
	return result
}

func httpGet(urlStr string, headers map[string]string) string {
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("User-Agent", UserAgent)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return string(body)
}

func getTmdbApiKey() string {
	if key := os.Getenv("TMDB_KEY"); key != "" {
		return key
	}
	return TMDB_KEY
}

func parseTmdbMap(data map[string]interface{}, mediaType string) *TmdbInfo {
	idVal, _ := data["id"].(float64)

	titleVal, _ := data["title"].(string)
	if titleVal == "" {
		titleVal, _ = data["name"].(string)
	}
	if titleVal == "" {
		titleVal, _ = data["original_title"].(string)
	}
	if titleVal == "" {
		titleVal, _ = data["original_name"].(string)
	}

	dateVal, _ := data["release_date"].(string)
	if dateVal == "" {
		dateVal, _ = data["first_air_date"].(string)
	}

	yearVal := dateVal
	if strings.Contains(dateVal, "-") {
		yearVal = strings.Split(dateVal, "-")[0]
	}

	return &TmdbInfo{
		ID:    int(idVal),
		Title: titleVal,
		Year:  yearVal,
		Type:  mediaType,
	}
}

func fetchTmdbDetails(id string, mediaType string) *TmdbInfo {
	apiKey := getTmdbApiKey()
	if strings.HasPrefix(id, "tt") {
		findURL := fmt.Sprintf("https://api.themoviedb.org/3/find/%s?api_key=%s&external_source=imdb_id", id, apiKey)
		jsonStr := httpGet(findURL, nil)
		if jsonStr != "" {
			var findRes map[string]interface{}
			json.Unmarshal([]byte(jsonStr), &findRes)

			tvResults, _ := findRes["tv_results"].([]interface{})
			movieResults, _ := findRes["movie_results"].([]interface{})

			if mediaType == "tv" && len(tvResults) > 0 {
				return parseTmdbMap(tvResults[0].(map[string]interface{}), "tv")
			} else if len(movieResults) > 0 {
				return parseTmdbMap(movieResults[0].(map[string]interface{}), "movie")
			} else if len(tvResults) > 0 {
				return parseTmdbMap(tvResults[0].(map[string]interface{}), "tv")
			}
		}
	}

	if mediaType == "movie" || mediaType == "tv" {
		reqURL := fmt.Sprintf("https://api.themoviedb.org/3/%s/%s?api_key=%s", mediaType, id, apiKey)
		jsonStr := httpGet(reqURL, nil)
		if jsonStr != "" {
			var data map[string]interface{}
			json.Unmarshal([]byte(jsonStr), &data)
			return parseTmdbMap(data, mediaType)
		}
	}

	return nil
}

func addCorsHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, HEAD")
	w.Header().Set("Access-Control-Allow-Headers", "*")
	w.Header().Set("Access-Control-Expose-Headers", "*")
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func extractVodId(response map[string]interface{}) string {
	if response == nil {
		return ""
	}
	if items, ok := response["result"].([]interface{}); ok && len(items) > 0 {
		if item, ok := items[0].(map[string]interface{}); ok {
			if idFloat, ok := item["id"].(float64); ok {
				return strconv.FormatFloat(idFloat, 'f', -1, 64)
			} else if idStr, ok := item["id"].(string); ok {
				return idStr
			}
		}
	} else if resultMap, ok := response["result"].(map[string]interface{}); ok {
		if items, ok := resultMap["items"].([]interface{}); ok && len(items) > 0 {
			if item, ok := items[0].(map[string]interface{}); ok {
				if idFloat, ok := item["id"].(float64); ok {
					return strconv.FormatFloat(idFloat, 'f', -1, 64)
				} else if idStr, ok := item["id"].(string); ok {
					return idStr
				}
			}
		}
	}
	return ""
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	html := `<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>NanoServer Stream Worker</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; padding: 2rem; max-width: 800px; margin: auto; line-height: 1.6; }
        code { background: #f4f4f4; padding: 2px 6px; border-radius: 4px; }
        pre { background: #f4f4f4; padding: 12px; border-radius: 6px; overflow-x: auto; }
        .badge { display: inline-block; background: #10b981; color: white; padding: 4px 8px; border-radius: 4px; font-weight: bold; }
    </style>
</head>
<body>
    <h1>NanoServer Stream Worker</h1>
    <p><span class="badge">Running</span> Serverless Worker Ready (Vercel / Netlify / Cloudflare)</p>
    <h2>API Endpoints</h2>
    <ul>
        <li><code>GET /api/movie/{tmdb_id}</code> - Resolve movie stream (e.g., <code>/api/movie/550</code>)</li>
        <li><code>GET /api/tv/{tmdb_id}/{season}/{episode}</code> - Resolve TV episode stream (e.g., <code>/api/tv/1399/1/6</code>)</li>
        <li><code>GET /api/tmdb/movie/{tmdb_id}</code> - Fetch TMDB metadata</li>
        <li><code>GET /api/test</code> - Upstream connection test</li>
    </ul>
    <h3>Query Parameter Alternatives</h3>
    <ul>
        <li><code>?type=movie&id=550</code></li>
        <li><code>?type=tv&id=1399&season=1&episode=6</code></li>
    </ul>
</body>
</html>`
	w.Write([]byte(html))
}

func handleTmdbWithID(w http.ResponseWriter, r *http.Request, id string) {
	if id == "" {
		id = "550"
	}
	details := fetchTmdbDetails(id, "movie")
	if details == nil {
		writeJSONError(w, http.StatusNotFound, "TMDB details not found")
		return
	}
	writeJSON(w, http.StatusOK, details)
}

func handleTmdb(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/tmdb/movie/")
	id = strings.TrimPrefix(id, "/tmdb/movie/")
	if id == "" {
		id = r.URL.Query().Get("id")
	}
	handleTmdbWithID(w, r, id)
}

func handleTest(w http.ResponseWriter, r *http.Request) {
	vodId := "1466695562"
	vodDetails := getVodInfo(vodId, 0)
	if vodDetails == nil {
		writeJSONError(w, http.StatusBadGateway, "Upstream connection test failed")
		return
	}
	writeJSON(w, http.StatusOK, vodDetails)
}

func handleMovieWithID(w http.ResponseWriter, r *http.Request, id string) {
	if id == "" {
		id = "550"
	}

	// 1. Get TMDB Details
	tmdb := fetchTmdbDetails(id, "movie")
	if tmdb == nil {
		writeJSONError(w, http.StatusNotFound, "TMDB info not found")
		return
	}

	// 2. Search for the title
	res := apiPost("/api/search/result", map[string]string{"kw": tmdb.Title, "pn": "1"})
	vodId := extractVodId(res)

	// Fallback: If no match and title has punctuation, try simplified title
	if vodId == "" && (strings.Contains(tmdb.Title, ":") || strings.Contains(tmdb.Title, "-")) {
		simpleTitle := strings.Split(strings.Split(tmdb.Title, ":")[0], "-")[0]
		res = apiPost("/api/search/result", map[string]string{"kw": strings.TrimSpace(simpleTitle), "pn": "1"})
		vodId = extractVodId(res)
	}

	if vodId == "" {
		writeJSONError(w, http.StatusNotFound, "Video not found on server")
		return
	}

	// 3. Get Vod Info and extract signed URL
	vodDetails := getVodInfo(vodId, 0)
	var signedUrl string
	if vodDetails != nil {
		if result, ok := vodDetails["result"].(map[string]interface{}); ok {
			if collections, ok := result["vod_collection"].([]interface{}); ok && len(collections) > 0 {
				if ep, ok := collections[0].(map[string]interface{}); ok {
					if u, ok := ep["signed_url"].(string); ok {
						signedUrl = u
					}
				}
			}
		}
	}

	if signedUrl == "" {
		writeJSONError(w, http.StatusNotFound, "Stream URL not available")
		return
	}

	response := map[string]string{
		"id":   id,
		"name": tmdb.Title,
		"url":  signedUrl,
	}
	writeJSON(w, http.StatusOK, response)
}

func handleMovie(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/movie/")
	id = strings.TrimPrefix(id, "/movie/")
	if id == "" {
		id = r.URL.Query().Get("id")
	}
	handleMovieWithID(w, r, id)
}

func handleTvWithParams(w http.ResponseWriter, r *http.Request, id, season string, episode int) {
	// 1. Get TMDB Details
	tmdb := fetchTmdbDetails(id, "tv")
	if tmdb == nil {
		writeJSONError(w, http.StatusNotFound, "TMDB info not found")
		return
	}

	// 2. Search for the title + season
	searchQuery := fmt.Sprintf("%s Season %s", tmdb.Title, season)
	res := apiPost("/api/search/result", map[string]string{"kw": searchQuery, "pn": "1"})
	vodId := extractVodId(res)

	// Fallback search to just the title
	if vodId == "" {
		res = apiPost("/api/search/result", map[string]string{"kw": tmdb.Title, "pn": "1"})
		vodId = extractVodId(res)
	}

	if vodId == "" {
		writeJSONError(w, http.StatusNotFound, "Video not found on server")
		return
	}

	// 3. Get Vod Info and extract signed URL for the specific episode
	vodDetails := getVodInfo(vodId, 0)
	var signedUrl string
	if vodDetails != nil {
		if result, ok := vodDetails["result"].(map[string]interface{}); ok {
			if collections, ok := result["vod_collection"].([]interface{}); ok {
				for _, epIntf := range collections {
					if ep, ok := epIntf.(map[string]interface{}); ok {
						colNum := 0
						if colFloat, ok := ep["collection"].(float64); ok {
							colNum = int(colFloat)
						}
						if colNum == episode {
							if u, ok := ep["signed_url"].(string); ok {
								signedUrl = u
								break
							}
						}
					}
				}
			}
		}
	}

	if signedUrl == "" {
		writeJSONError(w, http.StatusNotFound, "Episode not found")
		return
	}

	response := map[string]string{
		"id":      id,
		"name":    tmdb.Title,
		"season":  season,
		"episode": strconv.Itoa(episode),
		"url":     signedUrl,
	}
	writeJSON(w, http.StatusOK, response)
}

func handleTv(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/tv/")
	path = strings.TrimPrefix(path, "/tv/")
	parts := strings.Split(path, "/")
	if len(parts) < 3 {
		writeJSONError(w, http.StatusBadRequest, "Invalid format, use /api/tv/{id}/{season}/{episode}")
		return
	}

	id := parts[0]
	season := parts[1]
	episode, _ := strconv.Atoi(parts[2])
	handleTvWithParams(w, r, id, season, episode)
}

// Handler is the universal HTTP entry point for Vercel Functions, Netlify Functions, and Go serverless runtimes.
func Handler(w http.ResponseWriter, r *http.Request) {
	addCorsHeaders(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	rawPath := r.URL.Path
	trimmedPath := strings.Trim(rawPath, "/")
	cleanPath := trimmedPath
	if strings.HasPrefix(cleanPath, "api/") {
		cleanPath = strings.TrimPrefix(cleanPath, "api/")
	}

	query := r.URL.Query()
	reqType := query.Get("type")

	// 1. Query parameter routing (?type=movie&id=550 or ?type=tv&id=1399&season=1&episode=6)
	if reqType == "movie" && query.Get("id") != "" {
		handleMovieWithID(w, r, query.Get("id"))
		return
	}
	if reqType == "tv" && query.Get("id") != "" {
		season := query.Get("season")
		if season == "" {
			season = query.Get("s")
		}
		if season == "" {
			season = "1"
		}
		episodeStr := query.Get("episode")
		if episodeStr == "" {
			episodeStr = query.Get("e")
		}
		epNum, _ := strconv.Atoi(episodeStr)
		if epNum <= 0 {
			epNum = 1
		}
		handleTvWithParams(w, r, query.Get("id"), season, epNum)
		return
	}

	// 2. Path-based routing
	switch {
	case cleanPath == "" || cleanPath == "api":
		handleRoot(w, r)
	case cleanPath == "test":
		handleTest(w, r)
	case strings.HasPrefix(cleanPath, "tmdb/movie/"):
		id := strings.TrimPrefix(cleanPath, "tmdb/movie/")
		handleTmdbWithID(w, r, id)
	case strings.HasPrefix(cleanPath, "movie/"):
		id := strings.TrimPrefix(cleanPath, "movie/")
		handleMovieWithID(w, r, id)
	case strings.HasPrefix(cleanPath, "tv/"):
		parts := strings.Split(strings.TrimPrefix(cleanPath, "tv/"), "/")
		if len(parts) >= 3 {
			id := parts[0]
			season := parts[1]
			epNum, _ := strconv.Atoi(parts[2])
			handleTvWithParams(w, r, id, season, epNum)
		} else {
			writeJSONError(w, http.StatusBadRequest, "Invalid format, use /api/tv/{id}/{season}/{episode}")
		}
	default:
		if id := query.Get("id"); id != "" {
			handleMovieWithID(w, r, id)
			return
		}
		handleRoot(w, r)
	}
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = strconv.Itoa(DefaultPort)
	}

	log.Printf("Server listening on port %s", port)
	if err := http.ListenAndServe(":"+port, http.HandlerFunc(Handler)); err != nil {
		log.Fatal(err)
	}
}
