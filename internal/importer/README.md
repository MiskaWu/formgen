# Importer 套件

把 Excel 工作表的資料匯入資料庫，並可選擇性地把每張表序列化成 protobuf 快照。

## 功能特點

- 動態匯入：用 `db.Table(name).Create(map)` 寫入，不需要預先產生 model
- 所有工作表可放在同一個交易中，失敗時整批回滾
- 四種匯入模式：insert / upsert / replace / truncate
- 結構陣列欄位（`Loot[0].ItemId`、`Loot[1].ItemId`…）自動聚合成 JSON 欄位
- 表頭版面可設定（`WithLayout`），需與產生 schema 時一致
- Protobuf 快照（可選）：gzip 壓縮、checksum 不變時不改寫，並自動嵌入被參考的表

## 使用方式

### 基本用法

```go
package main

import (
    "fmt"
    "os"

    "github.com/miskawu/formgen/internal/importer"
    "github.com/miskawu/formgen/internal/source/excel"

    "gorm.io/driver/mysql"
    "gorm.io/gorm"
)

func main() {
    db, err := gorm.Open(mysql.Open("user:pass@tcp(127.0.0.1:3306)/demo"), &gorm.Config{})
    if err != nil {
        panic(err)
    }

    data, err := os.ReadFile("testdata/Demo_Category.xlsx")
    if err != nil {
        panic(err)
    }
    sheets, err := excel.NewLoader().Load(data)
    if err != nil {
        panic(err)
    }

    imp := importer.New(db, "",
        importer.WithBatchSize(100),
        importer.WithImportMode(importer.ImportModeUpsert),
        importer.WithProgressCallback(func(table string, current, total int) {
            fmt.Printf("匯入 %s: %d/%d\n", table, current, total)
        }),
    )

    // 只匯入第一個工作表（其他工作表通常是說明或修改紀錄）
    if err := imp.Import(sheets[0]); err != nil {
        panic(err)
    }
}
```

### 使用 CLI

```bash
# 基本匯入（表需先以 migration 建好）
formgen db import testdata/Demo_Category.xlsx \
  --dsn "user:pass@tcp(127.0.0.1:3306)/demo?charset=utf8mb4&parseTime=True&loc=Local"

# 多個檔案在同一個交易中匯入，重複匯入時更新既有資料
formgen db import "testdata/Demo_*.xlsx" --dsn "..." --mode upsert

# 啟用快照
formgen db import "testdata/Demo_*.xlsx" --dsn "..." \
  --snapshot default \
  --snapshot-proto testdata/sample.proto \
  --snapshot-schema testdata/sample.yaml
```

### 匯入模式

| 模式 | 說明 | 適用場景 |
|------|------|---------|
| `insert` | 僅插入，遇到主鍵衝突時失敗（預設） | 首次匯入 |
| `upsert` | 已存在則更新（MySQL：`ON DUPLICATE KEY UPDATE`） | 重複匯入 |
| `replace` | `REPLACE INTO`（先刪後插） | 需要整列替換 |
| `truncate` | 匯入前清空表 | 僅限開發環境 |

upsert 以第一個一般欄位作為衝突鍵，與 SQL 生成器把第一欄設為 `PRIMARY KEY` 的慣例一致。

### 配置選項（[options.go](options.go)）

- `WithBatchSize(size int)`：進度回報的批次大小（預設 100）
- `WithProgressCallback(cb)`：進度回調
- `WithImportMode(mode)`：匯入模式
- `WithKeywordSuffix(suffix)`：欄位名與 SQL 保留字衝突時的後綴（預設 `_field`，需與 SQL 生成器一致）
- `WithLayout(layout)`：表頭版面（預設 `source.DefaultLayout()`）
- `WithSnapshot(SnapshotConfig)`：啟用快照
  - `ProtoFilePath`：`.proto` 檔路徑（需含各表的 `XxxList` wrapper，即 `formgen proto --wrapper` 或 `--envelope` 的輸出）
  - `Namespace`：命名空間
  - `EmbeddedEntities`：被整筆嵌入的表，這些表不單獨產生快照
  - `Entities`：已解析的 schema，用來辨識參考欄位

## Protobuf 快照

啟用後，每張表匯入完成時會：

1. 把所有資料列序列化成 `XxxList` wrapper（`protocompile` 執行期編譯 `.proto` + `dynamicpb`，不需要預先產生 Go 程式碼）
2. gzip 壓縮並計算 MD5 checksum
3. 寫入 `snapshots` 表；checksum 與既有快照相同時不做任何變更
4. checksum 變動（或首次寫入）時在 `versions` 表追加一筆 UUID 版本

`snapshots` / `versions` 的結構見 [internal/models](../models)。

### 參考嵌入

當 schema 中有 `*` 參考欄位時，快照會自動帶入被參考表的資料：

| 模式 | 表頭寫法 | 行為 | 被參考表是否另外產生快照 |
|------|---------|------|------------------------|
| 整筆嵌入 | 類型 `*entry`、metadata `ref:Demo_Region` | 以欄位值當 ID 找出單筆資料 | 否（資料已嵌入） |
| 欄位引用 | 類型 `*[]shortint`、metadata `ref:Demo_ShopGoods.GroupId` | 以欄位值比對被參考表的指定欄位，取出所有符合的資料 | 是 |

以範例資料為例：

```text
Demo_Mission.RegionId = 101   → 嵌入 Demo_Region 中 Id = 101 的那一筆
Demo_Shop.GoodsGroupId = 11   → 嵌入 Demo_ShopGoods 中 GroupId = 11 的所有資料
```

對應的 proto：

```protobuf
message DemoMission {
  DemoRegion region_id = 2;               // 單一 message
}

message DemoShop {
  repeated DemoShopGoods goods_group_id = 4;  // repeated message
}
```

### 程式化使用

```go
imp := importer.New(db, "",
    importer.WithSnapshot(importer.SnapshotConfig{
        ProtoFilePath:    "testdata/sample.proto",
        Namespace:        "default",
        EmbeddedEntities: resolver.FindEmbeddedEntities(entities),
        Entities:         entities,
    }),
)
if err := imp.EnsureSnapshotTables(); err != nil { // DDL 放在交易外
    return err
}

// 先為所有被參考的表建立索引
referenced := resolver.FindReferencedEntities(entities)
for _, sheet := range sheets {
    if referenced[sheet.Name] {
        if err := imp.IndexSheetData(sheet); err != nil {
            return err
        }
    }
}

// 再於同一個交易中匯入
return db.Transaction(func(tx *gorm.DB) error {
    for _, sheet := range sheets {
        if err := imp.ImportWithTx(tx, sheet); err != nil {
            return err
        }
    }
    return nil
})
```

完整範例見 [sample_e2e_test.go](sample_e2e_test.go)（以 SQLite 跑完整條匯入與快照流程）。

## 注意事項

- 匯入前需先建立資料表（`formgen sql` + `formgen db migrate`）
- 表名與欄位名會轉成 snake_case，與 SQL 生成器一致；與 SQL 保留字衝突的欄位加上 `_field` 後綴
- 快照的欄位名同樣是 snake_case，與 proto 生成器一致；`.proto` 中沒有定義的欄位會被略過
- 空白儲存格不寫入（交給資料庫預設值）
