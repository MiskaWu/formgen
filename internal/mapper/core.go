package mapper

import "maps"

// defaultTypeMapper 預設類型映射器
type defaultTypeMapper struct {
	mappings map[string]string
}

// Map 執行類型映射
func (m *defaultTypeMapper) Map(sourceType string) (string, bool) {
	protoType, ok := m.mappings[sourceType]
	return protoType, ok
}

// AddMapping 添加自定義映射
func (m *defaultTypeMapper) AddMapping(sourceType, protoType string) {
	m.mappings[sourceType] = protoType
}

// AddMappings 批量添加映射
func (m *defaultTypeMapper) AddMappings(mappings map[string]string) {
	maps.Copy(m.mappings, mappings)
}
