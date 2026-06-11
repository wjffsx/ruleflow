package config

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vpptu/ruleflow/pkg/ruleflow/builtin"
	"github.com/vpptu/ruleflow/pkg/ruleflow/nodes"
)

// stubLoader 记录 Apply 调用次数
type stubLoader struct {
	calls atomic.Int32
	cfg   *ChainConfig
}

func (s *stubLoader) Load(path string) (*ChainConfig, error) {
	s.calls.Add(1)
	return s.cfg, nil
}

func (s *stubLoader) Apply(_ context.Context, _ *ChainConfig) error {
	s.calls.Add(1)
	return nil
}

func newStubLoader() *stubLoader {
	return &stubLoader{
		cfg: &ChainConfig{
			Chain: ChainMeta{ID: "stub", Name: "stub", Version: 1, Status: "deployed", Root: true},
			Rules: []RuleConfig{{
				ID: "r1", Priority: 1, Enabled: true,
			}},
		},
	}
}

// TestFileWatcher_LoadOnStartup 验证文件存在即立即加载
func TestFileWatcher_LoadOnStartup(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "chain.yaml")
	if err := os.WriteFile(path, []byte("placeholder"), 0644); err != nil {
		t.Fatal(err)
	}

	loader := newStubLoader()
	fw, err := NewFileWatcher(path, loader)
	if err != nil {
		t.Fatal(err)
	}
	fw.WithDebounce(50 * time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := fw.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer fw.Stop()

	// 等待 bootstrap 加载完成
	time.Sleep(300 * time.Millisecond)

	// 验证 Apply 至少被调用一次
	if loader.calls.Load() < 1 {
		t.Errorf("expected Apply to be called >= 1, got %d", loader.calls.Load())
	}
}

// TestFileWatcher_TriggersReload 验证文件变更后调用 Load + Apply
func TestFileWatcher_TriggersReload(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "chain.yaml")
	if err := os.WriteFile(path, []byte("v1"), 0644); err != nil {
		t.Fatal(err)
	}

	loader := newStubLoader()
	fw, err := NewFileWatcher(path, loader)
	if err != nil {
		t.Fatal(err)
	}
	fw.WithDebounce(50 * time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := fw.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer fw.Stop()
	time.Sleep(200 * time.Millisecond)

	callsBefore := loader.calls.Load()

	// 触发文件写入事件
	if err := os.WriteFile(path, []byte("v2"), 0644); err != nil {
		t.Fatal(err)
	}
	time.Sleep(400 * time.Millisecond)

	if loader.calls.Load() <= callsBefore {
		t.Errorf("expected Apply to be called after file change; before=%d, after=%d",
			callsBefore, loader.calls.Load())
	}
}

// TestDirWatcher_LoadAllOnStartup 验证目录启动时加载所有配置
func TestDirWatcher_LoadAllOnStartup(t *testing.T) {
	tmp := t.TempDir()
	for _, name := range []string{"a.yaml", "b.yml", "c.json"} {
		if err := os.WriteFile(filepath.Join(tmp, name), []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	loader := newStubLoader()
	dw, err := NewDirWatcher(tmp, loader)
	if err != nil {
		t.Fatal(err)
	}
	dw.WithDebounce(50 * time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := dw.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer dw.Stop()
	time.Sleep(400 * time.Millisecond)

	if loader.calls.Load() < 3 {
		t.Errorf("expected at least 3 Apply calls (one per file), got %d", loader.calls.Load())
	}
}

// TestDirWatcher_IgnoreNonConfigFiles 验证非 yaml/json 文件被忽略
func TestDirWatcher_IgnoreNonConfigFiles(t *testing.T) {
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "README.md"), []byte("# hi"), 0644); err != nil {
		t.Fatal(err)
	}

	loader := newStubLoader()
	dw, err := NewDirWatcher(tmp, loader)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := dw.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer dw.Stop()

	time.Sleep(200 * time.Millisecond)
	if loader.calls.Load() != 0 {
		t.Errorf("non-config files should be ignored, got %d calls", loader.calls.Load())
	}
}

// TestFileWatcher_NonExistentPath 验证路径不存在时返回错误
func TestFileWatcher_NonExistentPath(t *testing.T) {
	_, err := NewFileWatcher("/no/such/path_xyz.yaml", &DefaultLoader{Registry: func() *nodes.Registry { r := nodes.NewEmptyRegistry(); r.RegisterPackage(builtin.Builtin); return r }()})
	if err == nil {
		t.Error("expected error for non-existent path")
	}
}

// TestFileWatcher_NilArgs 验证 nil 参数错误
func TestFileWatcher_NilArgs(t *testing.T) {
	loader := &DefaultLoader{Registry: func() *nodes.Registry { r := nodes.NewEmptyRegistry(); r.RegisterPackage(builtin.Builtin); return r }()}

	if _, err := NewFileWatcher("", loader); err == nil {
		t.Error("empty path should fail")
	}
	if _, err := NewFileWatcher("a.yaml", nil); err == nil {
		t.Error("nil loader should fail")
	}
}

// TestFileWatcher_DoubleStart 验证重复 Start 报错
func TestFileWatcher_DoubleStart(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "c.yaml")
	_ = os.WriteFile(path, []byte("x"), 0644)

	fw, _ := NewFileWatcher(path, newStubLoader())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := fw.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer fw.Stop()
	if err := fw.Start(ctx); err == nil {
		t.Error("double Start should return error")
	}
}

// TestDirWatcher_NotADirectory 验证传入文件而非目录时报错
func TestDirWatcher_NotADirectory(t *testing.T) {
	tmp := t.TempDir()
	file := filepath.Join(tmp, "a.yaml")
	_ = os.WriteFile(file, []byte("x"), 0644)
	_, err := NewDirWatcher(file, newStubLoader())
	if err == nil {
		t.Error("expected error for non-directory path")
	}
}
