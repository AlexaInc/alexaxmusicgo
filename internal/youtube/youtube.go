package youtube

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"alexamusic/internal/queue"
)

const (
	ytBase = "https://www.youtube.com/watch?v="
	dlDir  = "downloads"
)

// API endpoints extracted directly from hansaka1/ytdl index.js
const (
	sanityKeyURL  = "https://cnv.cx/v2/sanity/key"
	converterURL  = "https://cnv.cx/v2/converter"
	searchURL     = "https://search.nnmn.store/"
	mattWAPIBase  = "https://ytapi.apps.mattw.io/v3/videos?key=foo1&part=snippet%2CcontentDetails&id="
)

var (
	apiHeaders = map[string]string{
		"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Origin":     "https://frame.y2meta-uk.com",
		"Referer":    "https://frame.y2meta-uk.com/",
	}

	ytRegex = regexp.MustCompile(
		`(?i)(?:youtube\.com/(?:[^/]+/.+/|(?:v|e(?:mbed)?)/|.*[?&]v=)|youtu\.be/|youtube\.com/shorts/)([^"&?/\s]{11})`,
	)
)

// YouTube provides YouTube search and download via the same method as hansaka1/ytdl.
type YouTube struct {
	client *http.Client
}

func New() *YouTube {
	return &YouTube{
		client: &http.Client{Timeout: 300 * time.Second},
	}
}

// Valid returns true if the string looks like a YouTube URL or video ID.
func (y *YouTube) Valid(rawURL string) bool {
	return ytRegex.MatchString(rawURL)
}

// ExtractID extracts the 11-char YouTube video ID from a URL.
func (y *YouTube) ExtractID(rawURL string) string {
	m := ytRegex.FindStringSubmatch(rawURL)
	if len(m) >= 2 {
		return m[1]
	}
	return ""
}

// ExtractURL strips tracking params from a YouTube URL.
func (y *YouTube) ExtractURL(text string) string {
	loc := ytRegex.FindString(text)
	if loc == "" {
		return ""
	}
	if idx := strings.Index(loc, "&si"); idx != -1 {
		loc = loc[:idx]
	}
	if idx := strings.Index(loc, "?si"); idx != -1 {
		loc = loc[:idx]
	}
	return loc
}

// ─── SEARCH ──────────────────────────────────────────────────────────────────

// Search finds a YouTube video and returns track metadata.
// For URLs: uses MattW API (same as index.js METHOD 1).
// For text: uses search.nnmn.store (same as index.js METHOD 2).
func (y *YouTube) Search(query string, msgID int, video bool) (*queue.Track, error) {
	query = strings.TrimSpace(query)

	// Remove tracking params
	if strings.Contains(query, "&si") {
		if idx := strings.Index(query, "&si"); idx != -1 {
			query = query[:idx]
		}
	}

	// METHOD 1: URL detected → use MattW API
	if videoID := y.ExtractID(query); videoID != "" {
		log.Printf("[yt] URL detected, using MattW API for ID: %s", videoID)
		return y.searchByID(videoID, msgID, video)
	}

	// METHOD 2: Text search → use search.nnmn.store
	log.Printf("[yt] Text search: %s", query)
	return y.searchByText(query, msgID, video)
}

// searchByID fetches metadata via MattW API (mirrors index.js METHOD 1).
func (y *YouTube) searchByID(videoID string, msgID int, video bool) (*queue.Track, error) {
	req, _ := http.NewRequest("GET", mattWAPIBase+videoID, nil)
	req.Header.Set("Referer", "https://mattw.io/")
	req.Header.Set("User-Agent", apiHeaders["User-Agent"])

	resp, err := y.client.Do(req)
	if err != nil {
		return y.fallbackTrack(videoID, msgID, video), nil
	}
	defer resp.Body.Close()

	var data struct {
		Items []struct {
			Snippet struct {
				Title        string `json:"title"`
				ChannelTitle string `json:"channelTitle"`
				PublishedAt  string `json:"publishedAt"`
				Thumbnails   struct {
					Medium struct{ URL string `json:"url"` } `json:"medium"`
				} `json:"thumbnails"`
			} `json:"snippet"`
			ContentDetails struct {
				Duration string `json:"duration"`
			} `json:"contentDetails"`
		} `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil || len(data.Items) == 0 {
		return y.fallbackTrack(videoID, msgID, video), nil
	}

	item := data.Items[0]
	durStr := parseISO8601(item.ContentDetails.Duration)
	title := item.Snippet.Title
	if len(title) > 25 {
		title = title[:25]
	}
	chanName := item.Snippet.ChannelTitle
	if len(chanName) > 25 {
		chanName = chanName[:25]
	}

	return &queue.Track{
		ID:          videoID,
		ChannelName: chanName,
		Duration:    durStr,
		DurationSec: toSeconds(durStr),
		MessageID:   msgID,
		Title:       title,
		Thumbnail:   item.Snippet.Thumbnails.Medium.URL,
		URL:         ytBase + videoID,
		Video:       video,
	}, nil
}

// searchByText uses search.nnmn.store form POST (mirrors index.js METHOD 2).
func (y *YouTube) searchByText(query string, msgID int, video bool) (*queue.Track, error) {
	// Build multipart form-data
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	_ = w.WriteField("search_query", query)
	w.Close()

	req, _ := http.NewRequest("POST", searchURL, &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Origin", "https://v6.www-y2mate.com")
	req.Header.Set("Referer", "https://v6.www-y2mate.com/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/140.0.0.0 Safari/537.36")

	resp, err := y.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("text search failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Items []struct {
			ID           string `json:"id"`
			Title        string `json:"title"`
			Duration     string `json:"duration"`
			ChannelTitle string `json:"channelTitle"`
		} `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil || len(result.Items) == 0 {
		return nil, fmt.Errorf("no search results")
	}

	item := result.Items[0]
	title := item.Title
	if len(title) > 25 {
		title = title[:25]
	}
	chanName := item.ChannelTitle
	if chanName == "" {
		chanName = "Unknown"
	}
	if len(chanName) > 25 {
		chanName = chanName[:25]
	}
	thumb := fmt.Sprintf("https://i.ytimg.com/vi/%s/mqdefault.jpg", item.ID)

	return &queue.Track{
		ID:          item.ID,
		ChannelName: chanName,
		Duration:    item.Duration,
		DurationSec: toSeconds(item.Duration),
		MessageID:   msgID,
		Title:       title,
		Thumbnail:   thumb,
		URL:         ytBase + item.ID,
		Video:       video,
	}, nil
}

// fallbackTrack returns a minimal track when metadata APIs fail.
func (y *YouTube) fallbackTrack(videoID string, msgID int, video bool) *queue.Track {
	return &queue.Track{
		ID:        videoID,
		Title:     "YouTube Video",
		Duration:  "0:00",
		Thumbnail: fmt.Sprintf("https://i.ytimg.com/vi/%s/mqdefault.jpg", videoID),
		URL:       ytBase + videoID,
		MessageID: msgID,
		Video:     video,
	}
}

// ─── DOWNLOAD ─────────────────────────────────────────────────────────────────
// Mirrors index.js /download route exactly:
// 1. Get sanity key from cnv.cx/v2/sanity/key
// 2. POST to cnv.cx/v2/converter with link, format, etc.
// 3. Stream response from the returned URL

func (y *YouTube) getSanityKey() (string, error) {
	req, _ := http.NewRequest("GET", sanityKeyURL, nil)
	for k, v := range apiHeaders {
		req.Header.Set(k, v)
	}
	req.Header.Set("Timeout", "5000")

	resp, err := (&http.Client{Timeout: 8 * time.Second}).Do(req)
	if err != nil {
		return "", fmt.Errorf("sanity key request: %w", err)
	}
	defer resp.Body.Close()

	var data struct {
		Key string `json:"key"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil || data.Key == "" {
		return "", fmt.Errorf("sanity key empty")
	}
	return data.Key, nil
}

// Download fetches an audio/video file using the cnv.cx converter method.
func (y *YouTube) Download(videoID string, video bool) (string, error) {
	if err := os.MkdirAll(dlDir, 0755); err != nil {
		return "", err
	}
	ext := "mp3"
	if video {
		ext = "mp4"
	}
	filename := filepath.Join(dlDir, videoID+"."+ext)

	// Return cached file
	if info, err := os.Stat(filename); err == nil && info.Size() > 0 {
		return filename, nil
	}

	// Step 1: Get sanity key
	apiKey, err := y.getSanityKey()
	if err != nil {
		return "", fmt.Errorf("no sanity key: %w", err)
	}

	// Step 2: POST to converter
	format := "mp3"
	vCodec := ""
	videoQuality := ""
	if video {
		format = "mp4"
		vCodec = "h264"
		videoQuality = "720"
	}

	payload := url.Values{}
	payload.Set("link", ytBase+videoID)
	payload.Set("format", format)
	payload.Set("audioBitrate", "128")
	payload.Set("filenameStyle", "pretty")
	if video {
		payload.Set("vCodec", vCodec)
		payload.Set("videoQuality", videoQuality)
	}

	req, _ := http.NewRequest("POST", converterURL, strings.NewReader(payload.Encode()))
	for k, v := range apiHeaders {
		req.Header.Set(k, v)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("key", apiKey)

	convClient := &http.Client{Timeout: 60 * time.Second}
	convResp, err := convClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("converter request: %w", err)
	}
	defer convResp.Body.Close()

	var convData struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(convResp.Body).Decode(&convData); err != nil || convData.URL == "" {
		return "", fmt.Errorf("converter returned no URL")
	}

	// Step 3: Stream download from returned URL
	dlReq, _ := http.NewRequest("GET", convData.URL, nil)
	for k, v := range apiHeaders {
		dlReq.Header.Set(k, v)
	}
	dlReq.Header.Set("Accept", "*/*")

	dlResp, err := y.client.Do(dlReq)
	if err != nil {
		return "", fmt.Errorf("stream download failed: %w", err)
	}
	defer dlResp.Body.Close()

	if dlResp.StatusCode != 200 {
		return "", fmt.Errorf("download server returned status: %d", dlResp.StatusCode)
	}
	contentType := dlResp.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/html") {
		body, _ := io.ReadAll(dlResp.Body)
		log.Printf("[yt] ERROR: Received HTML instead of audio from %s: %s", convData.URL, string(body))
		return "", fmt.Errorf("download server returned an error page (likely captcha or blocked)")
	}

	f, err := os.Create(filename)
	if err != nil {
		return "", err
	}
	defer f.Close()

	buf := make([]byte, 1024*1024) // 1 MB chunks
	if _, err := io.CopyBuffer(f, dlResp.Body, buf); err != nil {
		os.Remove(filename)
		return "", fmt.Errorf("write failed: %w", err)
	}

	if info, err := os.Stat(filename); err != nil || info.Size() == 0 {
		os.Remove(filename)
		return "", fmt.Errorf("downloaded file is empty")
	}

	pcmFilename := filename + ".pcm.raw"
	// Return cached pre-transcoded file
	if info, err := os.Stat(pcmFilename); err == nil && info.Size() > 0 {
		return pcmFilename, nil
	}

	// Pre-transcode to PCM Stereo 48kHz (Zero-Lag Streaming)
	log.Printf("[yt] Pre-transcoding %s to PCM...", videoID)
	cmd := exec.Command("ffmpeg", "-y", "-i", filename, "-f", "s16le", "-ac", "2", "-ar", "48000", pcmFilename)
	if err := cmd.Run(); err != nil {
		log.Printf("[yt] WARNING: PCM pre-transcoding failed: %v", err)
		return filename, nil // fallback to original
	}

	log.Printf("[yt] Downloaded & Pre-transcoded %s (%.2f MB PCM)", videoID, float64(fileSize(pcmFilename))/(1024*1024))

	return pcmFilename, nil
}

// ─── PLAYLIST ─────────────────────────────────────────────────────────────────

// Playlist fetches basic metadata for playlist items by iterating video IDs.
// Since index.js doesn't have a playlist endpoint, we use the MattW API per video.
func (y *YouTube) Playlist(playlistURL string, limit int, user string, video bool) ([]*queue.Track, error) {
	// Extract playlist ID
	pID := extractPlaylistID(playlistURL)
	if pID == "" {
		return nil, fmt.Errorf("invalid playlist URL")
	}

	// Use YouTube RSS feed (no auth required) to get video IDs
	feedURL := fmt.Sprintf("https://www.youtube.com/feeds/videos.xml?playlist_id=%s", pID)
	resp, err := y.client.Get(feedURL)
	if err != nil {
		return nil, fmt.Errorf("playlist feed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	content := string(body)

	// Extract video IDs from RSS
	idRe := regexp.MustCompile(`<yt:videoId>([A-Za-z0-9_-]{11})</yt:videoId>`)
	matches := idRe.FindAllStringSubmatch(content, limit)

	var tracks []*queue.Track
	for _, m := range matches {
		if len(tracks) >= limit {
			break
		}
		videoID := m[1]
		track, err := y.searchByID(videoID, 0, video)
		if err != nil {
			continue
		}
		track.User = user
		tracks = append(tracks, track)
	}
	return tracks, nil
}

func extractPlaylistID(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return u.Query().Get("list")
}

// ─── HELPERS ──────────────────────────────────────────────────────────────────

// parseISO8601 converts "PT4M19S" → "4:19" (same as parseDuration in index.js).
func parseISO8601(iso string) string {
	if iso == "" {
		return "0:00"
	}
	re := regexp.MustCompile(`PT(?:(\d+)H)?(?:(\d+)M)?(?:(\d+)S)?`)
	m := re.FindStringSubmatch(iso)
	if m == nil {
		return "0:00"
	}
	h, _ := strconv.Atoi(m[1])
	min, _ := strconv.Atoi(m[2])
	sec, _ := strconv.Atoi(m[3])
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, min, sec)
	}
	return fmt.Sprintf("%d:%02d", min, sec)
}

// toSeconds converts "M:SS" or "H:MM:SS" to seconds.
func toSeconds(s string) int {
	parts := strings.Split(strings.TrimSpace(s), ":")
	total := 0
	for i, p := range parts {
		n, _ := strconv.Atoi(p)
		exp := len(parts) - 1 - i
		mul := 1
		for j := 0; j < exp; j++ {
			mul *= 60
		}
		total += n * mul
	}
	return total
}

// FormatETA formats seconds to human-readable.
func FormatETA(s int) string {
	if s < 60 {
		return fmt.Sprintf("%ds", s)
	} else if s < 3600 {
		return fmt.Sprintf("%d:%02d min", s/60, s%60)
	}
	return fmt.Sprintf("%d:%02d:%02d h", s/3600, (s%3600)/60, s%60)
}

func fileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}
