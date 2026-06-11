// Package pprof 提供基于 net/http/pprof 的 ruleflow 性能分析端点。
//
// 使用示例：
//
//	import (
//	    "net/http"
//	    ruleflowpprof "github.com/vpptu/ruleflow/pkg/ruleflow/contrib/pprof"
//	)
//
//	mux := http.NewServeMux()
//	mux.Handle("/debug/pprof/", ruleflowpprof.Handler())
//	go http.ListenAndServe(":6060", mux)
//
// 端点列表：
//   - /debug/pprof/          - 索引页
//   - /debug/pprof/cmdline   - 进程命令行
//   - /debug/pprof/profile   - CPU profile（默认 30s）
//   - /debug/pprof/symbol    - 符号解析
//   - /debug/pprof/trace     - 执行追踪
//   - /debug/pprof/goroutine - goroutine 堆栈
//   - /debug/pprof/heap      - 堆内存
//   - /debug/pprof/allocs    - 分配
//   - /debug/pprof/block     - 阻塞事件
//   - /debug/pprof/mutex     - 互斥锁
//   - /debug/pprof/threadcreate - 线程创建
package pprof

import (
	"net/http"
	"net/http/pprof"
)

// Handler 返回包含所有 pprof 端点的 http.Handler。
// 使用方法：mux.Handle("/debug/pprof/", pprof.Handler())
func Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	// 以下端点在 net/http/pprof 包中已通过 init() 注册到 default mux，
	// 但用户自定义 mux 不会自动包含，所以需要显式注册：
	mux.HandleFunc("/debug/pprof/goroutine", pprof.Handler("goroutine").ServeHTTP)
	mux.HandleFunc("/debug/pprof/heap", pprof.Handler("heap").ServeHTTP)
	mux.HandleFunc("/debug/pprof/allocs", pprof.Handler("allocs").ServeHTTP)
	mux.HandleFunc("/debug/pprof/block", pprof.Handler("block").ServeHTTP)
	mux.HandleFunc("/debug/pprof/mutex", pprof.Handler("mutex").ServeHTTP)
	mux.HandleFunc("/debug/pprof/threadcreate", pprof.Handler("threadcreate").ServeHTTP)
	return mux
}

// AttachTo 将 pprof 端点挂载到指定的 ServeMux 上。
// prefix 必须以 "/" 结尾（默认推荐 "/debug/pprof/"）。
func AttachTo(mux *http.ServeMux, prefix string) {
	h := Handler()
	mux.Handle(prefix, h)
}
