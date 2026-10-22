package importer

import (
	"encoding/json"
	"strconv"

	"github.com/miskawu/formgen/internal/source"
	"github.com/miskawu/formgen/internal/strcase"
)

// structFieldCollector 收集並聚合結構陣列欄位
// 用於將 Loot[0].ItemId, Loot[0].Amount, Loot[1].ItemId 等格式
// 聚合為單一的 JSON 陣列
// 使用 slice 保持插入順序，避免 map 遍歷的不確定性
type structFieldCollector struct {
	// 保持 arrayName 的插入順序
	arrayOrder []string
	arraySet   map[string]bool

	// arrayName -> index -> fieldName -> value
	data map[string]map[int]map[string]any
	// arrayName -> maxIndex（用於確定陣列大小）
	maxIndices map[string]int

	// arrayName -> fieldName 的插入順序
	fieldOrder map[string][]string
	fieldSet   map[string]map[string]bool
}

// newStructFieldCollector 創建新的結構陣列收集器
func newStructFieldCollector() *structFieldCollector {
	return &structFieldCollector{
		arrayOrder: make([]string, 0),
		arraySet:   make(map[string]bool),
		data:       make(map[string]map[int]map[string]any),
		maxIndices: make(map[string]int),
		fieldOrder: make(map[string][]string),
		fieldSet:   make(map[string]map[string]bool),
	}
}

// Add 添加一個嵌套欄位的值
func (c *structFieldCollector) Add(arrayName string, index int, fieldName string, value any) {
	// 記錄 arrayName 的首次出現順序
	if !c.arraySet[arrayName] {
		c.arrayOrder = append(c.arrayOrder, arrayName)
		c.arraySet[arrayName] = true
	}

	// 記錄 fieldName 的首次出現順序
	if c.fieldSet[arrayName] == nil {
		c.fieldSet[arrayName] = make(map[string]bool)
	}
	if !c.fieldSet[arrayName][fieldName] {
		c.fieldOrder[arrayName] = append(c.fieldOrder[arrayName], fieldName)
		c.fieldSet[arrayName][fieldName] = true
	}

	// 存儲資料
	if c.data[arrayName] == nil {
		c.data[arrayName] = make(map[int]map[string]any)
	}
	if c.data[arrayName][index] == nil {
		c.data[arrayName][index] = make(map[string]any)
	}
	c.data[arrayName][index][fieldName] = value

	if index > c.maxIndices[arrayName] {
		c.maxIndices[arrayName] = index
	}
}

// ToJSON 將收集的資料轉換為 JSON 字串
// 回傳: map[arrayName]jsonString
// 使用 orderedMap 確保 JSON 輸出順序與插入順序一致
func (c *structFieldCollector) ToJSON() (map[string]string, error) {
	result := make(map[string]string)

	// 按插入順序遍歷 arrayName
	for _, arrayName := range c.arrayOrder {
		indexMap := c.data[arrayName]
		maxIdx := c.maxIndices[arrayName]
		arr := make([]orderedMap, 0, maxIdx+1)

		// 按 index 順序構建陣列
		for i := 0; i <= maxIdx; i++ {
			if fields, ok := indexMap[i]; ok && len(fields) > 0 {
				// 使用 orderedMap 保持欄位順序
				om := newOrderedMap()
				for _, fieldName := range c.fieldOrder[arrayName] {
					if v, exists := fields[fieldName]; exists {
						om.Set(strcase.ToSnakeCase(fieldName), v)
					}
				}
				arr = append(arr, *om)
			}
		}

		if len(arr) == 0 {
			continue
		}

		// 序列化為 JSON
		jsonBytes, err := json.Marshal(arr)
		if err != nil {
			return nil, err
		}
		result[arrayName] = string(jsonBytes)
	}

	return result, nil
}

// ToSlice 將收集的資料轉換為 []map[string]any 格式（用於 protobuf snapshot）
// 回傳: map[arrayName][]map[string]any
// 注意：返回的 map 內部欄位順序由 proto_serializer 的 setRepeatedMessageField 保證
func (c *structFieldCollector) ToSlice() map[string][]map[string]any {
	result := make(map[string][]map[string]any)

	// 按插入順序遍歷 arrayName
	for _, arrayName := range c.arrayOrder {
		indexMap := c.data[arrayName]
		maxIdx := c.maxIndices[arrayName]
		arr := make([]map[string]any, 0, maxIdx+1)

		// 按 index 順序構建陣列
		for i := 0; i <= maxIdx; i++ {
			if fields, ok := indexMap[i]; ok && len(fields) > 0 {
				// 將欄位名轉為 snake_case（與 proto 欄位名一致）
				snakeFields := make(map[string]any)
				for k, v := range fields {
					snakeFields[strcase.ToSnakeCase(k)] = v
				}
				arr = append(arr, snakeFields)
			}
		}

		if len(arr) > 0 {
			result[arrayName] = arr
		}
	}

	return result
}

// orderedMap 保持 key 插入順序的 map
// 用於確保 JSON 序列化時欄位順序一致
type orderedMap struct {
	keys   []string
	values map[string]any
}

func newOrderedMap() *orderedMap {
	return &orderedMap{
		keys:   make([]string, 0),
		values: make(map[string]any),
	}
}

func (m *orderedMap) Set(key string, value any) {
	if _, exists := m.values[key]; !exists {
		m.keys = append(m.keys, key)
	}
	m.values[key] = value
}

// MarshalJSON 實現 json.Marshaler 介面，按 key 順序輸出
func (m orderedMap) MarshalJSON() ([]byte, error) {
	var buf []byte
	buf = append(buf, '{')
	for i, key := range m.keys {
		if i > 0 {
			buf = append(buf, ',')
		}
		// 序列化 key
		keyBytes, err := json.Marshal(key)
		if err != nil {
			return nil, err
		}
		buf = append(buf, keyBytes...)
		buf = append(buf, ':')
		// 序列化 value
		valBytes, err := json.Marshal(m.values[key])
		if err != nil {
			return nil, err
		}
		buf = append(buf, valBytes...)
	}
	buf = append(buf, '}')
	return buf, nil
}

// arrayFieldInfo 儲存解析後的陣列欄位資訊
type arrayFieldInfo struct {
	arrayName string // 陣列名稱，如 "Loot"
	index     int    // 陣列索引，如 0, 1, 2
	fieldName string // 欄位名稱，如 "ItemId"
}

// parseStructField 檢查欄位名是否為 ArrayName[index].FieldName 格式
// 回傳: arrayName, index, fieldName, isMatch
func parseStructField(fieldName string) (string, int, string, bool) {
	matches := source.ArrayFieldPattern.FindStringSubmatch(fieldName)
	if matches == nil {
		return "", 0, "", false
	}

	index, _ := strconv.Atoi(matches[2])
	return matches[1], index, matches[3], true
}
