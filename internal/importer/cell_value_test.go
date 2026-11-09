package importer

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/miskawu/formgen/internal/source"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// openTestDB 開一個測試用的記憶體 SQLite，測試結束時關閉。
func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get database instance: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	return db
}

var insertColumnsPattern = regexp.MustCompile(`\(([^()]*)\)\s*VALUES`)

// captureBoundValues 記錄每次寫入時綁定給指定欄位的參數。
//
// 匯入有兩條寫入路徑：insert／upsert 走 GORM 的 Create，replace 走 Exec，兩邊都要掛上。
// 回傳的是真正送進驅動的值，所以型別不對時，即使 SQLite 肯收也會被抓到。
func captureBoundValues(t *testing.T, db *gorm.DB, column string) *[]any {
	t.Helper()
	bound := &[]any{}
	capture := func(tx *gorm.DB) {
		matches := insertColumnsPattern.FindStringSubmatch(tx.Statement.SQL.String())
		if matches == nil {
			return
		}
		for i, name := range strings.Split(matches[1], ",") {
			if strings.Trim(strings.TrimSpace(name), "`\"") == column {
				*bound = append(*bound, tx.Statement.Vars[i])
				return
			}
		}
		t.Errorf("column %s not found in statement: %s", column, tx.Statement.SQL.String())
	}
	if err := db.Callback().Create().After("gorm:create").Register("test:capture_create", capture); err != nil {
		t.Fatalf("register create callback: %v", err)
	}
	if err := db.Callback().Raw().After("gorm:raw").Register("test:capture_raw", capture); err != nil {
		t.Fatalf("register raw callback: %v", err)
	}
	return bound
}

// TestImporter_BoolColumn_WritesBoolNotString 用範例表 Demo_Category 跑各匯入模式，
// 確認 bool 欄送進資料庫的是布林值而不是字串。
//
// 這張表刻意用 STRICT：SQLite 平常什麼都收，STRICT 才會像 MySQL strict mode 一樣
// 拒絕把 'true' 寫進整數欄。全部開 TEXT 欄的舊測試因此抓不到這個問題。
// TRUNCATE TABLE 在 SQLite 沒有這個語法；truncate 模式之後走的寫入和 insert 相同。
func TestImporter_BoolColumn_WritesBoolNotString(t *testing.T) {
	const ddl = `CREATE TABLE demo_category (id INT PRIMARY KEY, icon TEXT, name TEXT, enabled INT) STRICT`

	for _, mode := range []ImportMode{ImportModeInsert, ImportModeUpsert, ImportModeReplace} {
		t.Run(string(mode), func(t *testing.T) {
			db := openTestDB(t)
			if err := db.Exec(ddl).Error; err != nil {
				t.Fatalf("create table: %v", err)
			}
			bound := captureBoundValues(t, db, "enabled")

			imp := New(db, "", WithImportMode(mode))
			if err := imp.Import(loadSampleSheets(t, "Demo_Category")[0]); err != nil {
				t.Fatalf("Import() error = %v", err)
			}

			if want := []any{true, true, false}; !reflect.DeepEqual(*bound, want) {
				t.Errorf("bound enabled values = %#v, want %#v", *bound, want)
			}

			var stored []struct {
				Enabled int
				Kind    string
			}
			if err := db.Raw(`SELECT enabled, typeof(enabled) AS kind FROM demo_category ORDER BY id`).Scan(&stored).Error; err != nil {
				t.Fatalf("read back: %v", err)
			}
			got := fmt.Sprint(stored)
			if want := "[{1 integer} {1 integer} {0 integer}]"; got != want {
				t.Errorf("stored enabled = %s, want %s", got, want)
			}
		})
	}
}

// sheetWithColumn 建立一張只有 id 與一個待測欄位的工作表。
func sheetWithColumn(rawType string, cells ...string) source.Sheet {
	rows := [][]string{
		{"編號", "待測"},
		{"Id", "Flag"},
		{"int", rawType},
		{"", ""},
		{"", ""},
	}
	for i, cell := range cells {
		rows = append(rows, []string{fmt.Sprint(i + 1), cell})
	}
	return source.Sheet{Name: "items", Rows: rows}
}

// TestImporter_CellConversion 確認哪些欄位會轉成布林、哪些維持原字串。
func TestImporter_CellConversion(t *testing.T) {
	tests := []struct {
		name    string
		rawType string
		opts    []Option
		cells   []string
		want    []any
	}{
		{
			name:    "與快照相同的 bool 寫法都接受",
			rawType: "bool",
			cells:   []string{"true", "TRUE", "True", "t", "1", "false", "FALSE", "False", "f", "0"},
			want:    []any{true, true, true, true, true, false, false, false, false, false},
		},
		{
			name:    "設定檔裡對應到 bool 的別名也會轉換",
			rawType: "flag",
			opts:    []Option{WithTypeMappings(map[string]string{"flag": "bool"})},
			cells:   []string{"true", "false"},
			want:    []any{true, false},
		},
		{
			name:    "參考修飾符不影響判斷",
			rawType: "*bool",
			cells:   []string{"false"},
			want:    []any{false},
		},
		{
			name:    "別名對應到其他類型時維持字串",
			rawType: "bool",
			opts:    []Option{WithTypeMappings(map[string]string{"bool": "string"})},
			cells:   []string{"true"},
			want:    []any{"true"},
		},
		{
			name:    "bool 陣列存成 JSON，維持字串",
			rawType: "[]bool",
			cells:   []string{"[true,false]"},
			want:    []any{"[true,false]"},
		},
		{
			name:    "其他類型維持字串",
			rawType: "int",
			cells:   []string{"100"},
			want:    []any{"100"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := openTestDB(t)
			if err := db.Exec(`CREATE TABLE items (id INTEGER PRIMARY KEY, flag)`).Error; err != nil {
				t.Fatalf("create table: %v", err)
			}
			bound := captureBoundValues(t, db, "flag")

			if err := New(db, "", tt.opts...).Import(sheetWithColumn(tt.rawType, tt.cells...)); err != nil {
				t.Fatalf("Import() error = %v", err)
			}
			if !reflect.DeepEqual(*bound, tt.want) {
				t.Errorf("bound values = %#v, want %#v", *bound, tt.want)
			}
		})
	}
}

// TestImporter_InvalidBoolCell_ReportsLocation 確認打錯的 bool 值會指出位置並整批回滾，
// 而不是看目標資料庫的設定決定要報錯還是默默存進去。
func TestImporter_InvalidBoolCell_ReportsLocation(t *testing.T) {
	db := openTestDB(t)
	if err := db.Exec(`CREATE TABLE items (id INTEGER PRIMARY KEY, flag)`).Error; err != nil {
		t.Fatalf("create table: %v", err)
	}

	err := New(db, "").Import(sheetWithColumn("bool", "true", "maybe"))
	if err == nil {
		t.Fatal("Import() should fail on a cell that is not a bool")
	}
	for _, want := range []string{"row 7", "Flag", `"maybe"`} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %s", err, want)
		}
	}

	var count int64
	db.Table("items").Count(&count)
	if count != 0 {
		t.Errorf("items has %d rows after failed import, want 0", count)
	}
}
