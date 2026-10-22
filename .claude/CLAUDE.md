| Resolver | @internal/source/resolver/resolver.go |
| 表頭版面 | @internal/source/layout.go |
| 範例資料 | @internal/sampledata/sampledata.go |# formgen - Schema-First 代碼生成工具

> 統一的 YAML Schema 作為 Source of Truth，支援多格式輸出（SQL, ORM, Proto, YAML）

## 快速命令

```bash
# Build
go build -o formgen .

# Test（單一測試更快）
go test -run TestName ./internal/path
go test ./...

# Lint
golangci-lint run ./...

# 覆蓋率
go test ./... -coverprofile=coverage.out && go tool cover -html=coverage.out

# 範例資料與 golden 檔
go generate ./internal/sampledata            # 重新產生 testdata/Demo_*.xlsx
go test ./cmd -run TestGolden -update        # 重新產生 testdata/sample.*
```

**Pipeline**: `Excel/CSV/YAML → Loader → Parser → Resolver → Generator → SQL/ORM/Proto/YAML`

---

## 必須遵循的規則（MUST）

### 1. Options 模式
當方法設計有**多個可選配置**時使用 Options 模式。

```go
// 有多個可選配置時才使用
gen := sqlgen.New(
    sqlgen.WithCustomMappings(map[string]string{"userId": "VARCHAR(64)"}),
)
```

### 2. 檔案組織
```
internal/generator/
├── generator.go             # 統一的 Generator 介面
└── <format>/
    ├── <format>.go          # 模組入口：New() 返回 gen.Generator
    ├── generator.go         # 核心實現：generator struct、newGenerator()、Generate()
    ├── options.go           # Option 類型、With* 函數
    └── <format>_test.go     # 測試
```
**原則**：統一介面在 `generator/generator.go`，各子包的 `New()` 返回 `gen.Generator`。

### 3. 錯誤處理
- 使用 `%w` 保留錯誤鏈：`fmt.Errorf("failed to parse: %w", err)`
- 初始化不可恢復：`panic`
- 運行時可恢復：`return error`

### 4. 測試驅動開發（TDD）
**每次實作功能或修復 Bug 時必須遵循**：

1. **生成可測試程式碼**：依賴注入、單一職責、純函數優先、避免全域狀態
2. **同步生成測試**：功能程式碼與測試程式碼一起產出，使用表驅動模式
3. **執行測試驗證**：生成後必須執行 `go test`，失敗就修復並重試直到通過

### 5. 介面設計
- 小介面原則：每個介面只定義必要方法
- 接受介面，返回具體類型：`func New() *generator`（不是 `Generator`）

---

## 關鍵檔案參考

| 用途 | 檔案 |
|------|------|
| Schema 定義 | @internal/schema/scheme.go |
| Options 範例 | @internal/generator/sql/options.go |
| 測試範例 | @internal/generator/sql/sql_test.go |
| Resolver | @internal/source/resolver/resolver.go |
