package config

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// RemoteConfig yapısı - GitHub'dan çekilecek konfigürasyon
type RemoteConfig struct {
	// HTTP istek konfigürasyonu
	Request RequestConfig `json:"request"`
	// API URL'leri
	URLs URLConfig `json:"urls"`
	// Versiyon bilgisi
	Version string `json:"version"`
}

// RequestConfig HTTP istek detaylarını içerir
type RequestConfig struct {
	Headers     map[string]string `json:"headers"`
	UserAgents  []string          `json:"userAgents"`
	ContentType string            `json:"contentType"`
}

// URLConfig API endpoint URL'lerini içerir
type URLConfig struct {
	CoursePickerAPI    string `json:"coursePickerApi"`
	Origin             string `json:"origin"`
	Referer            string `json:"referer"`
	AcademicStatusBase string `json:"academicStatusBase"`
	TermList           string `json:"termList"`
	ProfilePhotoURL    string `json:"profilePhotoUrl"`
}

// ConfigManager remote config'i yönetir
type ConfigManager struct {
	mu             sync.RWMutex
	config         *RemoteConfig
	lastFetch      time.Time
	cacheDuration  time.Duration
	configURL      string
	versionURL     string
	fallbackConfig *RemoteConfig
}

// Default konfigürasyon URL'leri
const (
	DefaultConfigURL     = "https://raw.githubusercontent.com/ITU-BeeHub/BeeHub-backend/main/remote-config.json"
	DefaultVersionURL    = "https://raw.githubusercontent.com/ITU-BeeHub/BeeHub-backend/main/version.txt"
	DefaultCacheDuration = 5 * time.Minute
)

// Default fallback versiyon - hata yönetimi için anlamlı değer
const DefaultVersion = "0.0.0-unavailable"

// NewConfigManager yeni bir ConfigManager oluşturur
func NewConfigManager() *ConfigManager {
	return &ConfigManager{
		cacheDuration:  DefaultCacheDuration,
		configURL:      DefaultConfigURL,
		versionURL:     DefaultVersionURL,
		fallbackConfig: getDefaultFallbackConfig(),
	}
}

// NewConfigManagerWithURLs özel URL'ler ile ConfigManager oluşturur
func NewConfigManagerWithURLs(configURL, versionURL string) *ConfigManager {
	cm := NewConfigManager()
	if configURL != "" {
		cm.configURL = configURL
	}
	if versionURL != "" {
		cm.versionURL = versionURL
	}
	return cm
}

// getDefaultFallbackConfig varsayılan fallback konfigürasyonu döndürür
func getDefaultFallbackConfig() *RemoteConfig {
	return &RemoteConfig{
		Request: RequestConfig{
			Headers: map[string]string{
				"accept":       "application/json, text/plain, */*",
				"Content-Type": "application/json",
			},
			UserAgents: []string{
				"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
				"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36",
				"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
				"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:121.0) Gecko/20100101 Firefox/121.0",
				"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2 Safari/605.1.15",
			},
			ContentType: "application/json",
		},
		URLs: URLConfig{
			CoursePickerAPI:    "https://obs.itu.edu.tr/api/ders-kayit/v21",
			Origin:             "https://obs.itu.edu.tr",
			Referer:            "https://obs.itu.edu.tr/ogrenci/DersKayitIslemleri/DersKayit",
			AcademicStatusBase: "https://obs.itu.edu.tr/api/ogrenci/AkademikDurum",
			TermList:           "https://obs.itu.edu.tr/api/ogrenci/DonemListesi/",
			ProfilePhotoURL:    "https://portal.itu.edu.tr/services/ui/photo.aspx?subsession={subsession}",
		},
		Version: DefaultVersion,
	}
}

// GetConfig konfigürasyonu döndürür (cache'den veya remote'dan)
func (cm *ConfigManager) GetConfig() (*RemoteConfig, error) {
	cm.mu.RLock()
	if cm.config != nil && time.Since(cm.lastFetch) < cm.cacheDuration {
		config := cm.config
		cm.mu.RUnlock()
		return config, nil
	}
	cm.mu.RUnlock()

	return cm.refreshConfig()
}

// refreshConfig konfigürasyonu yeniler
func (cm *ConfigManager) refreshConfig() (*RemoteConfig, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Double-check locking
	if cm.config != nil && time.Since(cm.lastFetch) < cm.cacheDuration {
		return cm.config, nil
	}

	config, err := cm.fetchRemoteConfig()
	if err != nil {
		fmt.Printf("Remote config alınamadı, fallback kullanılıyor: %v\n", err)
		if cm.config != nil {
			return cm.config, nil // Mevcut cache'i kullan
		}
		cm.config = cm.fallbackConfig
		cm.lastFetch = time.Now()
		return cm.fallbackConfig, nil
	}

	cm.config = config
	cm.lastFetch = time.Now()
	return config, nil
}

// fetchRemoteConfig GitHub'dan konfigürasyonu çeker
func (cm *ConfigManager) fetchRemoteConfig() (*RemoteConfig, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(cm.configURL)
	if err != nil {
		return nil, fmt.Errorf("config fetch error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("config fetch failed with status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("config read error: %w", err)
	}

	var config RemoteConfig
	if err := json.Unmarshal(body, &config); err != nil {
		return nil, fmt.Errorf("config parse error: %w", err)
	}

	// Eksik alanları fallback ile doldur
	cm.mergeWithFallback(&config)

	return &config, nil
}

// mergeWithFallback eksik alanları fallback değerleriyle doldurur
func (cm *ConfigManager) mergeWithFallback(config *RemoteConfig) {
	fallback := cm.fallbackConfig

	if len(config.Request.Headers) == 0 {
		config.Request.Headers = fallback.Request.Headers
	}
	if len(config.Request.UserAgents) == 0 {
		config.Request.UserAgents = fallback.Request.UserAgents
	}
	if config.Request.ContentType == "" {
		config.Request.ContentType = fallback.Request.ContentType
	}
	if config.URLs.CoursePickerAPI == "" {
		config.URLs.CoursePickerAPI = fallback.URLs.CoursePickerAPI
	}
	if config.URLs.Origin == "" {
		config.URLs.Origin = fallback.URLs.Origin
	}
	if config.URLs.Referer == "" {
		config.URLs.Referer = fallback.URLs.Referer
	}
	if config.URLs.AcademicStatusBase == "" {
		config.URLs.AcademicStatusBase = fallback.URLs.AcademicStatusBase
	}
	if config.URLs.TermList == "" {
		config.URLs.TermList = fallback.URLs.TermList
	}
	if config.URLs.ProfilePhotoURL == "" {
		config.URLs.ProfilePhotoURL = fallback.URLs.ProfilePhotoURL
	}
	if config.Version == "" {
		config.Version = fallback.Version
	}
}

// GetVersion sadece versiyon bilgisini çeker
func (cm *ConfigManager) GetVersion() (string, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(cm.versionURL)
	if err != nil {
		return DefaultVersion, fmt.Errorf("version fetch error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return DefaultVersion, fmt.Errorf("version fetch failed with status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return DefaultVersion, fmt.Errorf("version read error: %w", err)
	}

	version := string(body)
	if version == "" {
		return DefaultVersion, nil
	}

	return version, nil
}

// GetRandomUserAgent rastgele bir User-Agent döndürür
func (cm *ConfigManager) GetRandomUserAgent() string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	config := cm.config
	if config == nil {
		config = cm.fallbackConfig
	}

	if len(config.Request.UserAgents) == 0 {
		return cm.fallbackConfig.Request.UserAgents[0]
	}

	// Basit round-robin veya random seçim için time-based index
	index := time.Now().UnixNano() % int64(len(config.Request.UserAgents))
	return config.Request.UserAgents[index]
}

// GetHeaders HTTP istekleri için header map'i döndürür
func (cm *ConfigManager) GetHeaders(token string) map[string]string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	config := cm.config
	if config == nil {
		config = cm.fallbackConfig
	}

	headers := make(map[string]string)

	// Base headers'ı kopyala
	for k, v := range config.Request.Headers {
		headers[k] = v
	}

	// Dinamik değerleri ekle
	headers["authorization"] = "Bearer " + token
	headers["origin"] = config.URLs.Origin
	headers["referer"] = config.URLs.Referer
	headers["User-Agent"] = cm.GetRandomUserAgent()

	return headers
}

// GetCoursePickerURL ders seçim API URL'ini döndürür
func (cm *ConfigManager) GetCoursePickerURL() string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if cm.config != nil {
		return cm.config.URLs.CoursePickerAPI
	}
	return cm.fallbackConfig.URLs.CoursePickerAPI
}

// ForceRefresh cache'i bypass edip yeniden çekmeye zorlar
func (cm *ConfigManager) ForceRefresh() (*RemoteConfig, error) {
	cm.mu.Lock()
	cm.lastFetch = time.Time{} // Cache'i invalidate et
	cm.mu.Unlock()

	return cm.refreshConfig()
}

// SetCacheDuration cache süresini değiştirir
func (cm *ConfigManager) SetCacheDuration(duration time.Duration) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.cacheDuration = duration
}

// Global ConfigManager instance
var globalConfigManager *ConfigManager
var configOnce sync.Once

// GetGlobalConfigManager global ConfigManager instance'ı döndürür
func GetGlobalConfigManager() *ConfigManager {
	configOnce.Do(func() {
		globalConfigManager = NewConfigManager()
	})
	return globalConfigManager
}
