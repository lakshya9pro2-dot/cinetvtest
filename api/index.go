package handler

import (
	"bytes"
	"compress/gzip"
	"crypto/aes"
	"crypto/cipher"
	"crypto/des"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
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
	DEFAULT_DEVICE_ID    = "511823550d113944a1db38c93aeb95d6"
	DEFAULT_TOKEN        = "gAAAAABq4bOqDhy8xJRscb7PcWvu1XclowGNCpvjAnTMWqtxdb353tgMz8lhuXu4bvQPdQenJ3KWPvgme8Zqanzxo21O9q9aGjC6t_dsNELuDDmMnnqWkgjIbxUgV4emBAHuLVwH5Y7FEZeHaQwAEQ50VyG5s6Ba9Uycr_9pDD-4k5Jb6x7Q48puopgD2S9Yfw8uGGj1tO45qEMmkp-nEyRDj-uvtmjz_vY8YWIpHwRTdDKHt-9hdgKrC9J-kSWTvYwe5cDOpAUa"
	DEFAULT_MOB_MFR      = "OnePlus"
	DEFAULT_MOB_MODEL    = "CPH2449"
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
	deviceId = os.Getenv("DEVICE_ID")
	if deviceId == "" {
		deviceId = DEFAULT_DEVICE_ID
	}
	cachedToken = os.Getenv("TOKEN")
	if cachedToken == "" {
		cachedToken = DEFAULT_TOKEN
	}
	mobMfr = DEFAULT_MOB_MFR
	mobModel = DEFAULT_MOB_MODEL
	log.Printf("[BOOT] NanoServer Go initialized. Emulated: %s %s (ID: %s)", mobMfr, mobModel, deviceId)
}

type TmdbInfo struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Year        string `json:"year"`
	Type        string `json:"type"`
	Season      *int   `json:"season,omitempty"`
	Episode     *int   `json:"episode,omitempty"`
	SeasonName  string `json:"season_name,omitempty"`
	SeasonYear  string `json:"season_year,omitempty"`
	Description string `json:"description,omitempty"`
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

	start := time.Now()
	req, err := http.NewRequest("POST", u, strings.NewReader(data.Encode()))
	if err != nil {
		log.Printf("[UPSTREAM POST ERROR] %s: %v", endpoint, err)
		return nil
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[UPSTREAM POST TIMEOUT/FAIL] %s: %v", endpoint, err)
		return nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	log.Printf("[UPSTREAM POST] %s -> Status: %d (%s)", endpoint, resp.StatusCode, time.Since(start))
	return body
}

func invalidateToken() {
	tokenLock.Lock()
	defer tokenLock.Unlock()
	log.Println("[AUTH] Token invalidated")
	cachedToken = ""
}

func fetchToken() string {
	tokenLock.Lock()
	defer tokenLock.Unlock()

	if cachedToken != "" {
		return cachedToken
	}

	log.Println("[AUTH] Requesting new session token from upstream (/api/public/init)...")
	curTime := strconv.FormatInt(time.Now().UnixMilli(), 10)
	headers := buildHeaders(curTime, "")
	buf := httpsPost("/api/public/init", map[string]string{"invited_by": "", "is_install": "1"}, headers)
	if len(buf) == 0 {
		log.Println("[AUTH WARN] Upstream /api/public/init empty. Using stable token.")
		cachedToken = DEFAULT_TOKEN
		return cachedToken
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
					log.Printf("[AUTH SUCCESS] Token acquired: %s...", cachedToken[:8])
					return cachedToken
				}
			}
		}
	}
	log.Printf("[AUTH WARN] Failed to extract token from upstream: %s. Using stable token.", jsonStr[:min(60, len(jsonStr))])
	cachedToken = DEFAULT_TOKEN
	return cachedToken
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
		log.Printf("[API ERROR] Unmarshal failed: %v", err)
		return nil
	}

	// Retry once on unauthorized token
	if code, ok := result["code"].(float64); ok && code == 401 {
		log.Println("[API] Token unauthorized (401). Retrying with fresh token...")
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
	log.Printf("[VOD] Requesting vod details for ID: %s", vodId)
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
		log.Printf("[VOD ERROR] Empty response for VOD ID: %s", vodId)
		return nil
	}

	dec := aesDecrypt(strings.TrimSpace(string(buf)))
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(dec), &result); err != nil {
		log.Printf("[VOD ERROR] Unmarshal failed: %v", err)
		return nil
	}

	if code, ok := result["code"].(float64); ok && code == 401 {
		log.Println("[VOD] Token unauthorized (401). Retrying...")
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
			log.Printf("[VOD SUCCESS] Found %d stream collection item(s)", len(collections))
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

func fetchTmdbTvDetails(id string, season string) *TmdbInfo {
	apiKey := getTmdbApiKey()
	log.Printf("[TMDB] Fetching TV show and Season %s details for ID: %s", season, id)
	realId := id
	if strings.HasPrefix(id, "tt") {
		findURL := fmt.Sprintf("https://api.themoviedb.org/3/find/%s?api_key=%s&external_source=imdb_id", id, apiKey)
		jsonStr := httpGet(findURL, nil)
		if jsonStr != "" {
			var findRes map[string]interface{}
			json.Unmarshal([]byte(jsonStr), &findRes)
			if tvResults, ok := findRes["tv_results"].([]interface{}); ok && len(tvResults) > 0 {
				if first, ok := tvResults[0].(map[string]interface{}); ok {
					if fid, ok := first["id"].(float64); ok {
						realId = strconv.Itoa(int(fid))
						log.Printf("[TMDB] Resolved IMDb %s to TMDB ID %s", id, realId)
					}
				}
			}
		}
	}

	reqURL := fmt.Sprintf("https://api.themoviedb.org/3/tv/%s?api_key=%s", realId, apiKey)
	jsonStr := httpGet(reqURL, nil)
	if jsonStr == "" {
		log.Printf("[TMDB ERROR] Could not fetch TV show for ID: %s", realId)
		return nil
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return nil
	}

	idVal, _ := data["id"].(float64)
	showTitle, _ := data["name"].(string)
	if showTitle == "" {
		showTitle, _ = data["original_name"].(string)
	}

	showDate, _ := data["first_air_date"].(string)
	showYear := showDate
	if strings.Contains(showDate, "-") {
		showYear = strings.Split(showDate, "-")[0]
	}

	seasonNum, _ := strconv.Atoi(season)
	var seasonName, seasonYear, seasonDesc string

	if seasons, ok := data["seasons"].([]interface{}); ok {
		for _, sIntf := range seasons {
			if s, ok := sIntf.(map[string]interface{}); ok {
				sNum := 0
				if snf, ok := s["season_number"].(float64); ok {
					sNum = int(snf)
				}
				if sNum == seasonNum {
					seasonName, _ = s["name"].(string)
					seasonDesc, _ = s["overview"].(string)
					sDate, _ := s["air_date"].(string)
					if strings.Contains(sDate, "-") {
						seasonYear = strings.Split(sDate, "-")[0]
					} else {
						seasonYear = sDate
					}
					break
				}
			}
		}
	}

	// Fallback to fetch season endpoint if overview or air date is empty
	if (seasonDesc == "" || seasonYear == "") && realId != "" && season != "" {
		sURL := fmt.Sprintf("https://api.themoviedb.org/3/tv/%s/season/%s?api_key=%s", realId, season, apiKey)
		sJson := httpGet(sURL, nil)
		if sJson != "" {
			var sData map[string]interface{}
			if err := json.Unmarshal([]byte(sJson), &sData); err == nil {
				if seasonDesc == "" {
					seasonDesc, _ = sData["overview"].(string)
				}
				if seasonYear == "" {
					sDate, _ := sData["air_date"].(string)
					if strings.Contains(sDate, "-") {
						seasonYear = strings.Split(sDate, "-")[0]
					} else {
						seasonYear = sDate
					}
				}
			}
		}
	}

	if seasonYear == "" {
		seasonYear = showYear
	}
	if seasonName == "" {
		seasonName = fmt.Sprintf("Season %s", season)
	}

	log.Printf("[TMDB SUCCESS] TV '%s' Season %s -> Year: %s, Desc: '%.60s...'", showTitle, season, seasonYear, seasonDesc)

	return &TmdbInfo{
		ID:          int(idVal),
		Title:       showTitle,
		Year:        showYear,
		SeasonYear:  seasonYear,
		SeasonName:  seasonName,
		Description: seasonDesc,
		Type:        "tv",
		Season:      &seasonNum,
	}
}

func fetchTmdbDetails(id string, mediaType string) *TmdbInfo {
	apiKey := getTmdbApiKey()
	log.Printf("[TMDB] Fetching details for %s ID: %s", mediaType, id)
	if strings.HasPrefix(id, "tt") {
		findURL := fmt.Sprintf("https://api.themoviedb.org/3/find/%s?api_key=%s&external_source=imdb_id", id, apiKey)
		jsonStr := httpGet(findURL, nil)
		if jsonStr != "" {
			var findRes map[string]interface{}
			json.Unmarshal([]byte(jsonStr), &findRes)

			tvResults, _ := findRes["tv_results"].([]interface{})
			movieResults, _ := findRes["movie_results"].([]interface{})

			if mediaType == "tv" && len(tvResults) > 0 {
				res := parseTmdbMap(tvResults[0].(map[string]interface{}), "tv")
				log.Printf("[TMDB SUCCESS] Resolved IMDb %s -> TV '%s' (%s)", id, res.Title, res.Year)
				return res
			} else if len(movieResults) > 0 {
				res := parseTmdbMap(movieResults[0].(map[string]interface{}), "movie")
				log.Printf("[TMDB SUCCESS] Resolved IMDb %s -> Movie '%s' (%s)", id, res.Title, res.Year)
				return res
			} else if len(tvResults) > 0 {
				res := parseTmdbMap(tvResults[0].(map[string]interface{}), "tv")
				log.Printf("[TMDB SUCCESS] Resolved IMDb %s -> TV '%s' (%s)", id, res.Title, res.Year)
				return res
			}
		}
	}

	if mediaType == "movie" || mediaType == "tv" {
		reqURL := fmt.Sprintf("https://api.themoviedb.org/3/%s/%s?api_key=%s", mediaType, id, apiKey)
		jsonStr := httpGet(reqURL, nil)
		if jsonStr != "" {
			var data map[string]interface{}
			json.Unmarshal([]byte(jsonStr), &data)
			res := parseTmdbMap(data, mediaType)
			log.Printf("[TMDB SUCCESS] Resolved %s %s -> '%s' (%s)", mediaType, id, res.Title, res.Year)
			return res
		}
	}

	log.Printf("[TMDB ERROR] Could not find TMDB details for ID: %s", id)
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
	log.Printf("[HTTP %d] %s", status, message)
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

func extractCandidates(response map[string]interface{}) []map[string]interface{} {
	var candidates []map[string]interface{}
	if response == nil {
		return candidates
	}
	if items, ok := response["result"].([]interface{}); ok {
		for _, it := range items {
			if m, ok := it.(map[string]interface{}); ok {
				candidates = append(candidates, m)
			}
		}
	} else if resultMap, ok := response["result"].(map[string]interface{}); ok {
		if items, ok := resultMap["items"].([]interface{}); ok {
			for _, it := range items {
				if m, ok := it.(map[string]interface{}); ok {
					candidates = append(candidates, m)
				}
			}
		}
	}
	return candidates
}

func normalizeCleanText(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == ' ' {
			b.WriteRune(r)
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func calcWordOverlap(s1, s2 string) float64 {
	w1 := strings.Fields(normalizeCleanText(s1))
	w2 := strings.Fields(normalizeCleanText(s2))
	if len(w1) == 0 || len(w2) == 0 {
		return 0.0
	}
	set1 := make(map[string]bool)
	for _, w := range w1 {
		if len(w) > 3 {
			set1[w] = true
		}
	}
	if len(set1) == 0 {
		return 0.0
	}
	matches := 0
	for _, w := range w2 {
		if len(w) > 3 && set1[w] {
			matches++
		}
	}
	return float64(matches) / float64(len(set1))
}

type tvCandidate struct {
	idStr      string
	item       map[string]interface{}
	score      int
	vodDetails map[string]interface{}
}

func findBestMatchingTvVod(items []map[string]interface{}, showTitle, season, seasonYear, seasonDesc string) (string, map[string]interface{}) {
	if len(items) == 0 {
		return "", nil
	}

	normShow := normalizeCleanText(showTitle)
	normSeasonStr := "season " + season
	normTarget := normalizeCleanText(fmt.Sprintf("%s Season %s", showTitle, season))

	var candidates []tvCandidate
	for _, item := range items {
		name, _ := item["vod_name"].(string)
		year, _ := item["vod_year"].(string)
		var idStr string
		if idFloat, ok := item["id"].(float64); ok {
			idStr = strconv.FormatFloat(idFloat, 'f', -1, 64)
		} else if idS, ok := item["id"].(string); ok {
			idStr = idS
		}
		if idStr == "" {
			continue
		}

		score := 0
		normName := normalizeCleanText(name)

		// 1. Name matching
		if normName == normTarget {
			score += 100
		} else if strings.HasPrefix(normName, normShow) && strings.Contains(normName, normSeasonStr) {
			score += 70
			// Penalize extra subtitles (e.g. "Dexter: Resurrection" vs "Dexter")
			cleaned := strings.TrimSpace(strings.Replace(strings.Replace(normName, normShow, "", 1), normSeasonStr, "", 1))
			if len(cleaned) > 0 {
				score -= 40
			}
		} else if strings.Contains(normName, normShow) {
			score += 20
		}

		// 2. Year matching
		if year != "" && seasonYear != "" {
			candYear, err1 := strconv.Atoi(year)
			targetYear, err2 := strconv.Atoi(seasonYear)
			if err1 == nil && err2 == nil {
				diff := int(math.Abs(float64(candYear - targetYear)))
				if diff == 0 {
					score += 80
				} else if diff == 1 {
					score += 30
				} else {
					score -= 100 // heavy penalty for year mismatch
				}
			}
		}

		candidates = append(candidates, tvCandidate{
			idStr: idStr,
			item:  item,
			score: score,
		})
	}

	if len(candidates) == 0 {
		return "", nil
	}

	// Sort candidates by score descending
	for i := 0; i < len(candidates)-1; i++ {
		for j := i + 1; j < len(candidates); j++ {
			if candidates[j].score > candidates[i].score {
				candidates[i], candidates[j] = candidates[j], candidates[i]
			}
		}
	}

	// Evaluate top candidates (up to 3) by checking vodDetails (blurb and series_info)
	checkLimit := min(3, len(candidates))
	var bestCandidate *tvCandidate

	for i := 0; i < checkLimit; i++ {
		c := &candidates[i]
		log.Printf("[TV CANDIDATE] #%d: ID %s (%s, %s) -> Base Score: %d", i+1, c.idStr, c.item["vod_name"], c.item["vod_year"], c.score)

		details := getVodInfo(c.idStr, 0)
		if details == nil {
			continue
		}
		c.vodDetails = details

		if resObj, ok := details["result"].(map[string]interface{}); ok {
			// Check series_info for exact season mapping
			if seriesInfo, ok := resObj["series_info"].([]interface{}); ok {
				for _, sMap := range seriesInfo {
					if sm, ok := sMap.(map[string]interface{}); ok {
						seriesName, _ := sm["series"].(string)
						if strings.EqualFold(strings.TrimSpace(seriesName), "Season "+season) {
							if sidFloat, ok := sm["vod_id"].(float64); ok {
								exactId := strconv.FormatFloat(sidFloat, 'f', -1, 64)
								if exactId != c.idStr {
									log.Printf("[TV SERIES_INFO] Found exact season VOD ID %s in series_info", exactId)
									exactDetails := getVodInfo(exactId, 0)
									if exactDetails != nil {
										c.idStr = exactId
										c.vodDetails = exactDetails
										c.score += 100
									}
								} else {
									c.score += 80
								}
								break
							}
						}
					}
				}
			}

			// Verify description / blurb
			blurb, _ := resObj["vod_blurb"].(string)
			if blurb != "" && seasonDesc != "" {
				overlap := calcWordOverlap(seasonDesc, blurb)
				log.Printf("[TV BLURB VERIFY] ID %s Description overlap: %.2f", c.idStr, overlap)
				if overlap >= 0.25 {
					c.score += 150
				} else if overlap >= 0.10 {
					c.score += 50
				}
			}

			// Verify detail vod_year if search item had empty year
			detailYear, _ := resObj["vod_year"].(string)
			if detailYear != "" && seasonYear != "" {
				cYear, err1 := strconv.Atoi(detailYear)
				tYear, err2 := strconv.Atoi(seasonYear)
				if err1 == nil && err2 == nil {
					diff := int(math.Abs(float64(cYear - tYear)))
					if diff == 0 {
						c.score += 30
					} else if diff > 1 {
						c.score -= 80
					}
				}
			}
		}

		if bestCandidate == nil || c.score > bestCandidate.score {
			bestCandidate = c
		}
	}

	if bestCandidate == nil || bestCandidate.score <= 0 {
		if len(candidates) > 0 {
			first := &candidates[0]
			if first.vodDetails == nil {
				first.vodDetails = getVodInfo(first.idStr, 0)
			}
			return first.idStr, first.vodDetails
		}
		return "", nil
	}

	log.Printf("[TV MATCH WINNER] Selected VOD ID: %s (Score: %d)", bestCandidate.idStr, bestCandidate.score)
	return bestCandidate.idStr, bestCandidate.vodDetails
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
        <li><code>GET /api/health</code> - Diagnostics & connectivity test</li>
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

func handleHealth(w http.ResponseWriter, r *http.Request) {
	token := fetchToken()
	tmdb := fetchTmdbDetails("550", "movie")
	res := map[string]interface{}{
		"status":          "ok",
		"timestamp":       time.Now().UTC().Format(time.RFC3339),
		"emulated_device": fmt.Sprintf("%s %s (%s)", mobMfr, mobModel, deviceId),
		"upstream": map[string]interface{}{
			"host":           MAIN_URL,
			"token_acquired": token != "",
			"token_preview":  token[:min(8, len(token))] + "...",
		},
		"tmdb": map[string]interface{}{
			"connected":    tmdb != nil,
			"sample_title": tmdb != nil && tmdb.Title != "",
		},
	}
	writeJSON(w, http.StatusOK, res)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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
	log.Printf("[MOVIE] Processing request for TMDB ID: %s", id)
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
	log.Printf("[SEARCH] Querying upstream for '%s'...", tmdb.Title)
	res := apiPost("/api/search/result", map[string]string{"kw": tmdb.Title, "pn": "1"})
	vodId := extractVodId(res)

	// Fallback: If no match and title has punctuation, try simplified title
	if vodId == "" && (strings.Contains(tmdb.Title, ":") || strings.Contains(tmdb.Title, "-")) {
		simpleTitle := strings.Split(strings.Split(tmdb.Title, ":")[0], "-")[0]
		log.Printf("[SEARCH FALLBACK] Querying simplified title: '%s'...", strings.TrimSpace(simpleTitle))
		res = apiPost("/api/search/result", map[string]string{"kw": strings.TrimSpace(simpleTitle), "pn": "1"})
		vodId = extractVodId(res)
	}

	if vodId == "" {
		log.Printf("[MOVIE NOT FOUND] Upstream search returned no results for '%s'", tmdb.Title)
		writeJSONError(w, http.StatusNotFound, "Video not found on server")
		return
	}

	// 3. Get Vod Info and extract signed URL
	log.Printf("[MOVIE FOUND] Upstream VOD ID: %s. Resolving signed URL...", vodId)
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
		log.Printf("[MOVIE ERROR] Signed stream URL empty for VOD ID: %s", vodId)
		writeJSONError(w, http.StatusNotFound, "Stream URL not available")
		return
	}

	log.Printf("[MOVIE SUCCESS] Resolved stream URL for '%s': %s", tmdb.Title, signedUrl[:min(60, len(signedUrl))]+"...")
	response := map[string]string{
		"id":   id,
		"name": tmdb.Title,
		"url":  signedUrl,
	}
	writeJSON(w, http.StatusOK, response)
}

func handleTvWithParams(w http.ResponseWriter, r *http.Request, id, season string, episode int) {
	log.Printf("[TV] Processing request for TV ID: %s, Season: %s, Episode: %d", id, season, episode)

	// 1. Get TMDB Details with Season-specific Year and Description
	tmdb := fetchTmdbTvDetails(id, season)
	if tmdb == nil {
		writeJSONError(w, http.StatusNotFound, "TMDB info not found")
		return
	}

	// 2. Search for the title + season (e.g. "The Mentalist - Season 1")
	searchQuery := fmt.Sprintf("%s - Season %s", tmdb.Title, season)
	log.Printf("[TV SEARCH] Primary query: '%s'...", searchQuery)
	res := apiPost("/api/search/result", map[string]string{"kw": searchQuery, "pn": "1"})
	candidates := extractCandidates(res)

	// Fallback 1: Retrying without hyphen (e.g. "The Mentalist Season 1")
	if len(candidates) == 0 {
		searchQuery2 := fmt.Sprintf("%s Season %s", tmdb.Title, season)
		log.Printf("[TV FALLBACK 1] Retrying search with: '%s'...", searchQuery2)
		res = apiPost("/api/search/result", map[string]string{"kw": searchQuery2, "pn": "1"})
		candidates = extractCandidates(res)
	}

	// Fallback 2: Retrying with title only
	if len(candidates) == 0 {
		log.Printf("[TV FALLBACK 2] Retrying search with title only: '%s'...", tmdb.Title)
		res = apiPost("/api/search/result", map[string]string{"kw": tmdb.Title, "pn": "1"})
		candidates = extractCandidates(res)
	}

	if len(candidates) == 0 {
		log.Printf("[TV NOT FOUND] Upstream search returned no results for '%s'", tmdb.Title)
		writeJSONError(w, http.StatusNotFound, "Video not found on server")
		return
	}

	// 3. Find the best matching VOD by validating Season Name, Year, and Description
	vodId, vodDetails := findBestMatchingTvVod(candidates, tmdb.Title, season, tmdb.SeasonYear, tmdb.Description)
	if vodId == "" || vodDetails == nil {
		log.Printf("[TV NOT FOUND] No matching VOD verified for '%s' Season %s", tmdb.Title, season)
		writeJSONError(w, http.StatusNotFound, "Video not found on server")
		return
	}

	// 4. Extract signed URL for the specific episode
	log.Printf("[TV FOUND] Upstream VOD ID: %s. Resolving episode %d...", vodId, episode)
	var signedUrl string
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

	if signedUrl == "" {
		log.Printf("[TV ERROR] Episode %d not found in collection for VOD ID: %s", episode, vodId)
		writeJSONError(w, http.StatusNotFound, "Episode not found")
		return
	}

	log.Printf("[TV SUCCESS] Resolved stream URL for '%s' S%sE%d", tmdb.Title, season, episode)
	response := map[string]string{
		"id":          id,
		"name":        tmdb.Title,
		"season":      season,
		"episode":     strconv.Itoa(episode),
		"season_year": tmdb.SeasonYear,
		"description": tmdb.Description,
		"url":         signedUrl,
	}
	writeJSON(w, http.StatusOK, response)
}

// Handler is the universal HTTP entry point for Vercel Functions, Netlify Functions, and Go serverless runtimes.
func Handler(w http.ResponseWriter, r *http.Request) {
	log.Printf("[REQ] %s %s", r.Method, r.URL.RequestURI())
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

	// Fast return on favicon.ico
	if cleanPath == "favicon.ico" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

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
	case cleanPath == "health" || cleanPath == "debug":
		handleHealth(w, r)
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
