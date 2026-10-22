# formgen

[![Go Version](https://img.shields.io/badge/Go-1.25-blue.svg)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

> **formgen** is a schema-first code generation toolchain for spreadsheet-driven data.
> It reads Excel sheets with a small header convention, resolves cross-sheet references,
> writes a normalized YAML schema, and generates SQL DDL, GORM models, `.proto` files and
> Go envelope helpers from it. It can also import the sheets into a database and store each
> table as a gzip-compressed protobuf snapshot, compiled at runtime with `protocompile`
> and populated through `dynamicpb` — no generated Go code required.

formgen 是一套 Schema-First 的代碼生成工具鏈：以 Excel 表格作為資料定義的來源，先轉成一份保留細緻型別資訊的 YAML Schema，再由這份 Schema 產生 SQL DDL、GORM model、Protocol Buffers 定義與 Go 包裝函數；也能把表格資料匯入資料庫，並把每張表存成 protobuf 快照。

適合「資料由非工程人員在試算表裡維護，程式端需要多種格式」的情境：同一份表格定義，資料庫、後端與客戶端各取所需，結構永遠一致。

---

## 架構

```text
            Parser                     Transformer                      Generator
 ┌──────────────────────────┐  ┌───────────────────────────┐  ┌─────────────────────────────┐
 │ excel.Loader             │  │ resolver                  │  │ sql      → CREATE TABLE      │
 │   .xlsx → []Sheet        │→ │   ref: 參考 → 巢狀欄位     │→ │ orm      → GORM model        │
 │ source.Parser            │  │ yaml generator            │  │ proto    → .proto / plugin   │
 │   表頭 → schema.Entity   │  │   型別正規化 → YAML Schema │  │ wrapper  → envelope 包裝函數 │
 └──────────────────────────┘  └───────────────────────────┘  └─────────────────────────────┘
                                          │
                                          ▼
                     importer（Excel → DB，可選 protobuf 快照）
```

- **Parser**：`excel.Loader` 讀出工作表，`source.Parser` 依表頭版面解析出 `schema.Entity`（欄位名、型別、陣列、參考、結構陣列）。
- **Transformer**：`resolver` 解析跨表參考，yaml generator 依設定檔把各種型別寫法正規化，輸出 YAML Schema。YAML Schema 是後續所有生成器的唯一來源。
- **Generator**：每種輸出格式一個生成器，共用 `Generate([]schema.Entity) ([]byte, error)` 介面，各自用 Options 模式設定。

---

## 快速開始

需要 Go 1.25 以上。repo 內附一組虛構的範例表（`testdata/Demo_*.xlsx`），可以直接跑完整流程：

```bash
git clone https://github.com/miskawu/formgen.git
cd formgen
go build -o formgen .

# 1. 產生設定檔（型別映射、表頭版面）
./formgen config init

# 2. Excel → YAML Schema（多個檔案合併成一份）
./formgen schema testdata/Demo_*.xlsx -o out/schema.yaml

# 3. YAML Schema → 各種輸出
./formgen sql   -i out/schema.yaml -o out/schema.sql
./formgen orm   -i out/schema.yaml -o out/models.go --proto-path example.com/demo/pb
./formgen proto -i out/schema.yaml -o out/schema.proto --envelope --go-package "example.com/demo/pb;pb"
./formgen wrapper -i out/schema.yaml -o out/envelope_gen.go \
  --proto-path example.com/demo/pb --proto-alias pb
```

產生結果可以和 `testdata/sample.*`（golden 檔）對照。

匯入資料庫（MySQL）：

```bash
./formgen db migrate load --file out/schema.sql --dir out/migrations
./formgen db migrate up --dir out/migrations --dsn "user:pass@tcp(127.0.0.1:3306)/demo"
./formgen db import "testdata/Demo_*.xlsx" \
  --dsn "user:pass@tcp(127.0.0.1:3306)/demo?charset=utf8mb4&parseTime=True&loc=Local" \
  --mode upsert \
  --snapshot default --snapshot-proto out/schema.proto --snapshot-schema out/schema.yaml
```

---

## Excel 表頭格式

formgen 讀取每個檔案的**第一個工作表**，工作表名稱就是 Entity（資料表）名稱；其餘工作表（欄位說明、修改紀錄等）會被忽略。預設版面如下：

| 列 | 用途 | 範例 |
|----|------|------|
| 1 | 顯示名稱／註解（不讀取） | `商店Id`、`商品群組` |
| 2 | **欄位名稱** | `Id`、`GoodsGroupId`、`Loot[0].ItemId` |
| 3 | **值域類型** | `entry`、`int`、`[]tag`、`*entry`、`*[]shortint` |
| 4 | **metadata** | `ref:Demo_ShopLabel`、`ref:Demo_ShopGoods.GroupId` |
| 5 | **輸出類型** | `both`、`client`、`skip` |
| 6+ | 資料 | |

範例（`testdata/Demo_Shop.xlsx` 節錄）：

| 商店Id | 開關 | 標籤 | 排序 | 商品群組 | 名稱備註 |
|---|---|---|---|---|---|
| Id | __extra | Label | SortOrder | GoodsGroupId | NameMemo |
| entry | int | *entry | int | *[]shortint | string |
| | cond:bool | ref:Demo_ShopLabel | | ref:Demo_ShopGoods.GroupId | |
| both | skip | both | client | both | skip |
| 1 | 1 | 1 | 1 | 11 | 一般商店 |

### 值域類型

- **基本型別**：`int`、`int8`…`uint64`、`float`、`double`、`string`、`bool`、`bytes`、`timestamp`、`date`、`text`、`json` 等；也可在設定檔自訂別名（預設已含 `entry`、`asset`、`tag`、`tinyint`、`shortint`）。型別會依 `formgen.yaml` 的 `mappings.custom` 正規化，沒有對應的型別會報錯。
- **`[]` 陣列**：`[]int`、`[]tag`，資料以 JSON 陣列填寫（`[1,2]`、`["a","b"]`）；SQL 存成 JSON 欄位、proto 為 `repeated`。
- **`*` 參考**：表示此欄位指向另一張表，參考目標寫在 metadata 列（見下方）。

### metadata：`ref:` 參考

| 寫法 | 意義 | proto 輸出 |
|------|------|-----------|
| `*entry` + `ref:Demo_Region` | **整筆嵌入**：欄位值是被參考表的 Id | `DemoRegion region_id = 2;` |
| `*[]shortint` + `ref:Demo_ShopGoods.GroupId` | **欄位引用**：取出被參考表中該欄位相符的所有資料 | `repeated DemoShopGoods goods_group_id = 4;` |
| 找不到被參考的表 | 發出警告並退回一般欄位（欄位引用會退回非陣列） | `int32 drops = 6;` |

沒有 `*` 修飾的欄位，metadata 只是註解（例如 `cond:bool`）。

### 輸出類型

填 `skip` 的欄位不會進入 Schema、資料庫與快照；其他值（如 `both`、`client`）目前一律輸出，可作為給人看的標註。

### 結構陣列

欄位名稱寫成 `Array[index].Field` 會被聚合成一個結構陣列欄位：

```text
Loot[0].Kind  Loot[0].ItemId  Loot[0].Amount  Loot[1].Kind  Loot[1].ItemId ...
```

→ YAML `type: LootStruct, isStruct: true`、proto 的巢狀 `message LootStruct` + `repeated LootStruct loot`、SQL/匯入時存成 JSON 陣列。同名欄位在各 index 的型別必須一致。

### 自訂表頭版面

如果你的表格版面不同，可以在 `formgen.yaml` 調整各列位置（Excel 列號，第一列為 1）與 skip 標記，`schema` 與 `db import` 都會套用：

```yaml
layout:
  nameRow: 1
  typeRow: 2
  metaRow: 3
  outputRow: 4
  dataRow: 5
  skipMarker: "-"
```

---

## 亮點

### `ref:` 參考解析

`resolver` 在 Schema 階段就把跨表參考展開：整筆嵌入的欄位帶上被參考表的完整欄位（`nestedColumns`），欄位引用記錄目標欄位；找不到的參考會發警告並安全地退回一般欄位，不會中斷整批轉換。匯入時，importer 先替被參考的表建立索引（依 Id 或依指定欄位），序列化快照時自動把資料嵌進去，被整筆嵌入的表不再另外產生快照。

### protocompile + dynamicpb：執行期序列化

匯入快照不需要先用 `protoc` 產生 Go 程式碼。`serializer` 在執行期用 [`bufbuild/protocompile`](https://github.com/bufbuild/protocompile) 編譯 `.proto`，以 `dynamicpb` 建立 message，把試算表的字串值依欄位型別轉換後填入（含巢狀 message、repeated、JSON 陣列），再以 deterministic 模式序列化並 gzip。checksum 不變時快照不改寫，變動時追加版本紀錄。反序列化同樣走 `dynamicpb`，可直接把快照還原成 `map[string]any`。

### 模板化 proto plugin

`formgen proto --plugin <file.tmpl>` 讓你用 Go template 決定整份 `.proto` 的樣子。模板拿到的資料包含所有 message、List wrapper 與 oneof 欄位清單（`PayloadFields`），可以自由加上自己的 service、envelope 或其他 message。`--plugin default` 使用內建模板；`testdata/envelope_plugin.tmpl` 是一個完整的自訂範例。

### oneof envelope 與包裝函數自動生成

`formgen proto --envelope` 會為每張表產生 `XxxList` wrapper，並額外產生：

```protobuf
message TableEnvelope {
  string key = 1;
  string checksum = 2;

  oneof payload {
    DemoCategoryList demo_category_list = 10;
    DemoShopList demo_shop_list = 14;
    // ...每張表一個欄位
  }
}

message TableBundle {
  string version = 1;
  repeated TableEnvelope items = 2;
}
```

`formgen wrapper` 再產生對應的 Go 包裝函數，用 switch-case 做靜態型別斷言，把任一張表的 `XxxList` 放進 envelope 的 oneof，不必手寫幾十個 case：

```go
func wrapEnvelope(name, checksum string, msg proto.Message) (*pb.TableEnvelope, error) {
	item := &pb.TableEnvelope{Key: name, Checksum: checksum}
	switch name {
	case "Demo_Shop":
		v, ok := msg.(*pb.DemoShopList)
		if !ok {
			return nil, fmt.Errorf("type mismatch: expected *DemoShopList, got %T", msg)
		}
		item.Payload = &pb.TableEnvelope_DemoShopList{DemoShopList: v}
	// ...
	}
	return item, nil
}
```

envelope 名稱可用 `--envelope <Name>` 指定，搭配自訂 plugin 模板使用。envelope 是可選的輸出格式，不需要時只用 `--wrapper` 或預設輸出即可。

---

## 指令一覽

| 指令 | 說明 |
|------|------|
| `formgen config init` | 產生 `formgen.yaml`（型別映射、表頭版面） |
| `formgen schema <xlsx...> -o schema.yaml` | Excel → YAML Schema，支援多檔與 glob |
| `formgen sql -i schema.yaml -o schema.sql` | CREATE TABLE（第一欄為主鍵、陣列存 JSON、保留字加 `_field`） |
| `formgen orm -i schema.yaml -o models.go [--proto-path ...]` | GORM model，可選 `ToProto()` |
| `formgen proto -i schema.yaml -o schema.proto [--wrapper\|--envelope\|--plugin ...]` | Protocol Buffers 定義 |
| `formgen wrapper -i schema.yaml --proto-path ... [--envelope Name]` | envelope 包裝函數 |
| `formgen db migrate up\|down\|steps\|version\|force\|create\|load` | 資料庫遷移（golang-migrate） |
| `formgen db import <xlsx...> --dsn ... [--mode ...] [--snapshot ...]` | 匯入資料，可選 protobuf 快照 |

設定檔搜尋順序：`--config` 指定的路徑 → `./formgen.yaml` → `$HOME/formgen.yaml` → `/etc/formgen/formgen.yaml`。

### 型別映射

YAML Schema 中的正規化型別再依各生成器的預設映射輸出：

| Schema 型別 | SQL | Go (ORM) | Proto |
|-------------|-----|----------|-------|
| int8 / int16 / int32 / int64 | TINYINT / SMALLINT / INT / BIGINT | int8 / int16 / int32 / int64 | int32 / int32 / int32 / int64 |
| uint8 … uint64 | … UNSIGNED | uint8 … uint64 | uint32 / uint64 |
| float32 / float64 | FLOAT / DOUBLE | float32 / float64 | float / double |
| string / text | VARCHAR(255) / TEXT | string | string |
| bool | BOOLEAN | bool | bool |
| bytes | BLOB | []byte | bytes |
| timestamp / datetime | TIMESTAMP / DATETIME | time.Time | int64 |
| 陣列 | JSON | slice | repeated |

各生成器都提供 `WithCustomMappings` 覆寫映射，詳見 [`internal/generator/*/mapper.go`](internal/generator)。

---

## 專案結構

```text
formgen/
├── cmd/                     # CLI（cobra）
├── internal/
│   ├── source/              # Parser：Sheet、表頭版面、欄位解析
│   │   ├── excel/           # Excel 載入
│   │   └── resolver/        # 跨表參考解析
│   ├── schema/              # 中介 Schema 定義與 YAML 載入
│   ├── generator/           # Generator：sql / orm / proto / wrapper / yaml
│   ├── mapper/              # 型別映射
│   ├── config/              # 設定檔（viper）
│   ├── importer/            # Excel → DB 匯入與快照
│   ├── serializer/          # protocompile + dynamicpb 序列化
│   ├── migration/           # golang-migrate 包裝
│   ├── models/              # 快照與版本表
│   └── sampledata/          # 範例表定義（產生 testdata/*.xlsx）
├── migrations/              # 預設的遷移檔目錄
└── testdata/                # 範例 Excel 與 golden 輸出
```

各模組的細節：[SQL Generator](internal/generator/sql/README.md)、[ORM Generator](internal/generator/orm/README.md)、[Importer](internal/importer/README.md)、[Migration](internal/migration/README.md)。

---

## 開發

```bash
go test ./...                                  # 全部測試
go test ./... -coverprofile=coverage.out       # 覆蓋率
go tool cover -html=coverage.out

go generate ./internal/sampledata              # 修改範例表後重新產生 testdata/Demo_*.xlsx
go test ./cmd -run TestGolden -update          # 修改生成器後更新 testdata/sample.* golden 檔
```

- 範例表全部由 [`internal/sampledata`](internal/sampledata/sampledata.go) 以程式產生，輸出位元組可重現；測試會檢查 `testdata/` 中的檔案是否與定義一致。
- `cmd` 的 golden 測試用範例表跑完整條 `schema → sql / orm / proto / wrapper` 流程並比對輸出。
- importer 的端對端測試以 SQLite 跑匯入、參考嵌入與快照（[sample_e2e_test.go](internal/importer/sample_e2e_test.go)）。

## 已知限制

- ORM 的 `ToProto()` 目前直接逐欄指派，只適用於 Go 型別與 proto 型別一致的欄位；`int8`/`int16`（proto 為 `int32`）、參考欄位與結構陣列需要自行轉換（`testdata/sample.go` 可看到這些情況）。
- GORM model 的欄位 tag 使用原始欄位名，SQL DDL 與 importer 使用 snake_case；兩者混用時需自行對齊命名策略。
- 輸出類型列目前只區分 `skip` 與其他值。
- `db import` 與 `db migrate` 目前只支援 MySQL。

## License

[MIT](LICENSE) © 2026 MiskaWu
