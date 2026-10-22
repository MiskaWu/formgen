package importer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/miskawu/formgen/internal/models"
	"github.com/miskawu/formgen/internal/schema"
	"github.com/miskawu/formgen/internal/serializer"
	"github.com/miskawu/formgen/internal/source"
	"github.com/miskawu/formgen/internal/source/excel"
	"github.com/miskawu/formgen/internal/source/resolver"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const testdataDir = "../../testdata"

// loadSampleSheets 載入合成範例檔的第一個工作表
func loadSampleSheets(t *testing.T, names ...string) []source.Sheet {
	t.Helper()
	loader := excel.NewLoader()
	sheets := make([]source.Sheet, 0, len(names))
	for _, name := range names {
		data, err := os.ReadFile(filepath.Join(testdataDir, name+".xlsx"))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		loaded, err := loader.Load(data)
		if err != nil {
			t.Fatalf("load %s: %v", name, err)
		}
		sheets = append(sheets, loaded[0])
	}
	return sheets
}

// createTables 依 schema 建立全部欄位為 TEXT 的 sqlite 表（欄位命名規則與 importer 一致）
func createTables(t *testing.T, db *gorm.DB, imp *Importer, entities []schema.Entity) {
	t.Helper()
	for _, entity := range entities {
		cols := make([]string, 0, len(entity.Columns))
		for _, col := range entity.Columns {
			name := imp.formatColumnName(col.Name)
			def := name + " TEXT"
			if name == "id" {
				def += " PRIMARY KEY"
			}
			cols = append(cols, def)
		}
		ddl := fmt.Sprintf("CREATE TABLE %s (%s)", formatTableName(entity.Name), strings.Join(cols, ", "))
		if err := db.Exec(ddl).Error; err != nil {
			t.Fatalf("create table for %s: %v", entity.Name, err)
		}
	}
}

// TestImporter_SampleShop_WithSnapshot 用合成的商店三張表跑完整匯入流程：
// Demo_Shop 整筆嵌入 Demo_ShopLabel、並以 GroupId 欄位引用多筆 Demo_ShopGoods。
func TestImporter_SampleShop_WithSnapshot(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	sheets := loadSampleSheets(t, "Demo_Shop", "Demo_ShopGoods", "Demo_ShopLabel")

	parser := source.NewParser()
	entities := make([]schema.Entity, 0, len(sheets))
	for _, sheet := range sheets {
		entity, err := parser.Parse(sheet)
		if err != nil {
			t.Fatalf("parse %s: %v", sheet.Name, err)
		}
		entities = append(entities, entity)
	}
	resolved, err := resolver.New(entities, resolver.WithWarnHandler(func(string) {})).Resolve()
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}

	protoPath := filepath.Join(testdataDir, "sample.proto")
	imp := New(db, "", WithSnapshot(SnapshotConfig{
		ProtoFilePath:    protoPath,
		Namespace:        "test",
		EmbeddedEntities: resolver.FindEmbeddedEntities(resolved),
		Entities:         resolved,
	}))
	createTables(t, db, imp, resolved)

	importAll := func() {
		t.Helper()
		if err := imp.EnsureSnapshotTables(); err != nil {
			t.Fatalf("EnsureSnapshotTables: %v", err)
		}
		referenced := resolver.FindReferencedEntities(resolved)
		for _, sheet := range sheets {
			if referenced[sheet.Name] {
				if err := imp.IndexSheetData(sheet); err != nil {
					t.Fatalf("IndexSheetData %s: %v", sheet.Name, err)
				}
			}
		}
		err := db.Transaction(func(tx *gorm.DB) error {
			for _, sheet := range sheets {
				if err := imp.ImportWithTx(tx, sheet); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("import: %v", err)
		}
	}
	importAll()

	// 所有表（包含被嵌入的 Demo_ShopLabel）都寫入 DB
	for table, want := range map[string]int64{"demo_shop": 3, "demo_shop_goods": 4, "demo_shop_label": 3} {
		var count int64
		db.Table(table).Count(&count)
		if count != want {
			t.Errorf("%s has %d rows, want %d", table, count, want)
		}
	}

	// skip 欄位不寫入、保留字欄位加後綴
	var goods []map[string]any
	db.Table("demo_shop_goods").Order("id").Find(&goods)
	if len(goods) == 0 || goods[0]["weight_field"] != "10" {
		t.Errorf("first goods row = %v, want weight_field=10", goods)
	}

	// 被嵌入的 Demo_ShopLabel 不產生獨立快照
	var snapshots []models.Snapshot
	db.Order("name").Find(&snapshots)
	gotNames := make([]string, 0, len(snapshots))
	for _, s := range snapshots {
		gotNames = append(gotNames, s.Name)
	}
	if strings.Join(gotNames, ",") != "Demo_Shop,Demo_ShopGoods" {
		t.Fatalf("snapshots = %v, want [Demo_Shop Demo_ShopGoods]", gotNames)
	}

	// 解開 Demo_Shop 的快照，確認嵌入與欄位引用的資料都在
	ser, err := serializer.NewProtoSerializer(protoPath)
	if err != nil {
		t.Fatal(err)
	}
	shops, err := ser.Deserialize("DemoShopList", "DemoShop", snapshots[0].Data)
	if err != nil {
		t.Fatalf("Deserialize: %v", err)
	}
	if len(shops) != 3 {
		t.Fatalf("got %d shops, want 3", len(shops))
	}
	label, _ := shops[0]["label"].(map[string]any)
	if label["label_text"] != int32(7001) {
		t.Errorf("shop 1 label = %v, want embedded label_text=7001", shops[0]["label"])
	}
	shopGoods, _ := shops[0]["goods_group_id"].([]any)
	if len(shopGoods) != 2 {
		t.Errorf("shop 1 goods = %v, want 2 goods of group 11", shops[0]["goods_group_id"])
	}

	// 內容不變時重新匯入：checksum 相同，不新增版本
	var versionsBefore int64
	db.Model(&models.Version{}).Count(&versionsBefore)
	if versionsBefore != 2 {
		t.Errorf("versions = %d, want 2", versionsBefore)
	}
	imp.importMode = ImportModeUpsert
	importAll()
	var versionsAfter int64
	db.Model(&models.Version{}).Count(&versionsAfter)
	if versionsAfter != versionsBefore {
		t.Errorf("re-import with same data created new versions: %d -> %d", versionsBefore, versionsAfter)
	}
}

func TestImporter_ImportSheet_WithCustomLayout(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	if err := db.Exec(`CREATE TABLE items (id INTEGER PRIMARY KEY, name TEXT)`).Error; err != nil {
		t.Fatal(err)
	}

	// 沒有註解列、輸出類型用 "-" 表示略過
	layout := source.Layout{NameRow: 0, TypeRow: 1, MetaRow: 2, OutputRow: 3, DataRow: 4, SkipMarker: "-"}
	sheet := source.Sheet{
		Name: "items",
		Rows: [][]string{
			{"id", "name", "secret"},
			{"int", "string", "string"},
			{"", "", ""},
			{"", "", "-"},
			{"1", "Sword", "x"},
			{"2", "Shield", "y"},
		},
	}

	imp := New(db, "", WithLayout(layout))
	if err := imp.Import(sheet); err != nil {
		t.Fatalf("Import() error = %v", err)
	}

	var records []map[string]any
	db.Table("items").Order("id").Find(&records)
	if len(records) != 2 || records[1]["name"] != "Shield" {
		t.Errorf("records = %v, want 2 rows with second name=Shield", records)
	}

	// 預設版面下同一份資料列數不足以構成表頭 + 資料，應回報錯誤而非誤讀
	if err := New(db, "").Import(source.Sheet{Name: "items", Rows: sheet.Rows[:5]}); err == nil {
		t.Error("Import() with default layout should fail on a sheet that has no data rows")
	}
}

// TestImporter_Upsert_UpdatesExistingRows 確認 upsert 模式會更新既有資料列而不是報錯
func TestImporter_Upsert_UpdatesExistingRows(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	if err := db.Exec(`CREATE TABLE items (id INTEGER PRIMARY KEY, name TEXT, price INTEGER)`).Error; err != nil {
		t.Fatal(err)
	}

	sheet := func(price string) source.Sheet {
		return source.Sheet{
			Name: "items",
			Rows: [][]string{
				{"Id", "名稱", "價格"},
				{"Id", "Name", "Price"},
				{"int", "string", "int"},
				{"", "", ""},
				{"", "", ""},
				{"1", "Sword", price},
				{"2", "Shield", "50"},
			},
		}
	}

	imp := New(db, "", WithImportMode(ImportModeUpsert))
	if err := imp.Import(sheet("100")); err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	if err := imp.Import(sheet("120")); err != nil {
		t.Fatalf("second upsert: %v", err)
	}

	var records []map[string]any
	db.Table("items").Order("id").Find(&records)
	if len(records) != 2 {
		t.Fatalf("got %d rows, want 2", len(records))
	}
	if fmt.Sprint(records[0]["price"]) != "120" {
		t.Errorf("price after upsert = %v, want 120", records[0]["price"])
	}
}
