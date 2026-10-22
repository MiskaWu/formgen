---
name: schema-validate
description: Schema 驗證 - 驗證 YAML Schema 的正確性
---

# Schema 驗證

驗證 YAML Schema 檔案的正確性和一致性，確保能被所有生成器正確處理。

## 執行步驟

### 1. 載入 Schema

讀取指定的 YAML Schema 檔案：

```bash
# 如果用戶未指定，列出可用的 schema 檔案
ls *.yaml testdata/*.yaml
```

解析 YAML 結構，確認基本格式正確。

### 2. 驗證 Entity 結構

每個 Entity 應該有：

```yaml
version: "1"
entities:
  - name: EntityName      # 必須：Entity 名稱
    columns:              # 必須：欄位列表
      - name: fieldName   # 必須：欄位名稱
        type: string      # 必須：正規化後的類型
        isArray: false    # 可選：是否為陣列
        ref: OtherEntity  # 可選：參考目標（Entity 或 Entity.Field）
```

**檢查點**：
- [ ] 每個 Entity 是否有 `name`
- [ ] 每個 Entity 是否有 `columns`
- [ ] 每個 Column 是否有 `name` 和 `type`

### 3. 驗證類型定義

確認所有 `type` 都是有效的類型：

#### 基本類型

| 類型 | 說明 |
|------|------|
| `int32`, `int64` | 整數 |
| `float`, `double` | 浮點數 |
| `string` | 字串 |
| `bool` | 布林值 |
| `timestamp` | 時間戳 |
| `[]type` | 陣列類型 |

#### 自定義類型（來自配置）

檢查 `formgen.yaml` 中的自定義映射（`formgen schema` 用它把 Excel 類型正規化）：

```yaml
mappings:
  custom:
    entry: int32
    asset: string
```

**檢查點**：
- [ ] 所有 `type` 是否為已知類型
- [ ] 未知類型是否有自定義映射
- [ ] 報告無法識別的類型

### 4. 驗證參考關係

對於有 `ref` 的欄位：

```yaml
# 整筆嵌入：nestedColumns 為被參考表的欄位
- name: RegionId
  type: int32
  isArray: false
  ref: Demo_Region
  nestedColumns: [...]

# 欄位引用：取出被參考表中該欄位相符的所有資料
- name: GoodsGroupId
  type: int16
  isArray: true
  ref: Demo_ShopGoods.GroupId
```

**檢查點**：
- [ ] `ref` 的 Entity 是否存在於 Schema 中
- [ ] 如果是 `Entity.Field` 形式，目標 Entity 是否有該欄位
- [ ] 是否有循環參考（A → B → A）

### 5. 驗證陣列欄位

對於 `isArray: true` 的欄位：

```yaml
# 簡單陣列
- name: Tags
  type: string
  isArray: true

# 結構陣列（由 Loot[0].ItemId 這類表頭產生）
- name: Loot
  type: LootStruct
  isArray: true
  isStruct: true
  nestedColumns:
    - name: ItemId
      type: int32
      isArray: false
    - name: Amount
      type: int32
      isArray: false
```

**檢查點**：
- [ ] 結構陣列是否有 `nestedColumns`
- [ ] `nestedColumns` 中的類型是否有效

### 6. 跨生成器兼容性檢查

確認 Schema 能被所有生成器正確處理：

#### SQL 生成器
- 欄位名是否為 SQL 保留字（會自動加引號）
- 陣列類型會轉為 JSON

#### ORM 生成器
- 欄位名是否為 Go 保留字
- 結構陣列會生成嵌套 struct

#### Proto 生成器
- 欄位名是否為 Proto 保留字
- 類型是否都有對應的 Proto 類型

**檢查點**：
- [ ] 列出使用保留字的欄位（警告，不是錯誤）
- [ ] 確認所有類型都有對應映射

### 7. 輸出報告

```
## Schema 驗證報告

### 檔案資訊
- 檔案: testdata/sample.yaml
- Entity 數量: 5
- 總欄位數: 42

### 驗證結果

#### ✅ 通過
- [x] Entity 結構正確
- [x] 所有類型定義有效
- [x] 參考關係正確

#### ⚠️ 警告
- `UserInfo.select` 是 SQL 保留字，SQL 生成時會加引號
- `ItemInfo.type` 是多個語言的保留字

#### ❌ 錯誤
- `OrderInfo.itemRef` 參考的 Entity `ItemData` 不存在
- `RewardConfig.rewards` 是結構陣列但缺少 nestedColumns

### 類型統計
| 類型 | 使用次數 |
|------|---------|
| int32 | 15 |
| string | 12 |
| int64 | 8 |
| []int32 | 4 |
| entry | 3 |

### 參考關係圖
UserInfo → RoleInfo → SkillInfo
OrderInfo → ItemInfo
```

## 快速驗證命令

```bash
# 生成各格式輸出來驗證（乾跑）
formgen sql -i testdata/sample.yaml -o /tmp/check.sql
formgen orm -i testdata/sample.yaml -o /tmp/check.go
formgen proto -i testdata/sample.yaml -o /tmp/check.proto
```

如果任何生成器報錯，Schema 就有問題。
