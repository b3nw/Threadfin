package src

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// theTVDB API constants
const (
	TVDBAPIBaseURL = "https://api4.thetvdb.com/v4"
	TVDBTimeout    = 30 * time.Second
	TVDBUserAgent  = "Threadfin/2.5.0"
)

// TVDB API structures
type TVDBLoginRequest struct {
	APIKey string `json:"apikey"`
}

type TVDBLoginResponse struct {
	Status string `json:"status"`
	Data   struct {
		Token string `json:"token"`
	} `json:"data"`
}

type TVDBSearchResponse struct {
	Status string `json:"status"`
	Data   []struct {
		TVDBId       string `json:"tvdb_id"`
		Name         string `json:"name"`
		FirstAired   string `json:"first_air_date"`
		Overview     string `json:"overview"`
		Type         string `json:"type"`
		Year         string `json:"year"`
		ImageURL     string `json:"image_url"` // Direct poster URL from search results
		Translations struct {
			NameTranslations []struct {
				Name     string `json:"name"`
				Language string `json:"language"`
			} `json:"nameTranslations"`
		} `json:"translations"`
	} `json:"data"`
}

type TVDBArtworkResponse struct {
	Status string `json:"status"`
	Data   struct {
		Artworks []struct {
			Id       int    `json:"id"`
			Image    string `json:"image"`
			Type     int    `json:"type"`
			Language string `json:"language"`
			Score    int    `json:"score"`
		} `json:"artworks"`
	} `json:"data"`
}

// TVDB Client structure
type TVDBClient struct {
	apiKey      string
	token       string
	tokenExpiry time.Time
	httpClient  *http.Client
	mutex       sync.RWMutex
	rateLimiter chan struct{}
}

// TVDB Cache structures - Memory-only lookup cache
type TVDBLookupEntry struct {
	PosterURL string        `json:"poster_url"`
	CacheURL  string        `json:"cache_url"`
	Timestamp time.Time     `json:"timestamp"`
	TTL       time.Duration `json:"ttl"`
}

type TVDBSearchResult struct {
	ID       string
	Name     string
	Type     string
	Year     string
	Overview string
	ImageURL string // Direct poster URL from search results
}

type TVDBPoster struct {
	URL      string
	Type     string
	Language string
	Score    int
}

// Global TVDB client instance and memory-only lookup cache
var (
	tvdbClient *TVDBClient
	// Memory-only lookup cache: title -> lookup entry with TTL
	tvdbLookupCache      = make(map[string]*TVDBLookupEntry)
	tvdbLookupCacheMutex sync.RWMutex
	// Failed lookup cache: title -> timestamp of failed attempt
	failedLookupCache      = make(map[string]time.Time)
	failedLookupCacheMutex sync.RWMutex
	failedLookupWindow     = 24 * time.Hour // Don't retry failed lookups for 24 hours
	// Rate limiting and deduplication
	lastRequestTime     time.Time
	lastRequestMutex    sync.Mutex
	minRequestInterval  = 2 * time.Second            // Minimum 2 seconds between requests
	recentSearches      = make(map[string]time.Time) // Track recent searches
	recentSearchesMutex sync.RWMutex
	recentSearchWindow  = 30 * time.Minute // Don't re-search same title within 30 minutes

	// Track in-progress searches to prevent race conditions
	inProgressSearches      = make(map[string]bool)
	inProgressSearchesMutex sync.RWMutex

	// Global goroutine limiter - max 3 concurrent poster lookups
	globalGoroutineLimiter chan struct{}

	// Simple flag to track if new posters were found since last EPG generation
	hasNewPosters     bool
	hasNewPosterMutex sync.RWMutex
)

// InitializeTVDB initializes the theTVDB client with the provided API key
func InitializeTVDB(apiKey string) error {
	if len(strings.TrimSpace(apiKey)) == 0 {
		return errors.New("theTVDB API key is required")
	}

	tvdbClient = &TVDBClient{
		apiKey: strings.TrimSpace(apiKey),
		httpClient: &http.Client{
			Timeout: TVDBTimeout,
		},
		rateLimiter: make(chan struct{}, 3), // Max 3 concurrent requests
	}

	// Fill rate limiter channel
	for i := 0; i < 3; i++ {
		tvdbClient.rateLimiter <- struct{}{}
	}

	// Initialize global goroutine limiter
	globalGoroutineLimiter = make(chan struct{}, 3)
	for i := 0; i < 3; i++ {
		globalGoroutineLimiter <- struct{}{}
	}

	return nil
}

// authenticateTVDB gets a JWT token from theTVDB API
func (client *TVDBClient) authenticateTVDB() error {
	if client == nil {
		return errors.New("TVDB client not initialized")
	}

	client.mutex.Lock()
	defer client.mutex.Unlock()

	// Check if token is still valid (with 5 minute buffer)
	if time.Now().Before(client.tokenExpiry.Add(-5 * time.Minute)) {
		return nil
	}

	loginRequest := TVDBLoginRequest{
		APIKey: client.apiKey,
	}

	jsonData, err := json.Marshal(loginRequest)
	if err != nil {
		showInfo(fmt.Sprintf("theTVDB: Authentication failed - could not prepare request: %v", err))
		return fmt.Errorf("failed to marshal login request: %v", err)
	}

	req, err := http.NewRequest("POST", TVDBAPIBaseURL+"/login", bytes.NewBuffer(jsonData))
	if err != nil {
		showInfo(fmt.Sprintf("theTVDB: Authentication failed - could not create request: %v", err))
		return fmt.Errorf("failed to create login request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", TVDBUserAgent)

	resp, err := client.httpClient.Do(req)
	if err != nil {
		showInfo(fmt.Sprintf("theTVDB: Authentication failed - network error: %v", err))
		return fmt.Errorf("theTVDB login request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		showInfo(fmt.Sprintf("theTVDB: Authentication failed - API returned status %d: %s", resp.StatusCode, string(body)))
		return fmt.Errorf("theTVDB login failed with status %d: %s", resp.StatusCode, string(body))
	}

	var loginResponse TVDBLoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResponse); err != nil {
		showInfo(fmt.Sprintf("theTVDB: Authentication failed - invalid response format: %v", err))
		return fmt.Errorf("failed to decode login response: %v", err)
	}

	if loginResponse.Status != "success" || loginResponse.Data.Token == "" {
		showInfo("theTVDB: Authentication failed - API returned error or empty token")
		return errors.New("theTVDB login failed: invalid response")
	}

	client.token = loginResponse.Data.Token
	client.tokenExpiry = time.Now().Add(24 * time.Hour) // Tokens are valid for 24 hours

	showInfo("theTVDB: Authentication successful")
	return nil
}

// makeAuthenticatedRequest makes an authenticated request to theTVDB API
func (client *TVDBClient) makeAuthenticatedRequest(method, endpoint string, body io.Reader) (*http.Response, error) {
	// Rate limiting
	<-client.rateLimiter
	defer func() {
		// Return rate limit token after a delay
		go func() {
			time.Sleep(250 * time.Millisecond) // 4 requests per second max
			client.rateLimiter <- struct{}{}
		}()
	}()

	// Ensure we have a valid token
	if err := client.authenticateTVDB(); err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, TVDBAPIBaseURL+endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	client.mutex.RLock()
	token := client.token
	client.mutex.RUnlock()

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", TVDBUserAgent)

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("theTVDB API request failed: %v", err)
	}

	// Handle authentication errors
	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()
		// Clear token and retry once
		client.mutex.Lock()
		client.token = ""
		client.tokenExpiry = time.Time{}
		client.mutex.Unlock()

		if err := client.authenticateTVDB(); err != nil {
			return nil, err
		}

		// Retry the request
		req.Header.Set("Authorization", "Bearer "+client.token)
		return client.httpClient.Do(req)
	}

	return resp, nil
}

// searchTVDBSeries searches for TV series on theTVDB
func (client *TVDBClient) searchTVDBSeries(title string) ([]TVDBSearchResult, error) {
	cleanTitle := parseTitle(title)
	if cleanTitle == "" {
		return nil, errors.New("empty title provided")
	}

	// Debug: Show original vs cleaned title
	if System.Flag.Debug >= 2 {
		fmt.Printf("DEBUG: theTVDB search - Original title: '%s' -> Cleaned title: '%s'\n", title, cleanTitle)
	}

	// Enforce rate limiting before making API call
	enforceRateLimit()

	endpoint := fmt.Sprintf("/search?query=%s&type=series&dne=eng&limit=5", url.QueryEscape(cleanTitle))
	if System.Dev || System.Flag.Debug >= 2 {
		fmt.Printf("DEBUG: theTVDB search - URL: %s%s\n", TVDBAPIBaseURL, endpoint)
	}
	resp, err := client.makeAuthenticatedRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("theTVDB series search failed with status %d: %s", resp.StatusCode, string(body))
	}

	var searchResponse TVDBSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResponse); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %v", err)
	}

	var results []TVDBSearchResult
	for _, item := range searchResponse.Data {
		results = append(results, TVDBSearchResult{
			ID:       item.TVDBId,
			Name:     item.Name,
			Type:     item.Type,
			Year:     item.Year,
			Overview: item.Overview,
			ImageURL: item.ImageURL,
		})
	}

	return results, nil
}

// searchTVDBMovies searches for movies on theTVDB
func (client *TVDBClient) searchTVDBMovies(title string) ([]TVDBSearchResult, error) {
	cleanTitle := parseTitle(title)
	if cleanTitle == "" {
		return nil, errors.New("empty title provided")
	}

	// Debug: Show original vs cleaned title
	if System.Flag.Debug >= 2 {
		fmt.Printf("DEBUG: theTVDB movie search - Original title: '%s' -> Cleaned title: '%s'\n", title, cleanTitle)
	}

	// Enforce rate limiting before making API call
	enforceRateLimit()

	endpoint := fmt.Sprintf("/search?query=%s&type=movie&dne=eng&limit=5", url.QueryEscape(cleanTitle))
	if System.Dev || System.Flag.Debug >= 2 {
		fmt.Printf("DEBUG: theTVDB movie search - URL: %s%s\n", TVDBAPIBaseURL, endpoint)
	}

	resp, err := client.makeAuthenticatedRequest("GET", endpoint, nil)
	if err != nil {
		if System.Dev {
			showInfo(fmt.Sprintf("theTVDB: Movie search API request failed: %v", err))
		}
		return nil, err
	}
	defer resp.Body.Close()

	if System.Dev {
		showInfo(fmt.Sprintf("theTVDB: Movie search API response status: %d", resp.StatusCode))
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		if System.Dev {
			showInfo(fmt.Sprintf("theTVDB: Movie search failed - Status: %d, Body: %s", resp.StatusCode, string(body)))
		}
		return nil, fmt.Errorf("theTVDB movie search failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		if System.Dev {
			showInfo(fmt.Sprintf("theTVDB: Failed to read movie search response body: %v", err))
		}
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	if System.Dev {
		showInfo(fmt.Sprintf("theTVDB: Movie search raw response (first 500 chars): %s", string(body[:min(len(body), 500)])))
	}

	var searchResponse TVDBSearchResponse
	if err := json.Unmarshal(body, &searchResponse); err != nil {
		if System.Dev {
			showInfo(fmt.Sprintf("theTVDB: Failed to decode movie search response: %v", err))
		}
		return nil, fmt.Errorf("failed to decode search response: %v", err)
	}

	if System.Dev {
		showInfo(fmt.Sprintf("theTVDB: Movie search decoded %d results", len(searchResponse.Data)))
	}

	var results []TVDBSearchResult
	for _, item := range searchResponse.Data {
		results = append(results, TVDBSearchResult{
			ID:       item.TVDBId,
			Name:     item.Name,
			Type:     item.Type,
			Year:     item.Year,
			Overview: item.Overview,
			ImageURL: item.ImageURL,
		})
	}

	return results, nil
}

// searchTVDBBoth searches both series and movies on theTVDB
func (client *TVDBClient) searchTVDBBoth(title string) ([]TVDBSearchResult, error) {
	var allResults []TVDBSearchResult

	// Search series first
	seriesResults, err1 := client.searchTVDBSeries(title)
	if err1 == nil {
		allResults = append(allResults, seriesResults...)
	}

	// Search movies
	movieResults, err2 := client.searchTVDBMovies(title)
	if err2 == nil {
		allResults = append(allResults, movieResults...)
	}

	// Return error only if both searches failed
	if err1 != nil && err2 != nil {
		return nil, fmt.Errorf("both series and movie searches failed: series=%v, movies=%v", err1, err2)
	}

	return allResults, nil
}

// getTVDBPosters gets poster images for a given series/movie ID
func (client *TVDBClient) getTVDBPosters(contentID, contentType string) ([]TVDBPoster, error) {
	if contentID == "" {
		return nil, errors.New("content ID is required")
	}

	var endpoint string
	switch contentType {
	case "movie":
		endpoint = fmt.Sprintf("/movies/%s/artworks", contentID)
	case "series":
		endpoint = fmt.Sprintf("/series/%s/artworks", contentID)
	default:
		// Fallback to generic endpoint
		endpoint = fmt.Sprintf("/%s/%s/artworks", contentType, contentID)
	}

	if System.Dev {
		showInfo(fmt.Sprintf("theTVDB: Artwork API call - URL: %s%s", TVDBAPIBaseURL, endpoint))
	}

	resp, err := client.makeAuthenticatedRequest("GET", endpoint, nil)
	if err != nil {
		if System.Dev {
			showInfo(fmt.Sprintf("theTVDB: Artwork request failed for %s ID %s: %v", contentType, contentID, err))
		}
		return nil, err
	}
	defer resp.Body.Close()

	if System.Dev {
		showInfo(fmt.Sprintf("theTVDB: Artwork API response status: %d for %s ID %s", resp.StatusCode, contentType, contentID))
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		if System.Dev {
			showInfo(fmt.Sprintf("theTVDB: Artwork request failed - Status: %d, Body: %s", resp.StatusCode, string(body)))
		}
		return nil, fmt.Errorf("theTVDB artwork request failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		if System.Dev {
			showInfo(fmt.Sprintf("theTVDB: Failed to read artwork response body: %v", err))
		}
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	if System.Dev {
		showInfo(fmt.Sprintf("theTVDB: Artwork raw response (first 500 chars): %s", string(body[:min(len(body), 500)])))
	}

	var artworkResponse TVDBArtworkResponse
	if err := json.Unmarshal(body, &artworkResponse); err != nil {
		if System.Dev {
			showInfo(fmt.Sprintf("theTVDB: Failed to decode artwork response: %v", err))
		}
		return nil, fmt.Errorf("failed to decode artwork response: %v", err)
	}

	if System.Dev {
		showInfo(fmt.Sprintf("theTVDB: Artwork response decoded %d artworks", len(artworkResponse.Data.Artworks)))
	}

	var posters []TVDBPoster
	for _, artwork := range artworkResponse.Data.Artworks {
		// Type 1 = poster, Type 2 = banner, Type 3 = fanart
		artworkType := "unknown"
		switch artwork.Type {
		case 1:
			artworkType = "poster"
		case 2:
			artworkType = "banner"
		case 3:
			artworkType = "fanart"
		}

		posters = append(posters, TVDBPoster{
			URL:      artwork.Image,
			Type:     artworkType,
			Language: artwork.Language,
			Score:    artwork.Score,
		})
	}

	if System.Dev {
		showInfo(fmt.Sprintf("theTVDB: Found %d posters for %s ID %s", len(posters), contentType, contentID))

		// Debug: show poster types found
		if len(posters) > 0 {
			posterTypes := make(map[string]int)
			for _, poster := range posters {
				posterTypes[poster.Type]++
			}
			showInfo(fmt.Sprintf("theTVDB: Poster types available: %v", posterTypes))
		}
	}

	return posters, nil
}

// selectBestPoster selects the best poster based on user preferences
func selectBestPoster(posters []TVDBPoster) *TVDBPoster {
	if len(posters) == 0 {
		return nil
	}

	// Define poster type priorities (higher number = higher priority)
	typePriorities := map[string]int{
		"poster":     100, // Official posters are highest priority
		"season":     90,  // Season posters are good
		"series":     80,  // Series posters are good
		"background": 50,  // Backgrounds are okay
		"banner":     30,  // Banners are lower priority
		"clearlogo":  25,  // Clear logos are lower
		"clearart":   20,  // Clear art is lower
		"fanart":     10,  // Fan art is lowest priority
		"graphical":  5,   // Graphical banners are very low
	}

	// Also check URL patterns to catch cases where type might not be set correctly
	urlTypePriorities := map[string]int{
		"/posters/":     100, // URLs containing /posters/ are highest priority
		"/series/":      80,  // Series artwork
		"/seasons/":     90,  // Season artwork
		"/backgrounds/": 50,  // Background artwork
		"/banners/":     30,  // Banner artwork
		"/fanart/":      10,  // Fan art
		"/graphical/":   5,   // Graphical banners
	}

	// Find the best poster by combined score
	var bestPoster *TVDBPoster
	bestScore := -1

	for i := range posters {
		poster := &posters[i]
		score := poster.Score

		// Add type priority bonus
		if typePriority, exists := typePriorities[strings.ToLower(poster.Type)]; exists {
			score += typePriority
		}

		// Add URL pattern bonus
		for urlPattern, urlPriority := range urlTypePriorities {
			if strings.Contains(poster.URL, urlPattern) {
				score += urlPriority
				break // Only apply one URL pattern bonus
			}
		}

		// Boost score for English language
		if poster.Language == "eng" {
			score += 20
		}

		// Boost score for higher resolution (if width/height are available)
		// This is a heuristic - posters with higher scores from theTVDB are usually better
		if poster.Score > 0 {
			score += poster.Score / 10 // Add 10% of the original theTVDB score
		}

		if bestPoster == nil || score > bestScore {
			bestPoster = poster
			bestScore = score
		}
	}

	return bestPoster
}

// parseTitle cleans and extracts the main title from a program name
func parseTitle(title string) string {
	if title == "" {
		return ""
	}

	// Convert to lowercase for processing
	cleanTitle := strings.TrimSpace(title)

	// Remove common patterns
	patterns := []string{
		`\s*\([0-9]{4}\)`,       // Year in parentheses: (2023)
		`\s*\[[0-9]{4}\]`,       // Year in brackets: [2023]
		`\s*S[0-9]+E[0-9]+.*`,   // Season/Episode: S01E01...
		`\s*-\s*.*`,             // Everything after dash
		`\s*:\s*.*`,             // Everything after colon (episode titles)
		`\s*\|\s*.*`,            // Everything after pipe
		`\s*\.\s*[0-9]+\.\s*.*`, // Date patterns: .01.
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		cleanTitle = re.ReplaceAllString(cleanTitle, "")
	}

	// Clean up extra whitespace
	cleanTitle = strings.TrimSpace(cleanTitle)
	re := regexp.MustCompile(`\s+`)
	cleanTitle = re.ReplaceAllString(cleanTitle, " ")

	return cleanTitle
}

// GetCachedPosterURL checks for a cached poster URL for the given title
func GetCachedPosterURL(title string) string {
	tvdbLookupCacheMutex.RLock()
	defer tvdbLookupCacheMutex.RUnlock()

	if System.Dev {
		showInfo(fmt.Sprintf("theTVDB DEBUG:           GetCachedPosterURL() called with title='%s', cleanTitle='%s'", title, title))
	}

	// Use the original XMLTV title as the cache key (no cleaning/parsing)
	if entry, exists := tvdbLookupCache[title]; exists {
		// TTL check temporarily disabled for testing
		/*if time.Since(entry.Timestamp) < entry.TTL {*/
		if System.Dev {
			showInfo(fmt.Sprintf("theTVDB DEBUG:           Cache hit found for '%s': %s", title, entry.CacheURL))
		}
		return entry.CacheURL
		/*}
		// Expired, remove from cache
		delete(tvdbLookupCache, title)
		if System.Dev {
			showInfo(fmt.Sprintf("theTVDB DEBUG:           Cache entry expired for '%s'", title))
		}*/
	}

	if System.Dev {
		showInfo(fmt.Sprintf("theTVDB DEBUG:           No cache entry found for '%s'", title))
	}
	return ""
}

func setCachedPosterURL(title, posterURL string) {
	// Get image cache instance
	if Data.Cache.Images == nil {
		showInfo("theTVDB: Image cache not initialized, skipping cache")
		return
	}

	// Immediately download and cache the image
	imgc := Data.Cache.Images
	cacheURL := imgc.Image.DownloadImmediately(posterURL, Settings.HttpThreadfinDomain, Settings.Port, Settings.ForceHttps, Settings.HttpsPort, Settings.HttpsThreadfinDomain)

	// Store in memory cache
	tvdbLookupCacheMutex.Lock()
	defer tvdbLookupCacheMutex.Unlock()

	if System.Dev {
		showInfo(fmt.Sprintf("theTVDB DEBUG:           setCachedPosterURL() called with title='%s', posterURL='%s'", title, posterURL))
		showInfo(fmt.Sprintf("theTVDB DEBUG:           Downloaded and cached: '%s' -> '%s'", posterURL, cacheURL))
		showInfo(fmt.Sprintf("theTVDB DEBUG:           Storing in cache with title='%s'", title))
	}

	// Use the original XMLTV title as the cache key (no cleaning/parsing)
	// Check if this is actually a new poster
	isNewPoster := true
	if existingEntry, exists := tvdbLookupCache[title]; exists {
		if existingEntry.PosterURL == posterURL {
			isNewPoster = false
		}
	}

	tvdbLookupCache[title] = &TVDBLookupEntry{
		PosterURL: posterURL,
		CacheURL:  cacheURL,
		Timestamp: time.Now(),
		TTL:       time.Duration(Settings.TvdbCacheExpiry) * time.Hour,
	}

	if System.Dev {
		showInfo(fmt.Sprintf("theTVDB DEBUG:           Successfully cached poster for '%s' (title='%s'): %s -> %s", title, title, posterURL, cacheURL))
	}

	// Mark that we have a new poster for EPG regeneration
	if isNewPoster {
		markNewPosterAvailable()
		if System.Dev {
			showInfo(fmt.Sprintf("theTVDB: Marked new poster for EPG regeneration: '%s'", title))
		}
	}
}

// cleanupExpiredTVDBCache removes expired entries from the lookup cache - DISABLED FOR TESTING
func cleanupExpiredTVDBCache() {
	// Temporarily disabled to prevent cache cleanup during testing
	if System.Dev {
		showInfo("theTVDB: Cache cleanup disabled during testing")
	}
	return

	/*tvdbLookupCacheMutex.Lock()
	defer tvdbLookupCacheMutex.Unlock()

	now := time.Now()
	var expiredCount int
	for title, entry := range tvdbLookupCache {
		if now.Sub(entry.Timestamp) > entry.TTL {
			delete(tvdbLookupCache, title)
			expiredCount++
		}
	}

	if expiredCount > 0 {
		showInfo(fmt.Sprintf("theTVDB: Cleaned up %d expired cache entries", expiredCount))
	}*/
}

// checkRecentSearch checks if we've searched for this title recently
func checkRecentSearch(title string) bool {
	recentSearchesMutex.RLock()
	defer recentSearchesMutex.RUnlock()

	if lastSearch, exists := recentSearches[title]; exists {
		if time.Since(lastSearch) < recentSearchWindow {
			return true // Recently searched
		}
	}
	return false
}

// markRecentSearch marks a title as recently searched
func markRecentSearch(title string) {
	recentSearchesMutex.Lock()
	defer recentSearchesMutex.Unlock()

	recentSearches[title] = time.Now()

	// Clean up old entries to prevent memory leak
	cutoff := time.Now().Add(-recentSearchWindow)
	for key, timestamp := range recentSearches {
		if timestamp.Before(cutoff) {
			delete(recentSearches, key)
		}
	}
}

// isSearchInProgress checks if a search is currently in progress for this title
func isSearchInProgress(title string) bool {
	inProgressSearchesMutex.RLock()
	defer inProgressSearchesMutex.RUnlock()
	return inProgressSearches[title]
}

// markSearchInProgress marks a search as in progress
func markSearchInProgress(title string) bool {
	inProgressSearchesMutex.Lock()
	defer inProgressSearchesMutex.Unlock()

	if inProgressSearches[title] {
		return false // Already in progress
	}
	inProgressSearches[title] = true
	return true // Successfully marked as in progress
}

// markSearchComplete marks a search as complete
func markSearchComplete(title string) {
	inProgressSearchesMutex.Lock()
	defer inProgressSearchesMutex.Unlock()
	delete(inProgressSearches, title)
}

// enforceRateLimit ensures we don't make requests too frequently
func enforceRateLimit() {
	lastRequestMutex.Lock()
	defer lastRequestMutex.Unlock()

	timeSinceLastRequest := time.Since(lastRequestTime)
	if timeSinceLastRequest < minRequestInterval {
		sleepTime := minRequestInterval - timeSinceLastRequest
		time.Sleep(sleepTime)
	}
	lastRequestTime = time.Now()
}

// Failed lookup cache management functions

// isFailedLookup checks if this title has recently failed a lookup
func isFailedLookup(title string) bool {
	failedLookupCacheMutex.RLock()
	defer failedLookupCacheMutex.RUnlock()

	if failedTime, exists := failedLookupCache[title]; exists {
		return time.Since(failedTime) < failedLookupWindow
	}
	return false
}

// markFailedLookup marks a title as having failed a lookup
func markFailedLookup(title string) {
	failedLookupCacheMutex.Lock()
	defer failedLookupCacheMutex.Unlock()

	failedLookupCache[title] = time.Now()

	// Clean up old entries to prevent memory leak
	cutoff := time.Now().Add(-failedLookupWindow)
	for key, timestamp := range failedLookupCache {
		if timestamp.Before(cutoff) {
			delete(failedLookupCache, key)
		}
	}
}

// clearFailedLookup removes a title from the failed lookup cache (e.g., after successful lookup)
func clearFailedLookup(title string) {
	failedLookupCacheMutex.Lock()
	defer failedLookupCacheMutex.Unlock()
	delete(failedLookupCache, title)
}

// isRecentSearch checks if we've searched for this title recently
func isRecentSearch(title string) bool {
	recentSearchesMutex.RLock()
	defer recentSearchesMutex.RUnlock()

	if lastSearchTime, exists := recentSearches[title]; exists {
		return time.Since(lastSearchTime) < recentSearchWindow
	}
	return false
}

// TestTVDBConnection tests the connection to theTVDB API
func TestTVDBConnection(apiKey string) error {
	testClient := &TVDBClient{
		apiKey: strings.TrimSpace(apiKey),
		httpClient: &http.Client{
			Timeout: TVDBTimeout,
		},
	}

	err := testClient.authenticateTVDB()
	if err != nil {
		return fmt.Errorf("theTVDB connection test failed: %v", err)
	}

	return nil
}

// EnableDevModeForTesting enables development mode for testing purposes
// This function is intended for use by test scripts only
func EnableDevModeForTesting() {
	System.Dev = true
}

// GetTVDBStats returns statistics about the theTVDB cache
func GetTVDBStats() map[string]interface{} {
	tvdbLookupCacheMutex.RLock()
	defer tvdbLookupCacheMutex.RUnlock()

	stats := make(map[string]interface{})
	stats["poster_cache_entries"] = len(tvdbLookupCache)
	stats["api_initialized"] = tvdbClient != nil

	if tvdbClient != nil {
		tvdbClient.mutex.RLock()
		stats["token_valid"] = time.Now().Before(tvdbClient.tokenExpiry)
		tvdbClient.mutex.RUnlock()
	}

	return stats
}

// EPG Integration helper functions

// logEPGProgramInfo logs detailed information about the EPG program being searched
func logEPGProgramInfo(program *Program, xepgChannel XEPGChannelStruct) {
	if program == nil {
		showInfo("theTVDB: EPG program is nil")
		return
	}

	var title string
	if len(program.Title) > 0 {
		title = program.Title[0].Value
	}

	var description string
	if len(program.Desc) > 0 {
		description = program.Desc[0].Value
	}

	// Extract year from EPG data
	year := extractYearFromProgram(program)

	// Extract category information
	var categories []string
	for _, cat := range program.Category {
		categories = append(categories, cat.Value)
	}

	// Extract season/episode information
	seasonEpisode := extractSeasonEpisodeFromProgram(program)

	// Extract duration
	durationMinutes := extractProgramDuration(program)
	var durationStr string
	if durationMinutes > 0 {
		hours := durationMinutes / 60
		minutes := durationMinutes % 60
		if hours > 0 {
			durationStr = fmt.Sprintf("%dh %dm", hours, minutes)
		} else {
			durationStr = fmt.Sprintf("%dm", minutes)
		}
	}

	if System.Dev {
		showInfo(fmt.Sprintf("theTVDB: Searching for EPG program - Title: '%s', Year: %s, Categories: %v, Season/Episode: %s, Duration: %s, Channel: %s",
			title, year, categories, seasonEpisode, durationStr, xepgChannel.XmltvFile))

		if description != "" && len(description) > 100 {
			showInfo(fmt.Sprintf("theTVDB: Program description (first 100 chars): %s...", description[:100]))
		} else if description != "" {
			showInfo(fmt.Sprintf("theTVDB: Program description: %s", description))
		}
	}
}

// GetTVDBPosterForProgram attempts to get a poster URL for a program from theTVDB
func GetTVDBPosterForProgram(program *Program, xepgChannel XEPGChannelStruct) string {
	if tvdbClient == nil {
		initializeTVDBClientIfNeeded()
		if tvdbClient == nil {
			return ""
		}
	}

	// Get the program title
	var title string
	if len(program.Title) > 0 {
		title = program.Title[0].Value
	}
	if title == "" {
		return ""
	}

	// Filter out inappropriate search terms
	if !shouldSearchTitle(title) {
		return ""
	}

	// Check cache first for this exact title
	if cachedURL := GetCachedPosterURL(title); cachedURL != "" {
		if System.Dev {
			showInfo(fmt.Sprintf("theTVDB: Using cached poster for '%s': %s", title, cachedURL))
		}
		return cachedURL
	}

	// Check if this title has recently failed a lookup
	if isFailedLookup(title) {
		if System.Dev {
			showInfo(fmt.Sprintf("theTVDB: Skipping lookup for '%s' - recently failed", title))
		}
		return ""
	}

	// Log the start of poster lookup (dev mode only)
	if System.Dev {
		showInfo(fmt.Sprintf("theTVDB: Starting poster lookup for '%s'", title))
	}

	// Log EPG program information before searching (dev mode only)
	if System.Dev {
		logEPGProgramInfo(program, xepgChannel)
	}

	// Detect content type (movie vs series)
	contentType := detectContentType(program, xepgChannel)
	if System.Dev {
		showInfo(fmt.Sprintf("theTVDB: Detected content type: '%s'", contentType))
	}

	var searchResults []TVDBSearchResult
	var err error

	// Enforce rate limiting before making API calls
	enforceRateLimit()

	// Search based on detected content type with improved logic
	switch contentType {
	case "series":
		// If we detected series (has episode info), search series first
		searchResults, err = tvdbClient.searchTVDBSeries(title)

		// If no series found, try movies as fallback
		if err != nil || len(searchResults) == 0 {
			enforceRateLimit()
			if System.Dev {
				showInfo(fmt.Sprintf("theTVDB: No series found, trying movies for '%s'", title))
			}
			searchResults, err = tvdbClient.searchTVDBMovies(title)
		}
	default:
		// Default: search movies first (less ambiguous), then series if no results
		searchResults, err = tvdbClient.searchTVDBMovies(title)

		// If no movies found, try series as fallback
		if err != nil || len(searchResults) == 0 {
			enforceRateLimit()
			if System.Dev {
				showInfo(fmt.Sprintf("theTVDB: No movies found, trying series for '%s'", title))
			}
			searchResults, err = tvdbClient.searchTVDBSeries(title)
		}
	}

	if err != nil || len(searchResults) == 0 {
		if System.Dev {
			showInfo(fmt.Sprintf("theTVDB: No results found for '%s'", title))
		}
		markFailedLookup(title)
		return ""
	}

	if System.Dev {
		showInfo(fmt.Sprintf("theTVDB: Found %d search results for '%s'", len(searchResults), title))
	}

	// If we have results, try to find the best match using EPG metadata
	bestResult := selectBestSearchResult(searchResults, program, contentType)
	if bestResult == nil {
		if System.Dev {
			showInfo(fmt.Sprintf("theTVDB: No suitable match found for '%s' after scoring", title))
		}
		markFailedLookup(title)
		return ""
	}

	showInfo(fmt.Sprintf("theTVDB: Found poster for '%s' -> '%s'", title, bestResult.Name))

	// Use the image URL directly from search results if available
	if bestResult.ImageURL != "" {
		if System.Dev {
			showInfo(fmt.Sprintf("theTVDB: Using poster from search results for '%s': %s", title, bestResult.ImageURL))
		}
		setCachedPosterURL(title, bestResult.ImageURL)
		// Clear any failed lookup entry since we succeeded
		clearFailedLookup(title)
		// Return the cached URL, not the original TheTVDB URL
		if cachedURL := GetCachedPosterURL(title); cachedURL != "" {
			return cachedURL
		}
		return bestResult.ImageURL // Fallback to original if cache failed
	}

	// Fallback: try to get posters from separate API call if no image URL in search results
	enforceRateLimit()
	posters, err := tvdbClient.getTVDBPosters(bestResult.ID, bestResult.Type)
	if err != nil || len(posters) == 0 {
		showInfo(fmt.Sprintf("theTVDB: No posters found for '%s' (ID: %s)", title, bestResult.ID))
		markFailedLookup(title)
		return ""
	}

	showInfo(fmt.Sprintf("theTVDB: Found %d posters for '%s' (ID: %s)", len(posters), title, bestResult.ID))

	// Select the best poster based on user preferences
	selectedPoster := selectBestPoster(posters)
	if selectedPoster != nil {
		if System.Dev {
			showInfo(fmt.Sprintf("theTVDB: Selected poster type '%s' (lang: %s, score: %d) for '%s': %s",
				selectedPoster.Type, selectedPoster.Language, selectedPoster.Score, title, selectedPoster.URL))
		}
		setCachedPosterURL(title, selectedPoster.URL)
		// Clear any failed lookup entry since we succeeded
		clearFailedLookup(title)
		// Return the cached URL, not the original TheTVDB URL
		if cachedURL := GetCachedPosterURL(title); cachedURL != "" {
			return cachedURL
		}
		return selectedPoster.URL // Fallback to original if cache failed
	}

	// No suitable poster found
	markFailedLookup(title)
	return ""
}

// selectBestSearchResult chooses the best search result from multiple options using EPG metadata
func selectBestSearchResult(searchResults []TVDBSearchResult, program *Program, contentType string) *TVDBSearchResult {
	if len(searchResults) == 0 {
		return nil
	}

	// Extract metadata from EPG for matching
	epgYear := extractYearFromProgram(program)
	epgSeasonEpisode := extractSeasonEpisodeFromProgram(program)
	programCategory := extractCategoryFromProgram(program)

	if System.Dev {
		showInfo(fmt.Sprintf("theTVDB: EPG metadata - Year: %s, Season/Episode: %s, Category: %s",
			epgYear, epgSeasonEpisode, programCategory))
	}

	var bestMatch *TVDBSearchResult
	var bestScore int
	const minimumScoreThreshold = 500 // Increased significantly to require much better matches

	for i := range searchResults {
		result := &searchResults[i]
		score := calculateMatchScore(result, epgYear, epgSeasonEpisode, programCategory, contentType, program.Title[0].Value, extractProgramDuration(program))

		if System.Dev {
			showInfo(fmt.Sprintf("theTVDB: Result %d - '%s' (Type: %s, Year: %s) Score: %d",
				i+1, result.Name, result.Type, result.Year, score))
		}

		if score > bestScore {
			bestScore = score
			bestMatch = result
		}
	}

	// Only return a match if it meets the minimum threshold
	if bestMatch != nil && bestScore >= minimumScoreThreshold {
		if System.Dev {
			showInfo(fmt.Sprintf("theTVDB: Best match selected with score %d", bestScore))
		}
		return bestMatch
	}

	// If no good match found, log the issue and return nil
	if bestMatch != nil {
		if System.Dev {
			showInfo(fmt.Sprintf("theTVDB: Best match score %d is below threshold %d, rejecting match for '%s'",
				bestScore, minimumScoreThreshold, program.Title[0].Value))
		}
	} else {
		showInfo(fmt.Sprintf("theTVDB: No matches found for '%s'", program.Title[0].Value))
	}

	return nil
}

// extractYearFromProgram extracts year information from EPG program data
func extractYearFromProgram(program *Program) string {
	// Check program date field
	if program.Date != "" {
		return program.Date
	}

	// Look for year in description
	if len(program.Desc) > 0 {
		desc := program.Desc[0].Value
		// Look for 4-digit years (1900-2099)
		yearRegex := regexp.MustCompile(`\b(19|20)\d{2}\b`)
		if matches := yearRegex.FindStringSubmatch(desc); len(matches) > 0 {
			return matches[0]
		}
	}

	// Look for year in title (sometimes movies have year in title)
	if len(program.Title) > 0 {
		title := program.Title[0].Value
		yearRegex := regexp.MustCompile(`\b(19|20)\d{2}\b`)
		if matches := yearRegex.FindStringSubmatch(title); len(matches) > 0 {
			return matches[0]
		}
	}

	return ""
}

// extractSeasonEpisodeFromProgram extracts season/episode information from EPG
func extractSeasonEpisodeFromProgram(program *Program) string {
	if len(program.Desc) > 0 {
		desc := program.Desc[0].Value
		// Look for patterns like "S01E01", "S1E1", "Season 1 Episode 1"
		patterns := []string{
			`S(\d+)E(\d+)`,
			`Season\s+(\d+)\s+Episode\s+(\d+)`,
			`S(\d+)\s+E(\d+)`,
		}

		for _, pattern := range patterns {
			regex := regexp.MustCompile(`(?i)` + pattern)
			if matches := regex.FindStringSubmatch(desc); len(matches) >= 3 {
				return fmt.Sprintf("S%sE%s", matches[1], matches[2])
			}
		}
	}

	return ""
}

// extractCategoryFromProgram gets the category from EPG data
func extractCategoryFromProgram(program *Program) string {
	if len(program.Category) > 0 {
		return strings.ToLower(program.Category[0].Value)
	}
	return ""
}

// calculateMatchScore calculates how well a search result matches the EPG data
func calculateMatchScore(result *TVDBSearchResult, epgYear, epgSeasonEpisode, epgCategory, contentType string, programTitle string, programDurationMinutes int) int {
	score := 0

	// Base score for having a result
	score += 10

	// HIGHEST PRIORITY: Title matching
	resultTitleLower := strings.ToLower(strings.TrimSpace(result.Name))
	programTitleLower := strings.ToLower(strings.TrimSpace(programTitle))

	isExactMatch := resultTitleLower == programTitleLower

	// More strict partial matching - require significant overlap, not just word fragments
	var hasGoodPartialMatch bool
	if len(programTitleLower) >= 3 && len(resultTitleLower) >= 3 {
		// For short titles, require exact match or very close match
		if len(programTitleLower) <= 8 {
			hasGoodPartialMatch = strings.Contains(resultTitleLower, programTitleLower) ||
				strings.Contains(programTitleLower, resultTitleLower)
		} else {
			// For longer titles, require more substantial overlap
			hasGoodPartialMatch = (strings.Contains(resultTitleLower, programTitleLower) ||
				strings.Contains(programTitleLower, resultTitleLower)) &&
				(float64(len(programTitleLower))/float64(len(resultTitleLower)) > 0.3)
		}
	}

	if isExactMatch {
		score += 1000 // Exact title match gets highest priority
	} else if hasGoodPartialMatch {
		score += 50 // Good partial match
	} else {
		// NO good title similarity - heavily penalize
		score -= 200 // Increased penalty for poor matches
		if System.Dev {
			showInfo(fmt.Sprintf("theTVDB: No title similarity between '%s' and '%s', applying heavy penalty", programTitle, result.Name))
		}
	}

	// Duration filtering for movies - if program is too short, heavily penalize
	if strings.EqualFold(result.Type, "movie") && programDurationMinutes > 0 {
		if programDurationMinutes < 60 {
			score -= 50 // Very short programs are unlikely to be movies
		} else if programDurationMinutes < 90 {
			score -= 20 // Short programs are less likely to be movies
		} else if programDurationMinutes >= 90 && programDurationMinutes <= 240 {
			score += 20 // Good movie duration range
		}
	}

	// Bonus for matching the detected content type from EPG data
	if contentType != "" && strings.EqualFold(result.Type, contentType) {
		if isExactMatch {
			score += 15 // Smaller bonus for exact matches since they already have high score
		} else if hasGoodPartialMatch {
			score += 30 // Larger bonus for partial matches to help differentiate
		}
	}

	// Bonus for matching year
	if epgYear != "" && result.Year != "" {
		if epgYear == result.Year {
			if isExactMatch {
				score += 20 // Smaller bonus for exact matches
			} else if hasGoodPartialMatch {
				score += 40 // Larger bonus for partial matches
			}
		} else {
			// Check if years are close (within 1-2 years for re-runs, different releases)
			if yearDiff := abs(parseInt(epgYear) - parseInt(result.Year)); yearDiff <= 2 {
				if hasGoodPartialMatch {
					score += 15 // Only give year bonus if there's some title match
				}
			}
		}
	}

	// Bonus for appropriate category matching
	if epgCategory != "" {
		switch epgCategory {
		case "movie", "film":
			if strings.EqualFold(result.Type, "movie") {
				score += 20
			}
		case "series", "drama", "comedy", "action":
			if strings.EqualFold(result.Type, "series") {
				score += 20
			}
		case "kids", "children":
			if strings.EqualFold(result.Type, "series") {
				score += 10 // Kids shows are usually series
			}
		}
	}

	// Small bonus for more recent content (much reduced priority)
	if result.Year != "" {
		year := parseInt(result.Year)
		if year >= 2020 {
			score += 2
		} else if year >= 2010 {
			score += 1
		}
	}

	// Small penalty for very old content when we don't have specific year info
	if epgYear == "" && result.Year != "" {
		year := parseInt(result.Year)
		if year < 1970 {
			score -= 5
		}
	}

	return score
}

// Helper function to convert string to int, returns 0 if invalid
func parseInt(s string) int {
	if i, err := strconv.Atoi(s); err == nil {
		return i
	}
	return 0
}

// Helper function to get absolute value
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// Helper function to get minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// initializeTVDBClientIfNeeded initializes the theTVDB client if it hasn't been initialized
func initializeTVDBClientIfNeeded() {
	if tvdbClient == nil && Settings.TvdbApiKey != "" {
		err := InitializeTVDB(Settings.TvdbApiKey)
		if err != nil {
			ShowError(fmt.Errorf("theTVDB: Failed to initialize client: %v", err), 0)
		}
	}
}

// shouldSearchTitle determines if a title is worth searching for
func shouldSearchTitle(title string) bool {
	if title == "" {
		return false
	}

	titleLower := strings.ToLower(strings.TrimSpace(title))

	// Skip obvious non-content titles
	skipTerms := []string{
		"paid programming",
		"infomercial",
		"advertisement",
		"commercial",
		"shop at home",
		"teleshopping",
		"home shopping",
		"qvc",
		"hsn",
		"shopping",
		"test pattern",
		"technical difficulties",
		"no signal",
		"off air",
		"sign off",
		"color bars",
		"maintenance",
		"under eye bags",
		"nutrisystem",
		"prettylitter",
		"downright delicious",
		"a better pain pill",
		"ready for a safer shower",
		"reduce swelling",
		"freedom to leave the house",
		"h2o x5",
		"shark powerdetect",
		"urban indie film block",
		"cut to it",
		"new! shark",
		"jacuzzi",
		"inogen portable oxygen",
		"steam cleaner",
	}

	for _, term := range skipTerms {
		if strings.Contains(titleLower, term) {
			return false
		}
	}

	// Skip titles that are obviously promotional (too much punctuation, all caps, etc.)
	if strings.Count(title, "!") > 2 || strings.Count(title, "?") > 1 {
		return false
	}

	// Skip very short titles that are likely not real content
	if len(strings.TrimSpace(title)) < 3 {
		return false
	}

	return true
}

// detectContentType tries to determine if the program is a movie or series based on EPG data
func detectContentType(program *Program, xepgChannel XEPGChannelStruct) string {
	// First, check for definitive series indicators in description
	if len(program.Desc) > 0 {
		desc := program.Desc[0].Value
		// Look for season/episode patterns - these are very reliable indicators of series
		patterns := []string{
			`S\d+E\d+`,      // S01E01
			`S\d+\s+E\d+`,   // S01 E01
			`Season\s+\d+`,  // Season 1
			`Episode\s+\d+`, // Episode 1
			`\d+x\d+`,       // 1x01
		}

		for _, pattern := range patterns {
			if matched, _ := regexp.MatchString(`(?i)`+pattern, desc); matched {
				return "series"
			}
		}
	}

	// Check if subtitle suggests episode information
	if len(program.SubTitle) > 0 {
		subtitle := program.SubTitle[0].Value
		// If subtitle exists and doesn't look like a movie tagline, it's likely a series
		// Movie subtitles are usually short taglines, series subtitles are episode titles
		if len(subtitle) > 5 && !strings.Contains(strings.ToLower(subtitle), "the movie") {
			return "series"
		}
	}

	// Check program duration (movies are typically longer than 90 minutes)
	if program.Start != "" && program.Stop != "" {
		startTime, err1 := parseEPGTime(program.Start)
		stopTime, err2 := parseEPGTime(program.Stop)

		if err1 == nil && err2 == nil {
			duration := stopTime.Sub(startTime)
			if duration.Minutes() >= 90 {
				// Programs longer than 90 minutes are likely movies
				return "movie"
			} else if duration.Minutes() <= 30 {
				// Programs 30 minutes or less are likely series episodes
				return "series"
			}
		}
	}

	// Check EPG category for specific indicators
	if len(program.Category) > 0 {
		category := strings.ToLower(program.Category[0].Value)
		if System.Dev {
			showInfo(fmt.Sprintf("theTVDB: Content type detection - Category: '%s'", category))
		}
		switch category {
		case "movie", "film":
			if System.Dev {
				showInfo("theTVDB: Detected as movie based on category")
			}
			return "movie"
		case "series", "tv series", "television series":
			if System.Dev {
				showInfo("theTVDB: Detected as series based on category")
			}
			return "series"
		case "sports", "news", "kids", "children", "documentary series":
			if System.Dev {
				showInfo("theTVDB: Detected as series based on category (series-like content)")
			}
			return "series" // These are typically series-like
		case "drama", "comedy", "action":
			// These could be either - rely on other indicators
			// If we have episode info or short duration, lean toward series
			if len(program.SubTitle) > 0 {
				if System.Dev {
					showInfo("theTVDB: Detected as series based on category + subtitle")
				}
				return "series"
			}
		}
	}

	// Default to movie-first search for ambiguous cases
	// This is because movies are less ambiguous and easier to match correctly
	return "movie"
}

// parseEPGTime parses EPG timestamp format
func parseEPGTime(timeStr string) (time.Time, error) {
	// EPG times are typically in format: "20250615020000 +0000"
	if len(timeStr) >= 14 {
		// Try to parse the basic format: YYYYMMDDHHMMSS
		layout := "20060102150405"
		timeOnly := timeStr[:14]
		return time.Parse(layout, timeOnly)
	}
	return time.Time{}, fmt.Errorf("invalid time format")
}

// extractProgramDuration calculates the duration of the program in minutes
func extractProgramDuration(program *Program) int {
	if program.Start == "" || program.Stop == "" {
		return 0
	}

	// Parse the time format "20250615020000 +0000"
	layout := "20060102150405 -0700"

	startTime, err := time.Parse(layout, program.Start)
	if err != nil {
		return 0
	}

	stopTime, err := time.Parse(layout, program.Stop)
	if err != nil {
		return 0
	}

	duration := stopTime.Sub(startTime)
	return int(duration.Minutes())
}

// SaveTVDBCacheOnShutdown cleans up expired cache entries - call this on application shutdown
func SaveTVDBCacheOnShutdown() {
	if tvdbClient != nil {
		showInfo("theTVDB: Cache cleanup disabled for testing...")
		// cleanupExpiredTVDBCache() // DISABLED FOR TESTING
	}
}

// markNewPosterAvailable sets flag when new poster is cached
func markNewPosterAvailable() {
	hasNewPosterMutex.Lock()
	defer hasNewPosterMutex.Unlock()
	hasNewPosters = true
}

// checkAndClearNewPosters returns and clears the new poster flag
func checkAndClearNewPosters() bool {
	hasNewPosterMutex.Lock()
	defer hasNewPosterMutex.Unlock()

	result := hasNewPosters
	hasNewPosters = false
	return result
}
