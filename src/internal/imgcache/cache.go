package imgcache

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Cache : Cache strcut
type Cache struct {
	path       string
	cacheURL   string
	caching    bool
	debugLevel int
	images     map[string]string
	Queue      []string
	Cache      []string
	Image      imageFunc
	sync.RWMutex
}

type imageFunc struct {
	GetURL              func(string, string, string, bool, int, string) string
	Caching             func()
	Remove              func()
	DownloadImmediately func(string, string, string, bool, int, string) string
}

// New : New cahce
func New(path, cacheURL string, caching bool, debugLevel int) (c *Cache, err error) {

	c = &Cache{}

	c.images = make(map[string]string)
	c.path = path
	c.cacheURL = cacheURL
	c.caching = caching
	c.debugLevel = debugLevel
	c.Queue = []string{}
	c.Cache = []string{}

	var queue []string

	c.Image.GetURL = func(src string, http_domain string, http_port string, force_https bool, https_port int, https_domain string) (cacheURL string) {

		c.Lock()
		defer c.Unlock()

		src = strings.Trim(src, "\r\n")

		if !c.caching {
			return src
		}

		u, err := url.Parse(src)

		if err != nil || len(filepath.Ext(u.Path)) == 0 {
			return src
		}

		src_filtered := strings.Split(src, "?")
		var filename = fmt.Sprintf("%s%s", strToMD5(src_filtered[0]), filepath.Ext(u.Path))

		if cacheURL, ok := c.images[filename]; ok {
			if c.caching && force_https {
				u, err := url.Parse(cacheURL)
				if err == nil {
					cacheURL = fmt.Sprintf("https://%s:%d%s", https_domain, https_port, u.Path)
				}
			} else if c.caching && http_domain != "" {
				u, err := url.Parse(cacheURL)
				if err == nil {
					var baseUrl = ""
					if strings.Contains(http_domain, ":") {
						baseUrl = http_domain
					} else {
						baseUrl = fmt.Sprintf("%s:%s", http_domain, http_port)
					}
					cacheURL = fmt.Sprintf("http://%s%s", baseUrl, u.Path)
				}
			}
			return cacheURL
		}

		if indexOfString(filename, c.Cache) == -1 {
			if indexOfString(src, c.Queue) == -1 {
				c.Queue = append(c.Queue, src)
			}

		} else {
			c.images[filename] = c.cacheURL + filename
			src = c.cacheURL + filename
		}

		return src
	}

	c.Image.Caching = func() {

		c.Lock()
		defer c.Unlock()

		var filename string

		for _, src := range c.Queue {

			resp, err := http.Get(src)
			if err != nil {
				continue
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				continue
			}

			src_filtered := strings.Split(src, "?")
			filename = fmt.Sprintf("%s%s%s", c.path, strToMD5(src_filtered[0]), filepath.Ext(src_filtered[0]))

			file, err := os.Create(filename)
			if err != nil {
				continue
			}

			defer file.Close()

			_, err = io.Copy(file, resp.Body)
			if err != nil {
				continue
			}

			u, err := url.Parse(src_filtered[0])
			if err == nil {
				c.images[fmt.Sprintf("%s%s", strToMD5(src_filtered[0]), filepath.Ext(u.Path))] = c.cacheURL + filename
			}

			queue = append(queue, src_filtered[0])

		}

		for _, q := range queue {
			c.Queue = removeStringFromSlice(q, c.Queue)
		}

	}

	c.Image.Remove = func() {

		c.Lock()
		defer c.Unlock()

		files, err := os.ReadDir(c.path)
		if err != nil {
			return
		}

		for _, file := range files {

			switch c.caching {

			case true:
				if _, ok := c.images[file.Name()]; !ok {
					os.RemoveAll(c.path + file.Name())
				}

			case false:
				os.RemoveAll(c.path + file.Name())
			}

		}

	}

	// DownloadImmediately downloads an image immediately and returns the local cached URL
	c.Image.DownloadImmediately = func(src string, http_domain string, http_port string, force_https bool, https_port int, https_domain string) (cacheURL string) {

		c.Lock()
		defer c.Unlock()

		src = strings.Trim(src, "\r\n")

		if !c.caching {
			if c.debugLevel >= 2 {
				fmt.Printf("DEBUG: DownloadImmediately - caching disabled, returning original URL: %s\n", src)
			}
			return src
		}

		u, err := url.Parse(src)
		if err != nil || len(filepath.Ext(u.Path)) == 0 {
			if c.debugLevel >= 2 {
				fmt.Printf("DEBUG: DownloadImmediately - invalid URL or no extension, returning original: %s (err: %v)\n", src, err)
			}
			return src
		}

		src_filtered := strings.Split(src, "?")
		var filename = fmt.Sprintf("%s%s", strToMD5(src_filtered[0]), filepath.Ext(u.Path))

		if c.debugLevel >= 2 {
			fmt.Printf("DEBUG: DownloadImmediately - processing URL: %s\n", src)
			fmt.Printf("DEBUG: DownloadImmediately - filtered URL: %s\n", src_filtered[0])
			fmt.Printf("DEBUG: DownloadImmediately - generated filename: %s\n", filename)
		}

		// Check if already cached
		if cachedURL, ok := c.images[filename]; ok {
			if c.debugLevel >= 2 {
				fmt.Printf("DEBUG: DownloadImmediately - found in cache: %s -> %s\n", filename, cachedURL)
			}
			if force_https {
				u, err := url.Parse(cachedURL)
				if err == nil {
					return fmt.Sprintf("https://%s:%d%s", https_domain, https_port, u.Path)
				}
			} else if http_domain != "" {
				u, err := url.Parse(cachedURL)
				if err == nil {
					var baseUrl = ""
					if strings.Contains(http_domain, ":") {
						baseUrl = http_domain
					} else {
						baseUrl = fmt.Sprintf("%s:%s", http_domain, http_port)
					}
					return fmt.Sprintf("http://%s%s", baseUrl, u.Path)
				}
			}
			return cachedURL
		}

		if c.debugLevel >= 2 {
			fmt.Printf("DEBUG: DownloadImmediately - not in cache, downloading: %s\n", src)
		}

		// Download immediately
		resp, err := http.Get(src)
		if err != nil {
			if c.debugLevel >= 2 {
				fmt.Printf("DEBUG: DownloadImmediately - HTTP GET failed: %v\n", err)
			}
			return src // Return original URL if download fails
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			if c.debugLevel >= 2 {
				fmt.Printf("DEBUG: DownloadImmediately - HTTP status not OK: %d\n", resp.StatusCode)
			}
			return src // Return original URL if download fails
		}

		filepath_name := fmt.Sprintf("%s%s%s", c.path, strToMD5(src_filtered[0]), filepath.Ext(src_filtered[0]))

		if c.debugLevel >= 2 {
			fmt.Printf("DEBUG: DownloadImmediately - saving to file: %s\n", filepath_name)
		}

		file, err := os.Create(filepath_name)
		if err != nil {
			if c.debugLevel >= 2 {
				fmt.Printf("DEBUG: DownloadImmediately - file creation failed: %v\n", err)
			}
			return src // Return original URL if file creation fails
		}
		// Don't defer close here since we need to close before verification

		bytesWritten, err := io.Copy(file, resp.Body)
		if err != nil {
			if c.debugLevel >= 2 {
				fmt.Printf("DEBUG: DownloadImmediately - file write failed: %v\n", err)
			}
			return src // Return original URL if write fails
		}

		// Force file sync to disk
		file.Sync()
		file.Close() // Close file before verification

		if c.debugLevel >= 2 {
			fmt.Printf("DEBUG: DownloadImmediately - file saved successfully: %s (%d bytes)\n", filepath_name, bytesWritten)
		}

		// Verify file exists after creation
		if _, err := os.Stat(filepath_name); os.IsNotExist(err) {
			if c.debugLevel >= 2 {
				fmt.Printf("DEBUG: DownloadImmediately - ERROR: File does not exist after creation: %s\n", filepath_name)
			}
			return src
		} else if err != nil {
			if c.debugLevel >= 2 {
				fmt.Printf("DEBUG: DownloadImmediately - ERROR: Cannot stat file: %v\n", err)
			}
			return src
		} else {
			if c.debugLevel >= 2 {
				fmt.Printf("DEBUG: DownloadImmediately - File verified to exist: %s\n", filepath_name)
			}
		}

		// Update cache
		var local_cache_url = c.cacheURL + filename
		c.images[filename] = local_cache_url

		if c.debugLevel >= 2 {
			fmt.Printf("DEBUG: DownloadImmediately - updated cache: %s -> %s\n", filename, local_cache_url)
		}

		// Also add to the main cache tracking to prevent deletion by Remove()
		if indexOfString(filename, c.Cache) == -1 {
			c.Cache = append(c.Cache, filename)
			if c.debugLevel >= 2 {
				fmt.Printf("DEBUG: DownloadImmediately - added to Cache list: %s\n", filename)
			}
		}

		// Return properly formatted URL
		if force_https {
			u, err := url.Parse(local_cache_url)
			if err == nil {
				return fmt.Sprintf("https://%s:%d%s", https_domain, https_port, u.Path)
			}
		} else if http_domain != "" {
			u, err := url.Parse(local_cache_url)
			if err == nil {
				var baseUrl = ""
				if strings.Contains(http_domain, ":") {
					baseUrl = http_domain
				} else {
					baseUrl = fmt.Sprintf("%s:%s", http_domain, http_port)
				}
				return fmt.Sprintf("http://%s%s", baseUrl, u.Path)
			}
		}

		return local_cache_url
	}

	files, err := os.ReadDir(c.path)
	if err != nil {
		return
	}

	for _, file := range files {
		c.Cache = append(c.Cache, file.Name())
	}

	return
}
