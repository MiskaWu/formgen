package excel

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestLoader_Load(t *testing.T) {
	// 創建一個測試用的 Excel 文件
	file := excelize.NewFile()
	defer file.Close()

	sheetName := "TestSheet"
	file.SetSheetName("Sheet1", sheetName)

	// 填充測試數據
	testData := [][]any{
		{"表頭說明"},
		{"id", "name", "age", "items"},
		{"int", "string", "int", "string[]"},
		{"資料說明"},
		{"", "", "", ""},
	}

	for rowIdx, row := range testData {
		for colIdx, value := range row {
			cell := fmt.Sprintf("%c%d", 'A'+colIdx, rowIdx+1)
			file.SetCellValue(sheetName, cell, value)
		}
	}

	// 將 Excel 文件轉換為 byte[]
	buf, err := file.WriteToBuffer()
	if err != nil {
		t.Fatalf("failed to write excel to buffer: %v", err)
	}

	// 測試 Load 方法
	loader := NewLoader()
	sheets, err := loader.Load(buf.Bytes())
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// 驗證結果
	if len(sheets) != 1 {
		t.Errorf("expected 1 sheet, got %d", len(sheets))
	}

	sheet := sheets[0]
	if sheet.Name != sheetName {
		t.Fatalf("expected sheet name %s, got %s", sheetName, sheet.Name)
	}

	// Excel 會自動忽略完全空的行，所以實際行數可能少於 testData
	if len(sheet.Rows) < 1 {
		t.Errorf("expected at least 1 row, got %d", len(sheet.Rows))
	}

	// 打印轉換內容，方便查看
	t.Logf("=== Loaded Excel Content ===")
	for _, s := range sheets {
		t.Logf("\nSheet: %s", s.Name)
		for i, row := range s.Rows {
			t.Logf("  Row %d: %v", i, row)
		}
	}
}

func TestLoader_Load_MultipleSheets(t *testing.T) {
	// 創建包含多個工作表的 Excel 文件
	file := excelize.NewFile()
	defer file.Close()

	// 第一個工作表
	sheet1 := "Users"
	file.SetSheetName("Sheet1", sheet1)
	userData := [][]any{
		{"用戶表"},
		{"user_id", "username", "email"},
		{"int", "string", "string"},
		{"說明"},
		{"", "", ""},
	}

	for rowIdx, row := range userData {
		for colIdx, value := range row {
			cell := fmt.Sprintf("%c%d", 'A'+colIdx, rowIdx+1)
			file.SetCellValue(sheet1, cell, value)
		}
	}

	// 第二個工作表
	sheet2 := "Products"
	_, err := file.NewSheet(sheet2)
	if err != nil {
		t.Fatalf("failed to create sheet: %v", err)
	}

	productData := [][]any{
		{"產品表"},
		{"product_id", "name", "price", "tags"},
		{"int", "string", "float", "string[]"},
		{"說明"},
		{"", "", "", ""},
	}

	for rowIdx, row := range productData {
		for colIdx, value := range row {
			cell := fmt.Sprintf("%c%d", 'A'+colIdx, rowIdx+1)
			file.SetCellValue(sheet2, cell, value)
		}
	}

	// 將 Excel 文件轉換為 byte[]
	buf, err := file.WriteToBuffer()
	if err != nil {
		t.Fatalf("failed to write excel to buffer: %v", err)
	}

	// 測試 Load 方法
	loader := NewLoader()
	sheets, err := loader.Load(buf.Bytes())
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// 驗證結果
	if len(sheets) != 2 {
		t.Errorf("expected 2 sheets, got %d", len(sheets))
	}

	// 打印轉換內容
	t.Logf("\n=== Loaded Excel Content (Multiple Sheets) ===")
	for _, sheet := range sheets {
		t.Logf("\n📄 Sheet: %s", sheet.Name)
		t.Logf("   Total rows: %d", len(sheet.Rows))
		for i, row := range sheet.Rows {
			t.Logf("   Row %d: %v", i, row)
		}
	}
}

func TestLoader_Load_EmptyFile(t *testing.T) {
	// 創建空的 Excel 文件（沒有數據）
	file := excelize.NewFile()
	defer file.Close()

	buf, err := file.WriteToBuffer()
	if err != nil {
		t.Fatalf("failed to write excel to buffer: %v", err)
	}

	loader := NewLoader()
	sheets, err := loader.Load(buf.Bytes())
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// 應該至少有一個默認工作表（Sheet1）
	if len(sheets) == 0 {
		t.Error("expected at least 1 sheet, got 0")
	}

	t.Logf("\n=== Empty Excel Content ===")
	for _, sheet := range sheets {
		t.Logf("Sheet: %s, Rows: %d", sheet.Name, len(sheet.Rows))
	}
}

func TestLoader_Load_InvalidData(t *testing.T) {
	loader := NewLoader()
	// 測試無效的 byte 數據
	invalidData := []byte("this is not an excel file")
	_, err := loader.Load(invalidData)
	if err == nil {
		t.Error("expected error for invalid data, got nil")
	}

	t.Logf("Error (expected): %v", err)
}

// 測試輔助函數：創建帶有實際數據的 Excel 文件
func createSampleExcelWithData(t *testing.T) *bytes.Buffer {
	file := excelize.NewFile()
	defer file.Close()

	sheetName := "GameConfig"
	file.SetSheetName("Sheet1", sheetName)

	// 模擬遊戲配置表
	configData := [][]any{
		{"遊戲角色配置表"},                                           // Row 0: 說明
		{"id", "name", "level", "skills", "attributes"},       // Row 1: 欄位名稱
		{"int", "string", "int", "string[]", "float[]"},       // Row 2: 欄位類型
		{"角色ID", "角色名稱", "等級", "技能列表", "屬性值"},                 // Row 3: 中文說明
		{"", "", "", "", ""},                                  // Row 4: skip 標記
		{"1", "Warrior", "10", "Slash|Block", "100.5|50.3"},   // Row 5: 數據
		{"2", "Mage", "8", "Fireball|IceStorm", "80.0|120.5"}, // Row 6: 數據
	}

	for rowIdx, row := range configData {
		for colIdx, value := range row {
			cell := fmt.Sprintf("%c%d", 'A'+colIdx, rowIdx+1)
			file.SetCellValue(sheetName, cell, value)
		}
	}

	buf, err := file.WriteToBuffer()
	if err != nil {
		t.Fatalf("failed to write excel to buffer: %v", err)
	}

	return buf
}

func TestLoader_Load_WithSampleData(t *testing.T) {
	buf := createSampleExcelWithData(t)

	loader := NewLoader()
	sheets, err := loader.Load(buf.Bytes())
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// 打印詳細的轉換內容
	separator := strings.Repeat("=", 60)
	t.Logf("\n%s", separator)
	t.Logf("=== 🎮 Game Config Sample Excel Content ===")
	t.Logf("%s", separator)

	for _, sheet := range sheets {
		t.Logf("\n📋 Sheet Name: %s", sheet.Name)
		t.Logf("📊 Total Rows: %d\n", len(sheet.Rows))

		for i, row := range sheet.Rows {
			rowLabel := ""
			switch i {
			case 0:
				rowLabel = "說明"
			case 1:
				rowLabel = "欄位名稱"
			case 2:
				rowLabel = "欄位類型"
			case 3:
				rowLabel = "中文說明"
			case 4:
				rowLabel = "Skip標記"
			default:
				rowLabel = "資料"
			}

			t.Logf("   Row %d (%s): %v", i, rowLabel, row)
		}
	}

	t.Logf("\n%s", separator)
}

func TestLoader_Load_SampleFile(t *testing.T) {
	// 讀取合成範例檔（由 internal/sampledata 產生）
	filePath := "../../../testdata/Demo_Category.xlsx"
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("無法讀取文件 %s: %v", filePath, err)
	}

	loader := NewLoader()
	sheets, err := loader.Load(data)
	if err != nil {
		t.Fatalf("載入失敗: %v", err)
	}

	// 範例檔有兩個工作表：資料表本身與欄位說明，順序需保持
	if len(sheets) != 2 || sheets[0].Name != "Demo_Category" {
		t.Fatalf("unexpected sheets: %d, first = %q", len(sheets), sheets[0].Name)
	}
	if got := sheets[0].Rows[1]; len(got) < 1 || got[0] != "Id" {
		t.Errorf("field name row = %v, want to start with Id", got)
	}

	// 打印詳細的轉換內容
	separator := strings.Repeat("=", 70)
	t.Logf("\n%s", separator)
	t.Logf("===  📄 Demo_Category.xlsx 文件轉換內容  ===")
	t.Logf("%s\n", separator)

	// 只顯示第一張工作表
	if len(sheets) > 0 {
		sheet := sheets[0]
		t.Logf("📋 工作表名稱: %s", sheet.Name)
		t.Logf("📊 總行數: %d", len(sheet.Rows))
		t.Logf("%s", strings.Repeat("-", 70))

		for i, row := range sheet.Rows {
			rowLabel := ""
			switch i {
			case 0:
				rowLabel = "Row 0 (表頭說明)"
			case 1:
				rowLabel = "Row 1 (欄位名稱)"
			case 2:
				rowLabel = "Row 2 (欄位類型)"
			case 3:
				rowLabel = "Row 3 (中文說明)"
			case 4:
				rowLabel = "Row 4 (Skip標記)"
			default:
				rowLabel = fmt.Sprintf("Row %d (資料)", i)
			}

			// 格式化顯示每一列
			if len(row) == 0 {
				t.Logf("   %-20s: [空行]", rowLabel)
			} else {
				t.Logf("   %-20s: %v", rowLabel, row)
			}
		}

		t.Logf("\n%s\n", strings.Repeat("-", 70))
	}

	t.Logf("%s", separator)
	t.Logf("✅ 總共載入了 %d 個工作表（僅顯示第一張）", len(sheets))
	t.Logf("%s\n", separator)
}
