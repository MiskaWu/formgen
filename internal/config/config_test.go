package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/miskawu/formgen/internal/source"
)

func TestInit_WithoutConfigFile(t *testing.T) {
	// 重置全域配置
	globalConfig = nil

	// 不提供配置檔案，應該返回空映射
	err := Init("")
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	cfg := Get()
	if cfg == nil {
		t.Fatal("Get() returned nil")
	}

	// 沒有配置檔時，Custom 應該是空的（但不是 nil）
	if cfg.Mappings.Custom == nil {
		t.Error("expected Custom map to be initialized, got nil")
	}
}

func TestInit_WithConfigFile(t *testing.T) {
	// 重置全域配置
	globalConfig = nil

	// 建立臨時配置檔
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "formgen.yaml")
	// 注意：Viper 會將 key 轉為小寫
	cfgContent := `
mappings:
  custom:
    entry: int32
    asset: string
    customtype: int64
`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// 使用配置檔初始化
	err := Init(cfgPath)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	cfg := Get()

	// 驗證配置檔的值
	// 注意：Viper 會將 key 轉為小寫
	tests := []struct {
		key  string
		want string
	}{
		{"entry", "int32"},
		{"asset", "string"},
		{"customtype", "int64"},
	}

	for _, tt := range tests {
		if got := cfg.Mappings.Custom[tt.key]; got != tt.want {
			t.Errorf("Custom[%s] = %s, want %s", tt.key, got, tt.want)
		}
	}
}

func TestInit_InvalidConfigFile(t *testing.T) {
	// 重置全域配置
	globalConfig = nil

	// 建立無效的配置檔
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "formgen.yaml")
	cfgContent := `invalid yaml: [}`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// 應該返回錯誤
	err := Init(cfgPath)
	if err == nil {
		t.Error("Init() expected error for invalid config file, got nil")
	}
}

func TestGetCustomMappings(t *testing.T) {
	// 重置全域配置
	globalConfig = nil

	// 建立臨時配置檔
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "formgen.yaml")
	// 注意：Viper 會將 key 轉為小寫
	cfgContent := `
mappings:
  custom:
    testkey: testvalue
`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	err := Init(cfgPath)
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	mappings := GetCustomMappings()
	if mappings == nil {
		t.Fatal("GetCustomMappings() returned nil")
	}

	if mappings["testkey"] != "testvalue" {
		t.Errorf("expected testkey=testvalue, got %s", mappings["testkey"])
	}
}

func TestGet_AutoInitialize(t *testing.T) {
	// 重置全域配置
	globalConfig = nil

	// 直接呼叫 Get() 應該自動初始化
	cfg := Get()
	if cfg == nil {
		t.Fatal("Get() returned nil")
	}

	// Custom 應該被初始化為空 map
	if cfg.Mappings.Custom == nil {
		t.Error("expected Custom map to be initialized, got nil")
	}
}

func TestGetCustomMappings_Empty(t *testing.T) {
	// 重置全域配置
	globalConfig = nil

	err := Init("")
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	mappings := GetCustomMappings()
	if mappings == nil {
		t.Fatal("GetCustomMappings() returned nil")
	}

	// 沒有配置檔時應該是空 map
	if len(mappings) != 0 {
		t.Errorf("expected empty mappings, got %v", mappings)
	}
}

func TestLayoutConfig_ToLayout(t *testing.T) {
	tests := []struct {
		name    string
		cfg     LayoutConfig
		want    source.Layout
		wantErr bool
	}{
		{
			name: "未設定時使用預設版面",
			cfg:  LayoutConfig{},
			want: source.DefaultLayout(),
		},
		{
			name: "Excel 列號轉為 0-based 索引",
			cfg:  LayoutConfig{NameRow: 1, TypeRow: 2, MetaRow: 3, OutputRow: 4, DataRow: 8, SkipMarker: "ignore"},
			want: source.Layout{NameRow: 0, TypeRow: 1, MetaRow: 2, OutputRow: 3, DataRow: 7, SkipMarker: "ignore"},
		},
		{
			name: "只覆寫部分欄位",
			cfg:  LayoutConfig{DataRow: 10},
			want: source.Layout{NameRow: 1, TypeRow: 2, MetaRow: 3, OutputRow: 4, DataRow: 9, SkipMarker: "skip"},
		},
		{
			name:    "資料列不可在表頭之前",
			cfg:     LayoutConfig{DataRow: 3},
			wantErr: true,
		},
		{
			name:    "表頭列不可重複",
			cfg:     LayoutConfig{NameRow: 3, TypeRow: 3},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.cfg.ToLayout()
			if (err != nil) != tt.wantErr {
				t.Fatalf("ToLayout() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got != tt.want {
				t.Errorf("ToLayout() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestInit_WithLayoutConfig(t *testing.T) {
	globalConfig = nil

	cfgPath := filepath.Join(t.TempDir(), "formgen.yaml")
	cfgContent := `
layout:
  nameRow: 1
  typeRow: 2
  metaRow: 3
  outputRow: 4
  dataRow: 5
  skipMarker: "-"
`
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}
	if err := Init(cfgPath); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	got, err := GetLayout()
	if err != nil {
		t.Fatalf("GetLayout() error = %v", err)
	}
	want := source.Layout{NameRow: 0, TypeRow: 1, MetaRow: 2, OutputRow: 3, DataRow: 4, SkipMarker: "-"}
	if got != want {
		t.Errorf("GetLayout() = %+v, want %+v", got, want)
	}
}

func TestDefaultConfig_IsLoadable(t *testing.T) {
	globalConfig = nil

	cfgPath := filepath.Join(t.TempDir(), "formgen.yaml")
	if err := os.WriteFile(cfgPath, []byte(DefaultConfig()), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}
	if err := Init(cfgPath); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	mappings := GetCustomMappings()
	for _, key := range []string{"int", "entry", "asset", "tag", "shortint", "tinyint"} {
		if _, ok := mappings[key]; !ok {
			t.Errorf("default config should map %q", key)
		}
	}
	if _, err := GetLayout(); err != nil {
		t.Errorf("default config layout should be valid: %v", err)
	}
}

// TestInit_IgnoresBinaryNamedFormgen 確認目前目錄有名為 formgen 的執行檔時不會被當成設定檔
func TestInit_IgnoresBinaryNamedFormgen(t *testing.T) {
	globalConfig = nil

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "formgen"), []byte("\x7fELF\x02\x01\x01\x00binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)

	if err := Init(""); err != nil {
		t.Fatalf("Init() should ignore the formgen binary, got error: %v", err)
	}

	// 有 formgen.yaml 時照常讀取
	globalConfig = nil
	if err := os.WriteFile(filepath.Join(dir, "formgen.yaml"), []byte("mappings:\n  custom:\n    entry: int32\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Init(""); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if got := GetCustomMappings()["entry"]; got != "int32" {
		t.Errorf("entry mapping = %q, want int32", got)
	}
}
