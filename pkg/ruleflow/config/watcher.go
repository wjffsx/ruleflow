// Package config 提供基于 fsnotify 的规则链配置文件热重载。
//
// 设计要点：
//   - 单文件 / 目录两种监听模式
//   - 写事件触发后调用 Loader 重新解析配置
//   - 通过回调函数执行 LoadChain / UnloadChain（解耦 engine 依赖）
//   - ctx 控制后台 goroutine 生命周期
//   - 防抖（debounce）避免编辑器多次写入触发的抖动
package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/wjffsx/ruleflow/pkg/ruleflow/core"
	"github.com/wjffsx/ruleflow/pkg/ruleflow/core/contract"
	"github.com/wjffsx/ruleflow/pkg/ruleflow/nodes"
)

// ─────────────────────────────────────────────
//  Loader — 配置加载与 Reload 钩子
// ─────────────────────────────────────────────

// Loader 负责从文件加载配置并应用。
// 应用层可实现自定义 Loader，例如从远端配置中心拉取。
type Loader interface {
	// Load 从 path 加载 ChainConfig
	Load(path string) (*ChainConfig, error)
	// Apply 把配置应用到引擎（通过回调）
	Apply(ctx context.Context, cfg *ChainConfig) error
}

// ChainReloader 规则链重载回调函数类型。
// 由应用层注入，通常为 engine.LoadChain 的包装。
type ChainReloader func(ctx context.Context, chain *core.RuleChain) error

// DefaultLoader 默认基于 YAML 文件 + registry 的 Loader
type DefaultLoader struct {
	// Registry 用于解析条件/动作类型
	Registry *nodes.Registry
	// ReloadTimeout 每次 Apply 的最大耗时（默认 5s）
	ReloadTimeout time.Duration
	// OnLoad 规则链重载回调（由应用层注入，如 engine.LoadChain）
	OnLoad ChainReloader
}

// Load 实现 Loader
func (l *DefaultLoader) Load(path string) (*ChainConfig, error) {
	return LoadFromFile(path)
}

// Apply 实现 Loader
func (l *DefaultLoader) Apply(ctx context.Context, cfg *ChainConfig) error {
	chain, err := Parse(cfg, l.Registry)
	if err != nil {
		return fmt.Errorf("parse config: %w", err)
	}
	if l.ReloadTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, l.ReloadTimeout)
		defer cancel()
	}
	if l.OnLoad != nil {
		if err := l.OnLoad(ctx, chain); err != nil {
			return fmt.Errorf("load chain: %w", err)
		}
	}
	return nil
}

// ─────────────────────────────────────────────
//  FileWatcher — 单文件监听
// ─────────────────────────────────────────────

// FileWatcher 监听单个 YAML/JSON 配置文件，变更后热加载。
type FileWatcher struct {
	path     string
	loader   Loader
	debounce time.Duration
	logger   contract.Logger

	mu       sync.Mutex
	watcher  *fsnotify.Watcher
	stopCh   chan struct{}
	stopOnce sync.Once
}

// NewFileWatcher 构造文件监听器
//   - path: 配置文件绝对路径
//   - loader: 配置加载器
func NewFileWatcher(path string, loader Loader) (*FileWatcher, error) {
	if path == "" {
		return nil, errors.New("path is empty")
	}
	if loader == nil {
		return nil, errors.New("loader is nil")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve path: %w", err)
	}
	if _, err := os.Stat(abs); err != nil {
		return nil, fmt.Errorf("stat path: %w", err)
	}
	return &FileWatcher{
		path:     abs,
		loader:   loader,
		debounce: 200 * time.Millisecond,
		logger:   contract.NoopLogger(),
	}, nil
}

// WithDebounce 配置防抖窗口（默认 200ms）
func (w *FileWatcher) WithDebounce(d time.Duration) *FileWatcher {
	w.mu.Lock()
	defer w.mu.Unlock()
	if d > 0 {
		w.debounce = d
	}
	return w
}

// WithLogger 配置日志器
func (w *FileWatcher) WithLogger(l contract.Logger) *FileWatcher {
	w.mu.Lock()
	defer w.mu.Unlock()
	if l != nil {
		w.logger = l
	}
	return w
}

// Start 启动监听 goroutine。ctx 取消时优雅退出。
func (w *FileWatcher) Start(ctx context.Context) error {
	w.mu.Lock()
	if w.watcher != nil {
		w.mu.Unlock()
		return errors.New("watcher already started")
	}
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		w.mu.Unlock()
		return fmt.Errorf("create fsnotify watcher: %w", err)
	}
	if err := fw.Add(w.path); err != nil {
		_ = fw.Close()
		w.mu.Unlock()
		return fmt.Errorf("watch path: %w", err)
	}
	w.watcher = fw
	w.stopCh = make(chan struct{})
	stopCh := w.stopCh
	w.mu.Unlock()

	// 启动时先加载一次（同步执行，保证 Start 返回前已就绪）
	w.reload(ctx)

	go w.run(ctx, fw, stopCh)
	return nil
}

// Stop 停止监听
func (w *FileWatcher) Stop() error {
	w.stopOnce.Do(func() {
		w.mu.Lock()
		defer w.mu.Unlock()
		if w.stopCh != nil {
			close(w.stopCh)
		}
		if w.watcher != nil {
			_ = w.watcher.Close()
		}
	})
	return nil
}

// run 事件循环
func (w *FileWatcher) run(ctx context.Context, fw *fsnotify.Watcher, stopCh chan struct{}) {
	var debounceTimer *time.Timer
	var debounceCh <-chan time.Time

	for {
		select {
		case <-ctx.Done():
			return
		case <-stopCh:
			return
		case event, ok := <-fw.Events:
			if !ok {
				return
			}
			if !shouldHandle(event) {
				continue
			}
			w.logger.Info("file watcher: change detected",
				"path", w.path, "op", event.Op.String())

			// 防抖：200ms 内多次写入合并为一次
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			debounceCh = time.After(w.debounce)
			go func() {
				<-debounceCh
				w.reload(ctx)
			}()

		case err, ok := <-fw.Errors:
			if !ok {
				return
			}
			w.logger.Error("file watcher error", "err", err)
			_ = debounceTimer // keep linter happy
		}
	}
}

// reload 触发一次重新加载
func (w *FileWatcher) reload(ctx context.Context) {
	cfg, err := w.loader.Load(w.path)
	if err != nil {
		w.logger.Error("file watcher: load failed", "path", w.path, "err", err)
		return
	}
	if err := w.loader.Apply(ctx, cfg); err != nil {
		w.logger.Error("file watcher: apply failed",
			"path", w.path, "chain_id", cfg.Chain.ID, "err", err)
		return
	}
	w.logger.Info("file watcher: reloaded", "path", w.path, "chain_id", cfg.Chain.ID)
}

// shouldHandle 判定事件是否需要处理
func shouldHandle(event fsnotify.Event) bool {
	// 写入、创建、改名都视为可能的内容变更
	if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) || event.Has(fsnotify.Rename) {
		// 排除临时文件
		base := filepath.Base(event.Name)
		if strings.HasPrefix(base, ".") || strings.HasSuffix(base, "~") || strings.HasSuffix(base, ".swp") {
			return false
		}
		return true
	}
	return false
}

// ─────────────────────────────────────────────
//  DirWatcher — 目录监听
// ─────────────────────────────────────────────

// DirWatcher 监听整个目录下的 *.yaml/*.yml/*.json 文件，
// 每个文件对应一个规则链，文件名（去扩展名）即 chainID 的一部分。
type DirWatcher struct {
	dir      string
	loader   Loader
	debounce time.Duration
	logger   contract.Logger

	mu       sync.Mutex
	watcher  *fsnotify.Watcher
	stopCh   chan struct{}
	stopOnce sync.Once
}

// NewDirWatcher 构造目录监听器
func NewDirWatcher(dir string, loader Loader) (*DirWatcher, error) {
	if dir == "" {
		return nil, errors.New("dir is empty")
	}
	if loader == nil {
		return nil, errors.New("loader is nil")
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("resolve dir: %w", err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("stat dir: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", abs)
	}
	return &DirWatcher{
		dir:      abs,
		loader:   loader,
		debounce: 300 * time.Millisecond,
		logger:   contract.NoopLogger(),
	}, nil
}

// WithDebounce 配置防抖窗口
func (w *DirWatcher) WithDebounce(d time.Duration) *DirWatcher {
	w.mu.Lock()
	defer w.mu.Unlock()
	if d > 0 {
		w.debounce = d
	}
	return w
}

// WithLogger 配置日志器
func (w *DirWatcher) WithLogger(l contract.Logger) *DirWatcher {
	w.mu.Lock()
	defer w.mu.Unlock()
	if l != nil {
		w.logger = l
	}
	return w
}

// Start 启动监听
func (w *DirWatcher) Start(ctx context.Context) error {
	w.mu.Lock()
	if w.watcher != nil {
		w.mu.Unlock()
		return errors.New("watcher already started")
	}
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		w.mu.Unlock()
		return fmt.Errorf("create fsnotify watcher: %w", err)
	}
	if err := fw.Add(w.dir); err != nil {
		_ = fw.Close()
		w.mu.Unlock()
		return fmt.Errorf("watch dir: %w", err)
	}
	w.watcher = fw
	w.stopCh = make(chan struct{})
	stopCh := w.stopCh
	w.mu.Unlock()

	// 启动时先加载目录中所有现有文件
	w.bootstrap(ctx)

	go w.run(ctx, fw, stopCh)
	return nil
}

// bootstrap 启动时加载所有已存在的文件
func (w *DirWatcher) bootstrap(ctx context.Context) {
	entries, err := os.ReadDir(w.dir)
	if err != nil {
		w.logger.Error("dir watcher: read dir failed", "dir", w.dir, "err", err)
		return
	}
	for _, ent := range entries {
		if ent.IsDir() {
			continue
		}
		if !isConfigFile(ent.Name()) {
			continue
		}
		path := filepath.Join(w.dir, ent.Name())
		w.loadOne(ctx, path)
	}
}

// isConfigFile 判断文件是否为支持的配置格式
func isConfigFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".yaml" || ext == ".yml" || ext == ".json"
}

// Stop 停止监听
func (w *DirWatcher) Stop() error {
	w.stopOnce.Do(func() {
		w.mu.Lock()
		defer w.mu.Unlock()
		if w.stopCh != nil {
			close(w.stopCh)
		}
		if w.watcher != nil {
			_ = w.watcher.Close()
		}
	})
	return nil
}

// run 事件循环
func (w *DirWatcher) run(ctx context.Context, fw *fsnotify.Watcher, stopCh chan struct{}) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-stopCh:
			return
		case event, ok := <-fw.Events:
			if !ok {
				return
			}
			if !isConfigFile(filepath.Base(event.Name)) {
				continue
			}
			if !shouldHandle(event) {
				continue
			}
			w.logger.Info("dir watcher: change detected",
				"path", event.Name, "op", event.Op.String())
			// 防抖
			time.Sleep(w.debounce)
			w.loadOne(ctx, event.Name)

		case err, ok := <-fw.Errors:
			if !ok {
				return
			}
			w.logger.Error("dir watcher error", "err", err)
		}
	}
}

// loadOne 处理单个文件
func (w *DirWatcher) loadOne(ctx context.Context, path string) {
	cfg, err := w.loader.Load(path)
	if err != nil {
		w.logger.Error("dir watcher: load failed", "path", path, "err", err)
		return
	}
	if err := w.loader.Apply(ctx, cfg); err != nil {
		w.logger.Error("dir watcher: apply failed",
			"path", path, "chain_id", cfg.Chain.ID, "err", err)
		return
	}
	w.logger.Info("dir watcher: loaded", "path", path, "chain_id", cfg.Chain.ID)
}
