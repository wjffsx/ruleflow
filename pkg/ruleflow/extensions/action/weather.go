// Package action provides VPP action nodes
package action

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/core"
)

// ─────────────────────────────────────────────
//  WeatherQueryAction — 气象数据查询动作
// ─────────────────────────────────────────────

// weatherCacheEntry 气象缓存条目
type weatherCacheEntry struct {
	Data     map[string]float64
	ExpireAt time.Time
}

// WeatherQueryAction 气象数据查询动作
type WeatherQueryAction struct {
	IDValue      string   `json:"id"`
	WeatherAPI   string   `json:"weather_api"`
	Latitude     *float64 `json:"latitude"`
	Longitude    *float64 `json:"longitude"`
	Fields       []string `json:"fields"`
	OutputPrefix string   `json:"output_prefix"`
	CacheTTLSec  int      `json:"cache_ttl_sec"`

	HTTPClient *http.Client
	cache      map[string]*weatherCacheEntry
	cacheMu    sync.Mutex
}

// NewWeatherQueryAction 创建气象数据查询动作
func NewWeatherQueryAction(id, weatherAPI string, lat, lon *float64, fields []string, outputPrefix string, cacheTTLSec int, httpClient *http.Client) *WeatherQueryAction {
	if cacheTTLSec == 0 {
		cacheTTLSec = 600
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &WeatherQueryAction{
		IDValue:      id,
		WeatherAPI:   weatherAPI,
		Latitude:     lat,
		Longitude:    lon,
		Fields:       fields,
		OutputPrefix: outputPrefix,
		CacheTTLSec:  cacheTTLSec,
		HTTPClient:   httpClient,
		cache:        make(map[string]*weatherCacheEntry),
	}
}

func (a *WeatherQueryAction) Execute(ctx context.Context, data core.DataContext) error {
	// 获取经纬度（从配置或 Tag）
	lat := a.Latitude
	lon := a.Longitude
	if lat == nil {
		if latStr := data.GetTag("latitude"); latStr != "" {
			if l, err := parseFloat(latStr); err == nil {
				lat = &l
			}
		}
	}
	if lon == nil {
		if lonStr := data.GetTag("longitude"); lonStr != "" {
			if l, err := parseFloat(lonStr); err == nil {
				lon = &l
			}
		}
	}

	if lat == nil || lon == nil {
		return fmt.Errorf("weather_query: latitude/longitude not configured")
	}

	cacheKey := fmt.Sprintf("%.4f/%.4f", *lat, *lon)

	// 1. 检查缓存
	if a.cache != nil {
		a.cacheMu.Lock()
		entry, ok := a.cache[cacheKey]
		a.cacheMu.Unlock()
		if ok && time.Now().Before(entry.ExpireAt) {
			a.writeToDataContext(data, entry.Data)
			return nil
		}
	}

	// 2. 查询气象 API（如果配置）
	if a.WeatherAPI == "" {
		// 无 API 时使用模拟数据
		mockData := map[string]float64{
			"irradiance":  800.0,
			"temperature": 25.0,
			"wind_speed":  5.0,
		}
		a.writeToDataContext(data, mockData)
		data.SetTag("_weather_source", "mock")
		return nil
	}

	url := fmt.Sprintf("%s?lat=%.4f&lon=%.4f&fields=%s",
		a.WeatherAPI, *lat, *lon, strings.Join(a.Fields, ","))
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := a.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("weather: query failed: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]float64
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("weather: decode: %w", err)
	}

	// 3. 写入缓存
	if a.cache != nil {
		a.cacheMu.Lock()
		a.cache[cacheKey] = &weatherCacheEntry{
			Data:     result,
			ExpireAt: time.Now().Add(time.Duration(a.CacheTTLSec) * time.Second),
		}
		a.cacheMu.Unlock()
	}

	a.writeToDataContext(data, result)
	data.SetTag("_weather_source", "api")
	return nil
}

func (a *WeatherQueryAction) writeToDataContext(data core.DataContext, values map[string]float64) {
	for k, v := range values {
		tagKey := a.OutputPrefix + "." + k
		data.SetTag(tagKey, fmt.Sprintf("%.2f", v))
	}
}

func (a *WeatherQueryAction) ID() string   { return a.IDValue }
func (a *WeatherQueryAction) Type() string { return "weather_query" }
func (a *WeatherQueryAction) Description() string {
	return fmt.Sprintf("weather query fields=%v", a.Fields)
}

// parseFloat 辅助函数
func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}