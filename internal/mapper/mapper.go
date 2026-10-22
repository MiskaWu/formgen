package mapper

import "maps"

// TypeMapper 定義類型映射接口
type TypeMapper interface {
	Map(sourceType string) (protoType string, ok bool)
}

func NewTypeMapper(mappings map[string]string) *defaultTypeMapper {
	// 複製 map 避免修改原始映射
	copiedMappings := maps.Clone(mappings)
	return &defaultTypeMapper{
		mappings: copiedMappings,
	}
}
