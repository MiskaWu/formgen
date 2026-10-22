# Migration 使用說明

## 概述

本項目使用 [golang-migrate/migrate](https://github.com/golang-migrate/migrate) 作為生產級數據庫遷移工具。

## 功能特點

- 版本控制的數據庫架構變更
- 支持向上和向下遷移
- 自動從 SQL 文件生成遷移
- 安全的回滾機制
- 強制修復損壞的遷移狀態

## 安裝依賴

```bash
go get -u github.com/golang-migrate/migrate/v4
go get -u github.com/golang-migrate/migrate/v4/database/mysql
go get -u github.com/golang-migrate/migrate/v4/source/file
go get -u github.com/go-sql-driver/mysql
```

## 使用方法

### 1. 從 SQL 文件生成遷移文件

```bash
# 從 SQL 文件生成遷移（不需要數據庫連接）
go run main.go db migrate load --file=./schema.sql

# 指定遷移文件目錄
go run main.go db migrate load --file=./schema.sql --dir=./migrations
```

這會自動創建：
- `{timestamp}_{name}.up.sql` - 包含 CREATE TABLE 語句
- `{timestamp}_{name}.down.sql` - 包含對應的 DROP TABLE 語句

### 2. 執行遷移

```bash
# 執行所有待執行的遷移
go run main.go db migrate up --dsn="user:password@tcp(localhost:3306)/mydb"

# 執行指定步數的遷移
go run main.go db migrate steps --n=1 --dsn="user:password@tcp(localhost:3306)/mydb"
```

### 3. 回滾遷移

```bash
# 回滾所有遷移
go run main.go db migrate down --dsn="user:password@tcp(localhost:3306)/mydb"

# 回滾指定步數
go run main.go db migrate steps --n=-1 --dsn="user:password@tcp(localhost:3306)/mydb"
```

### 4. 查看當前版本

```bash
go run main.go db migrate version --dsn="user:password@tcp(localhost:3306)/mydb"
```

### 5. 創建新的遷移文件

```bash
# 創建空白遷移文件（不需要數據庫連接）
go run main.go db migrate create add_user_table
```

### 6. 強制修復版本（謹慎使用）

```bash
# 當遷移狀態損壞時使用
go run main.go db migrate force --version=1 --dsn="user:password@tcp(localhost:3306)/mydb"
```

## 配置參數

| 參數 | 說明 | 默認值 | 必需 |
|------|------|--------|------|
| `--dsn` | 數據庫連接字符串（需包含數據庫名稱） | - | 是（除 create/load） |
| `--dir` | 遷移文件目錄 | `./migrations` | 否 |
| `--file` | SQL 文件路徑（load 命令） | - | 是（load 命令） |
| `--n` | 遷移步數（steps 命令） | `1` | 否 |
| `--version` | 版本號（force 命令） | - | 是（force 命令） |
| `--multi-statements` | 啟用多語句支援 | `true` | 否 |

### 關於 multi-statements

MySQL 預設不支援一次執行多條 SQL 語句。本工具**預設自動啟用** `multiStatements=true`，
讓包含多個 `CREATE TABLE` 的遷移檔案能正常執行。

如果需要停用（例如出於安全考量）：
```bash
go run main.go db migrate up --dsn="..." --multi-statements=false
```

> **安全說明**：`multiStatements` 在接受使用者輸入時可能增加 SQL 注入風險。
> 但在 migration 場景下是安全的，因為 migration 檔案是預定義的，不接受外部輸入。

## DSN 格式

MySQL DSN 格式（**必須包含數據庫名稱**）：
```
user:password@tcp(host:port)/dbname[?options]
```

示例：
```
root:mypassword@tcp(localhost:3306)/myapp
admin:secret@tcp(db.example.com:3306)/production?charset=utf8mb4
```

## 遷移文件命名規則

遷移文件使用時間戳命名：
```
{timestamp}_{name}.up.sql    # 向上遷移
{timestamp}_{name}.down.sql  # 向下遷移
```

示例：
```
migrations/
├── 20260123180000_init.up.sql
├── 20260123180000_init.down.sql
├── 20260123180001_add_index.up.sql
└── 20260123180001_add_index.down.sql
```

## 最佳實踐

1. **總是先測試**：在開發環境測試遷移後再部署到生產環境
2. **編寫回滾腳本**：確保每個 UP 遷移都有對應的 DOWN 遷移
3. **版本控制**：將遷移文件納入 Git 版本控制
4. **小步迭代**：避免在單個遷移中進行過多變更
5. **備份數據**：在執行遷移前備份生產數據庫

## 故障排除

### 遷移狀態損壞（dirty）

如果遷移失敗導致狀態損壞：

1. 檢查當前版本：
   ```bash
   go run main.go db migrate version --dsn="user:password@tcp(localhost:3306)/mydb"
   ```

2. 手動修復數據庫到正確狀態（例如刪除部分建立的表）

3. 強制設置版本：
   ```bash
   # 設置為上一個成功的版本
   go run main.go db migrate force --version=N --dsn="user:password@tcp(localhost:3306)/mydb"

   # 或重置為初始狀態（-1 表示無任何 migration）
   go run main.go db migrate force --version=-1 --dsn="user:password@tcp(localhost:3306)/mydb"
   ```

4. 重新執行遷移：
   ```bash
   go run main.go db migrate up --dsn="user:password@tcp(localhost:3306)/mydb"
   ```

### 找不到遷移文件

確保 `--dir` 參數指向正確的遷移文件目錄。

## 完整示例

```bash
# 1. 從 SQL 文件生成遷移
go run main.go db migrate load --file=./schema.sql

# 2. 執行遷移
go run main.go db migrate up --dsn="root:password@tcp(localhost:3306)/myapp"

# 3. 查看狀態
go run main.go db migrate version --dsn="root:password@tcp(localhost:3306)/myapp"
```
