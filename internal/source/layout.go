package source

import (
	"fmt"
	"strings"
)

// DefaultSkipMarker 是輸出類型列中代表「略過此欄」的預設標記。
const DefaultSkipMarker = "skip"

// Layout 描述工作表表頭各列所在的位置（0-based 列索引）。
//
// 預設版面（見 DefaultLayout）：
//
//	row 0: 顯示名稱／註解（formgen 不讀）
//	row 1: 欄位名稱（支援 Array[index].Field 結構陣列寫法）
//	row 2: 值域類型（支援 * 參考修飾符與 [] 陣列修飾符）
//	row 3: metadata（參考表達式，如 ref:Item、ref:Table.Field）
//	row 4: 輸出類型（填 SkipMarker 的欄位會被略過）
//	row 5+: 資料列
type Layout struct {
	NameRow    int    // 欄位名稱列
	TypeRow    int    // 值域類型列
	MetaRow    int    // metadata（參考表達式）列
	OutputRow  int    // 輸出類型列
	DataRow    int    // 第一筆資料所在列
	SkipMarker string // 輸出類型列中代表略過的標記
}

// DefaultLayout 返回 formgen 預設的表頭版面。
func DefaultLayout() Layout {
	return Layout{
		NameRow:    1,
		TypeRow:    2,
		MetaRow:    3,
		OutputRow:  4,
		DataRow:    5,
		SkipMarker: DefaultSkipMarker,
	}
}

// Validate 檢查版面設定是否合理：列索引不可為負、表頭列不可重複，
// 且資料列必須在所有表頭列之後。
func (l Layout) Validate() error {
	header := map[string]int{
		"name":   l.NameRow,
		"type":   l.TypeRow,
		"meta":   l.MetaRow,
		"output": l.OutputRow,
	}
	seen := make(map[int]string, len(header))
	for _, key := range []string{"name", "type", "meta", "output"} {
		row := header[key]
		if row < 0 {
			return fmt.Errorf("layout: %s row must not be negative, got %d", key, row)
		}
		if other, dup := seen[row]; dup {
			return fmt.Errorf("layout: %s row and %s row both point to row %d", other, key, row)
		}
		seen[row] = key
		if l.DataRow <= row {
			return fmt.Errorf("layout: data row (%d) must come after %s row (%d)", l.DataRow, key, row)
		}
	}
	if strings.TrimSpace(l.SkipMarker) == "" {
		return fmt.Errorf("layout: skip marker must not be empty")
	}
	return nil
}

// HeaderRows 返回表頭至少需要的列數（最後一個表頭列的索引 + 1）。
func (l Layout) HeaderRows() int {
	return max(l.NameRow, l.TypeRow, l.MetaRow, l.OutputRow) + 1
}

// IsSkipped 判斷第 col 欄是否在輸出類型列被標記為略過。
func (l Layout) IsSkipped(rows [][]string, col int) bool {
	return Cell(rows, l.OutputRow, col) == strings.TrimSpace(l.SkipMarker)
}

// Cell 安全地取出 rows[row][col] 並去除前後空白；超出範圍時返回空字串。
func Cell(rows [][]string, row, col int) string {
	if row < 0 || row >= len(rows) || col < 0 || col >= len(rows[row]) {
		return ""
	}
	return strings.TrimSpace(rows[row][col])
}
