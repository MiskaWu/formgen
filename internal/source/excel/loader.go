package excel

import (
	"bytes"
	"fmt"

	"github.com/miskawu/formgen/internal/source"

	"github.com/xuri/excelize/v2"
)

// Load 從 byte[] 讀取 Excel 文件，回傳所有工作表
func (l *loader) Load(data []byte) ([]source.Sheet, error) {
	// 從 byte[] 打開 Excel 文件
	reader := bytes.NewReader(data)
	file, err := excelize.OpenReader(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to open excel file: %w", err)
	}
	defer file.Close()

	// 獲取所有工作表名稱
	sheetNames := file.GetSheetList()
	if len(sheetNames) == 0 {
		return nil, fmt.Errorf("no sheets found in excel file")
	}

	// 讀取所有工作表的數據
	sheets := make([]source.Sheet, 0, len(sheetNames))
	for _, sheetName := range sheetNames {
		rows, err := file.GetRows(sheetName)
		if err != nil {
			return nil, fmt.Errorf("failed to read rows from sheet %s: %w", sheetName, err)
		}
		sheets = append(sheets, source.Sheet{
			Name: sheetName,
			Rows: rows,
		})
	}

	return sheets, nil
}
