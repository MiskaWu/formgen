# 代碼組織指南

## 包結構

```
internal/
├── source/             # 資料來源層：從各種格式載入表格資料
│   ├── source.go       # source.Sheet 通用結構 + NewParser()
│   ├── layout.go       # 表頭版面（各列位置、skip 標記）
│   ├── parser.go       # source.Parser 解析欄位約定
│   ├── excel/          # Excel 載入器
│   │   ├── excel.go    # 對外公開的 API（NewLoader()、Load() 等）
│   │   └── loader.go   # 內部實現細節
│   ├── resolver/       # 跨表引用解析器
│   └── csv/            # CSV 載入器（未來擴展）
├── generator/          # 生成層：對外暴露統一 Generator 介面
│   ├── orm/            # ORM 生成器
│   ├── sql/            # SQL 生成器
│   ├── yaml/           # YAML Schema 生成器
│   ├── proto/          # Proto 生成器
│   └── wrapper/        # envelope 包裝函數生成器
├── mapper/             # 類型映射：可共用的類型轉換邏輯
├── schema/             # 中介結構：格式無關的資料定義
│   └── yaml/           # YAML 格式載入器
├── importer/           # 工具層：數據導入等高層操作
├── config/             # 配置管理：使用 Viper
├── migration/          # 資料庫遷移：使用 golang-migrate
├── models/             # 快照與版本表的 GORM model（db import --snapshot 使用）
├── sampledata/         # 範例與測試用的合成 Excel 表（go generate 產生 testdata/）
└── serializer/         # Protocol Buffer 序列化
```

## 檔案組織慣例（必須遵循）

Generator 介面統一定義在父包，各子包實現具體生成器：

```
internal/generator/
├── generator.go             # 統一的 Generator 介面定義
└── <format>/
    ├── <format>.go          # 模組入口：New() 返回 gen.Generator、embed 資源
    ├── generator.go         # 核心實現：generator struct、newGenerator()、Generate()
    ├── options.go           # Option 類型、defaultOptions()、With* 函數
    ├── mapper.go            # 類型映射（如有需要）
    ├── keyword.go           # 保留字處理（如有需要）
    └── <format>_test.go     # 測試
```

**核心原則**：
- `generator/generator.go` 定義統一的 `Generator` 介面，所有生成器共用
- `<format>.go` 是模組的**入口點**
  - `New()` 建構函數：返回 `gen.Generator`（使用別名避免與 struct 衝突）
  - embed 資源：嵌入模板等靜態資源
- `generator.go` 放 struct 定義及其所有方法
  - `newGenerator()` 包含實際的初始化邏輯（options 處理、模板解析等）
- 其他檔案（options.go、mapper.go 等）放**延伸實現**，支援主檔案的功能

## 命名慣例

| 類型 | 命名規則 | 範例 |
|------|---------|------|
| 包內實現 | 小寫，未導出 | `type generator struct` |
| 公開介面 | 大寫，導出 | `type Generator interface` |
| 構造函數 | `New` 前綴 | `func New(...) *generator` |
| 選項函數 | `With` 前綴 | `func WithCustomMappings(...) Option` |

## 介面設計

### 小介面原則

```go
// GOOD: 單一職責
type Generator interface {
    Generate(entities []schema.Entity) ([]byte, error)
}

// BAD: 職責過多
type Generator interface {
    Generate(entities []schema.Entity) ([]byte, error)
    Validate() error
    Configure(opts map[string]any) error
}
```

### 接受介面，返回具體類型

```go
// GOOD
func NewTypeMapper(mappings map[string]string) *defaultTypeMapper

// BAD: 返回介面會限制未來擴展
func NewTypeMapper(mappings map[string]string) TypeMapper
```

## 導入順序

```go
import (
    // 標準庫
    "bytes"
    "fmt"
    "os"

    // 第三方庫
    "github.com/spf13/cobra"
    "github.com/xuri/excelize/v2"

    // 本專案
    "github.com/miskawu/formgen/internal/generator"
    "github.com/miskawu/formgen/internal/schema"
)
```

## 註解規範

導出的類型和函數**必須**有文檔註解：

```go
// Generator 定義代碼生成器的介面。
// 所有生成器（SQL、ORM 等）都必須實現此介面。
type Generator interface {
    // Generate 將 Entity 列表轉換為目標格式的字節表示。
    Generate(entities []schema.Entity) ([]byte, error)
}
```
