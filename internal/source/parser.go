package source

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/miskawu/formgen/internal/schema"
)

// ArrayFieldPattern 匹配 ArrayName[index].FieldName 格式
// 例如: Loot[0].ItemId, Reward[1].Count
// 導出供 Importer 重用
var ArrayFieldPattern = regexp.MustCompile(`^(\w+)\[(\d+)\]\.(\w+)$`)

// arrayFieldInfo 儲存解析後的陣列欄位資訊
type arrayFieldInfo struct {
	arrayName string // 陣列名稱，如 "Loot"
	index     int    // 陣列索引，如 0, 1, 2
	fieldName string // 欄位名稱，如 "ItemId"
	rawType   string // 欄位類型
	isArray   bool   // 欄位本身是否為陣列
	isRef     bool   // 是否為參考類型
	refExpr   string // 參考表達式
	colIdx    int    // 原始欄位索引（用於保持順序）
}

type parser struct {
	layout Layout
}

// Parse 將 Sheet 解析為 schema.Entity
// 欄位約定（列位置由 Layout 決定，以下為預設版面）：
//   - row 0: 顯示名稱／註解（忽略）
//   - row 1: 欄位名稱（支援 ArrayName[index].FieldName 格式）
//   - row 2: 值域類型（支援 * 參考修飾符，如 *entry, *[]shortint）
//   - row 3: metadata，參考表達式（如 ref:ItemInfo, ref:Table.Field）或註解
//   - row 4: 輸出類型（填 skip 的欄位略過）
//   - row 5+: 資料行（忽略）
func (p *parser) Parse(sheet Sheet) (schema.Entity, error) {
	rows := sheet.Rows
	layout := p.layout

	if err := layout.Validate(); err != nil {
		return schema.Entity{}, err
	}
	if len(rows) < layout.HeaderRows() {
		return schema.Entity{}, fmt.Errorf("invalid sheet data: need at least %d rows", layout.HeaderRows())
	}

	// 取表頭中最寬的一列作為欄位數
	maxCols := max(len(rows[layout.NameRow]), len(rows[layout.TypeRow]), len(rows[layout.OutputRow]))

	// 收集普通欄位和陣列欄位
	var columns []schema.Column
	arrayFields := make(map[string][]arrayFieldInfo) // arrayName -> []arrayFieldInfo
	arrayOrder := make(map[string]int)               // arrayName -> 首次出現的 colIdx（用於保持順序）

	for colIdx := 0; colIdx < maxCols; colIdx++ {
		// 檢查是否需要跳過這個欄位
		if layout.IsSkipped(rows, colIdx) {
			continue
		}

		// 獲取欄位名稱
		name := Cell(rows, layout.NameRow, colIdx)
		if name == "" {
			continue // 跳過空欄位
		}

		// 獲取欄位類型
		rawType := Cell(rows, layout.TypeRow, colIdx)
		var isArray bool

		// 檢查是否為參考類型（帶 * 修飾符）
		var isReference bool
		var refExpr string
		if strings.HasPrefix(rawType, "*") {
			isReference = true
			rawType = strings.TrimPrefix(rawType, "*")

			// 從 metadata 列讀取參考表達式
			refExpr = Cell(rows, layout.MetaRow, colIdx)
		}

		// 檢查是否為陣列類型
		if strings.Contains(rawType, "[]") {
			isArray = true
			// 移除 [] 符號獲取基礎類型
			rawType = strings.ReplaceAll(rawType, "[]", "")
			rawType = strings.TrimSpace(rawType)
		}

		// 檢查是否為 ArrayName[index].FieldName 格式
		if matches := ArrayFieldPattern.FindStringSubmatch(name); matches != nil {
			arrayName := matches[1]
			index, _ := strconv.Atoi(matches[2])
			fieldName := matches[3]

			// 記錄首次出現的順序
			if _, exists := arrayOrder[arrayName]; !exists {
				arrayOrder[arrayName] = colIdx
			}

			arrayFields[arrayName] = append(arrayFields[arrayName], arrayFieldInfo{
				arrayName: arrayName,
				index:     index,
				fieldName: fieldName,
				rawType:   rawType,
				isArray:   isArray,
				isRef:     isReference,
				refExpr:   refExpr,
				colIdx:    colIdx,
			})
			continue
		}

		// 普通欄位
		columns = append(columns, schema.Column{
			Name:        name,
			RawType:     rawType,
			IsArray:     isArray,
			IsReference: isReference,
			RefExpr:     refExpr,
		})
	}

	// 處理陣列欄位
	if len(arrayFields) > 0 {
		arrayColumns, err := p.processArrayFields(arrayFields, arrayOrder)
		if err != nil {
			return schema.Entity{}, fmt.Errorf("failed to process array fields: %w", err)
		}
		columns = append(columns, arrayColumns...)
	}

	// 構建 Entity
	entity := schema.Entity{
		Name:    sheet.Name,
		Columns: columns,
	}

	return entity, nil
}

// processArrayFields 處理陣列欄位，將 ArrayName[index].FieldName 格式轉換為帶 NestedColumns 的 Column
func (p *parser) processArrayFields(arrayFields map[string][]arrayFieldInfo, arrayOrder map[string]int) ([]schema.Column, error) {
	// 按首次出現順序排序陣列名稱
	arrayNames := make([]string, 0, len(arrayFields))
	for name := range arrayFields {
		arrayNames = append(arrayNames, name)
	}
	sort.Slice(arrayNames, func(i, j int) bool {
		return arrayOrder[arrayNames[i]] < arrayOrder[arrayNames[j]]
	})

	var columns []schema.Column

	for _, arrayName := range arrayNames {
		fields := arrayFields[arrayName]

		// 驗證同名欄位的類型一致性並收集嵌套欄位資訊
		nestedFields, err := p.validateAndCollectNestedFields(arrayName, fields)
		if err != nil {
			return nil, err
		}

		// 生成陣列欄位
		columns = append(columns, schema.Column{
			Name:          arrayName,
			RawType:       arrayName + "Struct", // 自動生成類型名稱
			IsArray:       true,
			IsStruct:      true,
			NestedColumns: nestedFields,
		})
	}

	return columns, nil
}

// validateAndCollectNestedFields 驗證陣列欄位的類型一致性並收集嵌套欄位
func (p *parser) validateAndCollectNestedFields(arrayName string, fields []arrayFieldInfo) ([]schema.Column, error) {
	// 按 fieldName 分組，記錄每個欄位的類型資訊
	fieldTypes := make(map[string]arrayFieldInfo) // fieldName -> 第一次出現的類型資訊
	fieldOrder := make(map[string]int)            // fieldName -> 首次出現的順序

	for _, field := range fields {
		if existing, exists := fieldTypes[field.fieldName]; exists {
			// 驗證類型一致性
			if existing.rawType != field.rawType {
				return nil, fmt.Errorf("type mismatch for %s[].%s: index %d has type %q, but index %d has type %q",
					arrayName, field.fieldName,
					fieldOrder[field.fieldName], existing.rawType,
					field.index, field.rawType)
			}
			if existing.isArray != field.isArray {
				return nil, fmt.Errorf("array modifier mismatch for %s[].%s: inconsistent [] modifier between indices",
					arrayName, field.fieldName)
			}
			if existing.isRef != field.isRef {
				return nil, fmt.Errorf("reference modifier mismatch for %s[].%s: inconsistent * modifier between indices",
					arrayName, field.fieldName)
			}
		} else {
			fieldTypes[field.fieldName] = field
			fieldOrder[field.fieldName] = field.index
		}
	}

	// 按首次出現順序生成嵌套欄位
	type fieldWithOrder struct {
		name  string
		order int
	}
	orderedFields := make([]fieldWithOrder, 0, len(fieldTypes))
	for name := range fieldTypes {
		// 找到該欄位在 index 0 的原始 colIdx（用於排序）
		minColIdx := int(^uint(0) >> 1) // max int
		for _, f := range fields {
			if f.fieldName == name && f.colIdx < minColIdx {
				minColIdx = f.colIdx
			}
		}
		orderedFields = append(orderedFields, fieldWithOrder{name: name, order: minColIdx})
	}
	sort.Slice(orderedFields, func(i, j int) bool {
		return orderedFields[i].order < orderedFields[j].order
	})

	// 生成嵌套欄位
	nestedColumns := make([]schema.Column, 0, len(orderedFields))
	for _, of := range orderedFields {
		info := fieldTypes[of.name]
		nestedColumns = append(nestedColumns, schema.Column{
			Name:        info.fieldName,
			RawType:     info.rawType,
			IsArray:     info.isArray,
			IsReference: info.isRef,
			RefExpr:     info.refExpr,
		})
	}

	return nestedColumns, nil
}
