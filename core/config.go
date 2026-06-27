package core

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type Config struct {
	CacheDir           string            `json:"cacheDir"`
	DownloadURL        string            `json:"downloadUrl"`
	Timeout            time.Duration     `json:"timeout"`
	MaxRetries         int               `json:"maxRetries"`
	RetryDelay         time.Duration     `json:"retryDelay"`
	DisableCompression bool              `json:"disableCompression"`
	DefaultHeaders     map[string]string `json:"defaultHeaders"`
}

type ConfigManager struct {
	config Config
}

func NewConfigManager(userConfig Config) *ConfigManager {
	home, err := os.UserHomeDir()
	var cacheDir string
	if err == nil {
		cacheDir = filepath.Join(home, ".consultas-de-veiculos-sdk")
	} else {
		cacheDir = filepath.Join(".", ".consultas-de-veiculos-sdk")
	}

	downloadURL := os.Getenv("DOWNLOAD_URL")
	if downloadURL == "" {
		downloadURL = "https://painel.consultasdeveiculos.com/download-postman"
	}

	defaultConfig := Config{
		CacheDir:    cacheDir,
		DownloadURL: downloadURL,
		Timeout:     30 * time.Second,
		MaxRetries:  3,
		RetryDelay:  1 * time.Second,
		DisableCompression: false,
		DefaultHeaders: map[string]string{
			"Accept":       "application/json",
			"Content-Type": "application/json",
		},
	}

	if userConfig.CacheDir != "" {
		defaultConfig.CacheDir = userConfig.CacheDir
	}
	if userConfig.DownloadURL != "" {
		defaultConfig.DownloadURL = userConfig.DownloadURL
	}
	if userConfig.Timeout != 0 {
		defaultConfig.Timeout = userConfig.Timeout
	}
	if userConfig.MaxRetries != 0 {
		defaultConfig.MaxRetries = userConfig.MaxRetries
	}
	if userConfig.RetryDelay != 0 {
		defaultConfig.RetryDelay = userConfig.RetryDelay
	}
	if userConfig.DisableCompression {
		defaultConfig.DisableCompression = true
	}
	if userConfig.DefaultHeaders != nil {
		for k, v := range userConfig.DefaultHeaders {
			defaultConfig.DefaultHeaders[k] = v
		}
	}

	cm := &ConfigManager{config: defaultConfig}
	cm.ensureCacheDir()
	return cm
}

func (cm *ConfigManager) GetCacheDir() string {
	return cm.config.CacheDir
}

func (cm *ConfigManager) GetCachedPostmanPath() string {
	return filepath.Join(cm.config.CacheDir, "postman.json")
}

func (cm *ConfigManager) GetCachedManifestPath() string {
	return filepath.Join(cm.config.CacheDir, "manifest.json")
}

func (cm *ConfigManager) GetResponseCacheDir() string {
	return filepath.Join(cm.config.CacheDir, "cache")
}

func (cm *ConfigManager) GetDownloadURL() string {
	return cm.config.DownloadURL
}

func (cm *ConfigManager) GetTimeout() time.Duration {
	return cm.config.Timeout
}

func (cm *ConfigManager) GetMaxRetries() int {
	return cm.config.MaxRetries
}

func (cm *ConfigManager) GetRetryDelay() time.Duration {
	return cm.config.RetryDelay
}

func (cm *ConfigManager) GetCompression() bool {
	return !cm.config.DisableCompression
}

func (cm *ConfigManager) GetDefaultHeaders() map[string]string {
	return cm.config.DefaultHeaders
}

func (cm *ConfigManager) FindPostmanFile(dir string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}

	re := regexp.MustCompile(`(?i)^Consultas\s*-\s*V[\d.]+\.postman_collection\.json$`)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if re.MatchString(name) {
			return filepath.Join(dir, name)
		}
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.ToLower(entry.Name()) == "postman.json" {
			return filepath.Join(dir, entry.Name())
		}
	}

	return ""
}

func (cm *ConfigManager) ExtractVersionFromFilename(filename string) string {
	re := regexp.MustCompile(`(?i)V([\d.]+)\.postman_collection\.json$`)
	matches := re.FindStringSubmatch(filename)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func (cm *ConfigManager) ensureCacheDir() {
	_ = os.MkdirAll(cm.config.CacheDir, 0755)
	_ = os.MkdirAll(cm.GetResponseCacheDir(), 0755)
}

func (cm *ConfigManager) ClearCache() {
	_ = os.RemoveAll(cm.config.CacheDir)
	cm.ensureCacheDir()
}

func (cm *ConfigManager) HasLocalCache() bool {
	_, errPostman := os.Stat(cm.GetCachedPostmanPath())
	_, errManifest := os.Stat(cm.GetCachedManifestPath())
	return errPostman == nil && errManifest == nil
}
