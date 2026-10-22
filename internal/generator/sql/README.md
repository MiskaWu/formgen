# SQL Generator 使用範例

這是一個從 Schema 定義生成 SQL DDL (CREATE TABLE) 語句的生成器。

## 功能特性

1. **類型映射到 SQL 類型**
   - 支援所有基本整數類型 (int, int32, int64, uint, uint32, uint64)
   - 支援浮點數類型 (float, float32, float64, double)
   - 支援字符串和布爾類型
   - 支援時間類型 (timestamp, datetime, date, time)
   - 支援文字和 JSON 類型
   - 支援二進位類型 (byte, bytes → BLOB)

2. **自動 DDL 生成**
   - 自動將第一個欄位設為 PRIMARY KEY
   - 主鍵自動加上 NOT NULL 約束
   - 陣列類型自動轉換為 JSON 類型儲存
   - 為陣列欄位自動加上 COMMENT

3. **自定義類型映射**
   - 支援透過選項函數擴展或覆蓋類型映射

## 類型映射表

| 來源類型 | SQL 類型 |
|---------|----------|
| int, int32 | INT |
| int64 | BIGINT |
| uint, uint32 | INT UNSIGNED |
| uint64 | BIGINT UNSIGNED |
| float, float32 | FLOAT |
| float64, double | DOUBLE |
| string | VARCHAR(255) |
| bool | BOOLEAN |
| byte, bytes | BLOB |
| timestamp | TIMESTAMP |
| datetime | DATETIME |
| date | DATE |
| time | TIME |
| text | TEXT |
| json | JSON |

## 程式碼範例

```go
package main

import (
    "fmt"
    "github.com/miskawu/formgen/internal/generator/sql"
    "github.com/miskawu/formgen/internal/schema"
)

func main() {
    // 建立一個實體定義
    entity := schema.Entity{
        Name: "users",
        Columns: []schema.Column{
            {Name: "id", RawType: "int64", IsArray: false},
            {Name: "username", RawType: "string", IsArray: false},
            {Name: "email", RawType: "string", IsArray: false},
            {Name: "age", RawType: "int", IsArray: false},
            {Name: "balance", RawType: "double", IsArray: false},
            {Name: "is_active", RawType: "bool", IsArray: false},
            {Name: "tags", RawType: "string", IsArray: true},
            {Name: "created_at", RawType: "timestamp", IsArray: false},
        },
    }

    // 使用預設選項建立生成器
    gen := sql.New()

    // 生成 SQL
    output, err := gen.Generate([]schema.Entity{entity})
    if err != nil {
        panic(err)
    }

    fmt.Println(string(output))
}
```

## 生成結果

上面的程式碼會生成以下 SQL DDL：

```sql
CREATE TABLE users (
    id BIGINT PRIMARY KEY NOT NULL,
    username VARCHAR(255),
    email VARCHAR(255),
    age INT,
    balance DOUBLE,
    is_active BOOLEAN,
    tags JSON COMMENT 'array field (stored as JSON)',
    created_at TIMESTAMP
);
```

## 自定義選項

所有配置選項定義在 [options.go](options.go) 中。

### 自定義類型映射

如果需要自定義類型映射，可以使用 `WithCustomMappings` 選項：

```go
customMappings := map[string]string{
    "entry": "INT",
    "userId": "VARCHAR(50)",
    "longText": "LONGTEXT",
}

gen := sql.New(sql.WithCustomMappings(customMappings))
```

## 陣列處理

陣列類型會自動轉換為 JSON 類型儲存，並加上註解說明：

```go
entity := schema.Entity{
    Name: "posts",
    Columns: []schema.Column{
        {Name: "id", RawType: "int64", IsArray: false},
        {Name: "tags", RawType: "string", IsArray: true},
        {Name: "scores", RawType: "float", IsArray: true},
    },
}
```

生成結果：

```sql
CREATE TABLE posts (
    id BIGINT PRIMARY KEY NOT NULL,
    tags JSON COMMENT 'array field (stored as JSON)',
    scores JSON COMMENT 'array field (stored as JSON)'
);
```

## 測試

執行測試：

```bash
go test ./internal/generator/sql/... -v
```

所有測試包含：
- 類型映射測試
- 基本表生成測試
- 陣列欄位處理測試
- 自定義映射測試
- 混合類型測試

## SQL 保留字處理

生成器會自動檢測並處理 MySQL 保留字。當欄位名與保留字衝突時，會自動加上 `_field` 後綴：

| 原始欄位名 | 生成的欄位名 |
|-----------|-------------|
| `order` | `order_field` |
| `desc` | `desc_field` |
| `type` | `type_field` |
| `status` | `status_field` |
| `level` | `level_field` |
| `rank` | `rank_field` |

支援超過 250 個 MySQL 保留字，包括：
- DDL/DML 關鍵字：`select`, `insert`, `update`, `delete`, `create`, `alter` 等
- 資料類型：`int`, `varchar`, `text`, `date`, `time`, `timestamp`, `json` 等
- 聚合函數：`count`, `sum`, `avg`, `min`, `max`, `rank` 等
- 其他常見衝突：`type`, `status`, `level`, `match`, `range`, `value`, `data` 等

### 自定義後綴

可以使用 `WithKeywordSuffix` 選項自定義後綴：

```go
gen := sql.New(sql.WithKeywordSuffix("_col"))
// order -> order_col
```

完整的保留字列表請參考 [keyword.go](keyword.go)。

## 注意事項

1. **主鍵設定**：第一個欄位會自動被設為 PRIMARY KEY
2. **陣列儲存**：所有陣列類型都會被轉換為 JSON 類型
3. **NOT NULL**：只有主鍵欄位會自動加上 NOT NULL 約束
4. **VARCHAR 長度**：預設 VARCHAR 長度為 255，可透過自定義映射修改
5. **保留字處理**：與 SQL 保留字衝突的欄位名會自動加上後綴
