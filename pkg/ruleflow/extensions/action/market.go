// Package action provides VPP action nodes
package action

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/core"
)

// ─────────────────────────────────────────────
//  MarketPriceQueryAction — 电力市场出清价格查询动作
// ─────────────────────────────────────────────

// cacheEntry 缓存条目
type cacheEntry struct {
	Price    float64
	ExpireAt time.Time
}

// MarketPriceQueryAction 电力市场出清价格查询动作
type MarketPriceQueryAction struct {
	IDValue     string `json:"id"`
	Market      string `json:"market"`        // day_ahead / real_time / ancillary
	Product     string `json:"product"`       // energy / capacity / reserve / regulation
	Region      string `json:"region"`        // 区域编码
	CacheTTLSec int    `json:"cache_ttl_sec"` // 缓存有效期（秒）
	OutputTag   string `json:"output_tag"`    // 结果写入的 Tag 名
	Endpoint    string `json:"endpoint"`      // 市场 API 地址

	HTTPClient *http.Client
	Cache      map[string]*cacheEntry
	cacheMu    sync.Mutex
}

// NewMarketPriceQueryAction 创建市场价格查询动作
func NewMarketPriceQueryAction(id, market, product, region string, cacheTTLSec int, outputTag, endpoint string, httpClient *http.Client) *MarketPriceQueryAction {
	if cacheTTLSec == 0 {
		cacheTTLSec = 300
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &MarketPriceQueryAction{
		IDValue:     id,
		Market:      market,
		Product:     product,
		Region:      region,
		CacheTTLSec: cacheTTLSec,
		OutputTag:   outputTag,
		Endpoint:    endpoint,
		HTTPClient:  httpClient,
		Cache:       make(map[string]*cacheEntry),
	}
}

func (a *MarketPriceQueryAction) Execute(ctx context.Context, data core.DataContext) error {
	cacheKey := a.Market + "/" + a.Product + "/" + a.Region

	// 1. 检查缓存
	if a.Cache != nil {
		a.cacheMu.Lock()
		entry, ok := a.Cache[cacheKey]
		a.cacheMu.Unlock()
		if ok && time.Now().Before(entry.ExpireAt) {
			data.SetTag(a.OutputTag, fmt.Sprintf("%f", entry.Price))
			return nil
		}
	}

	// 2. 查询市场 API（如果 endpoint 配置）
	if a.Endpoint == "" {
		// 无 endpoint 时使用模拟价格
		data.SetTag(a.OutputTag, "0.45")
		data.SetTag("_market_source", "mock")
		return nil
	}

	url := fmt.Sprintf("%s/api/v1/price?market=%s&product=%s&region=%s",
		a.Endpoint, a.Market, a.Product, a.Region)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("market_price: create request: %w", err)
	}

	resp, err := a.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("market_price: query failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Price float64 `json:"price"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("market_price: decode response: %w", err)
	}

	// 3. 写入缓存
	if a.Cache != nil {
		a.cacheMu.Lock()
		a.Cache[cacheKey] = &cacheEntry{
			Price:    result.Price,
			ExpireAt: time.Now().Add(time.Duration(a.CacheTTLSec) * time.Second),
		}
		a.cacheMu.Unlock()
	}

	// 4. 写入 DataContext
	data.SetTag(a.OutputTag, fmt.Sprintf("%f", result.Price))
	data.SetTag("_market_source", "api")
	return nil
}

func (a *MarketPriceQueryAction) ID() string   { return a.IDValue }
func (a *MarketPriceQueryAction) Type() string { return "market_price_query" }
func (a *MarketPriceQueryAction) Description() string {
	return fmt.Sprintf("market price query %s/%s/%s", a.Market, a.Product, a.Region)
}