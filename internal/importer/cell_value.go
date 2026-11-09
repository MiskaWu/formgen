package importer

import (
	"fmt"
	"strconv"
	"strings"
)

// cellConverter 把儲存格文字轉成寫入資料庫用的 Go 值。
type cellConverter func(cell string) (any, error)

// dbCellConverters 以標準化類型查表，決定寫入資料庫前要不要先轉換。
//
// 沒列在這裡的類型照字串寫入：資料庫能把 '100' 轉成數字，但不一定認得 'true'。
// 要交給資料庫也會失敗的類型，在這裡加一列即可。
var dbCellConverters = map[string]cellConverter{
	"bool": convertBoolCell,
}

// convertBoolCell 回傳 Go 的 bool，由各資料庫的驅動編碼成自己的寫法
// （MySQL、SQLite 是 1/0，PostgreSQL 是 true/false）。
// 接受的寫法用 strconv.ParseBool，與 protobuf 快照的序列化器一致，
// 同一格資料不會發生資料庫和快照解讀不同的情況。
func convertBoolCell(cell string) (any, error) {
	v, err := strconv.ParseBool(cell)
	if err != nil {
		return nil, fmt.Errorf("invalid bool value %q", cell)
	}
	return v, nil
}

// dbConverterFor 依表頭類型列的寫法找出該欄的轉換函式，不需要轉換時回傳 nil。
// 陣列欄位在資料庫裡是 JSON，儲存格本身就是 JSON 文字，不轉換。
func (s *Importer) dbConverterFor(rawType string) cellConverter {
	baseType := strings.TrimPrefix(strings.TrimSpace(rawType), "*")
	if strings.Contains(baseType, "[]") {
		return nil
	}
	if mapped, ok := s.typeMapper.Map(baseType); ok {
		baseType = mapped
	}
	return dbCellConverters[baseType]
}
