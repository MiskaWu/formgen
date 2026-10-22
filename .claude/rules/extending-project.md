# 擴展專案指南

## 添加新的 Generator

以 GraphQL Schema 生成為例：

### Step 1: 創建包結構

```
internal/generator/graphql/
├── graphql.go       # 模組入口：New() 返回 gen.Generator、embed 資源
├── generator.go     # 核心實現：generator struct、newGenerator()、Generate()
├── options.go       # Option 類型
├── mapper.go        # 類型映射
└── graphql_test.go  # 測試
```

### Step 2: 定義類型映射

```go
// internal/generator/graphql/mapper.go
func defaultMappings() map[string]string {
    return map[string]string{
        "int32":     "Int",
        "int64":     "Int",
        "float":     "Float",
        "double":    "Float",
        "string":    "String",
        "bool":      "Boolean",
        "timestamp": "DateTime",
    }
}
```

### Step 3: 實現 Options

```go
// internal/generator/graphql/options.go
type options struct {
    typeMapper    mapper.TypeMapper
    generateInput bool
}

type Option func(*options)

func defaultOptions() *options {
    return &options{
        typeMapper:    mapper.NewTypeMapper(defaultMappings()),
        generateInput: false,
    }
}

func WithInputTypes() Option {
    return func(o *options) {
        o.generateInput = true
    }
}
```

### Step 4: 實現 Generator

```go
// internal/generator/graphql/graphql.go - 模組入口
import gen "github.com/miskawu/formgen/internal/generator"

// New 創建 graphql 生成器
func New(opts ...Option) gen.Generator {
    return newGenerator(opts...)
}

// internal/generator/graphql/generator.go - 核心實現
type generator struct {
    typeMapper    mapper.TypeMapper
    generateInput bool
}

func newGenerator(opts ...Option) *generator {
    o := defaultOptions()
    for _, opt := range opts {
        opt(o)
    }
    return &generator{
        typeMapper:    o.typeMapper,
        generateInput: o.generateInput,
    }
}

func (g *generator) Generate(entities []schema.Entity) ([]byte, error) {
    // 實現生成邏輯...
}

func (g *generator) buildSchema(entity schema.Entity) string {
    // 私有輔助函數...
}
```

### Step 5: 編寫測試

```go
// internal/generator/graphql/graphql_test.go
func TestGraphQLGenerator_Generate(t *testing.T) {
    tests := []struct {
        name     string
        entities []schema.Entity
        options  []Option
        want     []string
        wantErr  bool
    }{
        {
            name: "基本 type 生成",
            // ...
        },
    }
    // ...
}
```

---

## 添加新的 Parser/Loader

### Step 1: 創建包結構

```
internal/source/json/
├── json.go       # 模組入口：NewLoader() 建構函數
├── loader.go     # 核心實現：loader struct、Load() 及私有方法
└── json_test.go  # 測試
```

### Step 2: 實現 Loader

```go
// internal/source/json/json.go - 模組入口
func NewLoader() *loader {
    return newLoader()
}

// internal/source/json/loader.go - 核心實現
type loader struct{}

func newLoader() *loader {
    return &loader{}
}

func (l *loader) Load(data []byte) ([]source.Sheet, error) {
    // 實現載入邏輯...
}

func (l *loader) parseJSON(data []byte) (map[string]any, error) {
    // 私有輔助函數...
}
```

---

## Pipeline 模式

所有轉換流程都遵循：**Load → Parse → Generate**

```go
func convertExcelToSQL(excelPath, sqlPath string) error {
    // Step 1: Load - 讀取原始資料
    loader := excel.NewLoader()
    sheets, err := loader.Load(excelPath)
    if err != nil {
        return fmt.Errorf("failed to load excel: %w", err)
    }

    // Step 2: Parse - 轉換為中介 Schema
    parser := source.NewParser()
    entities, err := parser.Parse(sheets)
    if err != nil {
        return fmt.Errorf("failed to parse: %w", err)
    }

    // Step 3: Generate - 生成目標格式
    gen := sqlgen.New()
    output, err := gen.Generate(entities)
    if err != nil {
        return fmt.Errorf("failed to generate sql: %w", err)
    }

    return os.WriteFile(sqlPath, output, 0644)
}
```

---

## 資源嵌入

使用 Go 1.16+ 的 `embed` 指令：

```go
import _ "embed"

//go:embed templates/sql.tmpl
var sqlTemplate string
```

優點：編譯到單一二進位檔案，部署簡單。
