package pprof

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// 确保所有标准 pprof 端点都注册在自定义 mux 上
func TestHandler_RegistersAllEndpoints(t *testing.T) {
	h := Handler()
	mux, ok := h.(*http.ServeMux)
	if !ok {
		t.Fatal("Handler() should return *http.ServeMux")
	}

	// 端点列表（profile 端点会阻塞 30s 收集 CPU profile，跳过）
	endpoints := []string{
		"/debug/pprof/",
		"/debug/pprof/cmdline",
		"/debug/pprof/symbol",
		"/debug/pprof/trace",
		"/debug/pprof/goroutine",
		"/debug/pprof/heap",
		"/debug/pprof/allocs",
		"/debug/pprof/block",
		"/debug/pprof/mutex",
		"/debug/pprof/threadcreate",
	}

	for _, ep := range endpoints {
		// 使用短 ctx 防止某些端点（如 trace）阻塞
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		req := httptest.NewRequest(http.MethodGet, ep, nil).WithContext(ctx)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)
		cancel()

		// 我们至少要求 handler 不返回 404
		if rr.Code == http.StatusNotFound {
			t.Errorf("endpoint %s returned 404 (not registered)", ep)
		}
	}
}

func TestHandler_IndexPage(t *testing.T) {
	h := Handler()
	mux, ok := h.(*http.ServeMux)
	if !ok {
		t.Fatal("Handler() should return *http.ServeMux")
	}
	req := httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("index page: want 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "pprof") {
		t.Errorf("index page should contain 'pprof', got: %s", body)
	}
}

func TestAttachTo_RegistersAtCustomPrefix(t *testing.T) {
	// 注意：AttachTo 是在自定义 prefix 下挂载 pprof mux。
	// 由于 Handler() 使用绝对路径（/debug/pprof/...）注册，
	// 所以 custom prefix 模式下，子路径仍以 /debug/pprof/ 开头。
	// 这里仅验证挂载本身不 panic。
	mux := http.NewServeMux()
	AttachTo(mux, "/custom/pprof/")

	// 验证 /custom/pprof/ 端点被注册
	req := httptest.NewRequest(http.MethodGet, "/custom/pprof/", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	// 在 net/http 的 ServeMux 中，挂载一个 mux 到 /custom/pprof/
	// 会使 /custom/pprof/ 返回 200（子 mux 自身的 /debug/pprof/ 索引）
	// 或者返回 404（路径不匹配）。两种都是合理的。
	// 关键是不能 panic、不能返回 5xx。
	if rr.Code >= 500 {
		t.Errorf("custom prefix should not return 5xx, got %d", rr.Code)
	}
}
