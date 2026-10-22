package yaml

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/miskawu/formgen/internal/schema"
)

// column 代表 YAML 中的欄位定義
type column struct {
	Name          string   `yaml:"name"`
	Type          string   `yaml:"type"`
	IsArray       bool     `yaml:"isArray"`
	IsStruct      bool     `yaml:"isStruct,omitempty"`      // 是否為結構陣列欄位
	Ref           string   `yaml:"ref,omitempty"`           // 參考表達式（如 "ItemInfo" 或 "Table.Field"）
	NestedColumns []column `yaml:"nestedColumns,omitempty"` // 嵌套欄位
}

// entity 代表 YAML 中的實體定義
type entity struct {
	Name    string   `yaml:"name"`
	Columns []column `yaml:"columns"`
}

// rawSchema 代表 YAML Schema 文件的結構
type rawSchema struct {
	Version  string   `yaml:"version"`
	Entities []entity `yaml:"entities"`
}

// loader 實現從 YAML 格式載入 schema
type loader struct{}

// Load 從 YAML 格式的字節讀取並轉換為 Entity 列表
func (l *loader) Load(data []byte) ([]schema.Entity, error) {
	var raw rawSchema
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to unmarshal yaml: %w", err)
	}

	entities := make([]schema.Entity, 0, len(raw.Entities))
	for _, ent := range raw.Entities {
		columns := make([]schema.Column, 0, len(ent.Columns))
		for _, col := range ent.Columns {
			columns = append(columns, convertColumn(col))
		}
		entities = append(entities, schema.Entity{
			Name:    ent.Name,
			Columns: columns,
		})
	}

	return entities, nil
}

// convertColumn 將 YAML column 轉換為 schema.Column
func convertColumn(col column) schema.Column {
	schemaCol := schema.Column{
		Name:     col.Name,
		RawType:  col.Type,
		IsArray:  col.IsArray,
		IsStruct: col.IsStruct,
	}

	// 處理參考類型
	if col.Ref != "" {
		schemaCol.IsReference = true
		schemaCol.RefExpr = "ref:" + col.Ref

		// 解析 ref 表達式
		if idx := strings.Index(col.Ref, "."); idx != -1 {
			schemaCol.RefEntity = col.Ref[:idx]
			schemaCol.RefField = col.Ref[idx+1:]
		} else {
			schemaCol.RefEntity = col.Ref
		}
	}

	// 處理嵌套欄位（參考類型或結構陣列欄位）
	if len(col.NestedColumns) > 0 {
		nestedCols := make([]schema.Column, 0, len(col.NestedColumns))
		for _, nested := range col.NestedColumns {
			nestedCols = append(nestedCols, convertColumn(nested))
		}
		schemaCol.NestedColumns = nestedCols
	}

	return schemaCol
}

// NewLoader 建立一個 YAML 格式的 Loader
func NewLoader() *loader {
	return &loader{}
}
