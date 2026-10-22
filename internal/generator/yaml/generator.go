package yaml

import (
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/miskawu/formgen/internal/mapper"
	"github.com/miskawu/formgen/internal/schema"
)

// CurrentVersion 是當前 YAML Schema 的版本號
const CurrentVersion = "1"

// outputColumn 代表輸出 YAML 中的欄位
type outputColumn struct {
	Name          string         `yaml:"name"`
	Type          string         `yaml:"type"`
	IsArray       bool           `yaml:"isArray"`
	IsStruct      bool           `yaml:"isStruct,omitempty"`      // 是否為結構陣列欄位
	Ref           string         `yaml:"ref,omitempty"`           // 參考表達式（如 "ItemInfo" 或 "Table.Field"）
	NestedColumns []outputColumn `yaml:"nestedColumns,omitempty"` // 嵌套欄位
}

// outputEntity 代表輸出 YAML 中的實體
type outputEntity struct {
	Name    string         `yaml:"name"`
	Columns []outputColumn `yaml:"columns"`
}

// outputSchema 代表輸出的 YAML Schema 結構
type outputSchema struct {
	Version  string         `yaml:"version"`
	Entities []outputEntity `yaml:"entities"`
}

type generator struct {
	typeMapper mapper.TypeMapper
}

// newGenerator 創建 YAML 生成器的內部實現
func newGenerator(opts ...Option) *generator {
	o := defaultOptions()
	for _, opt := range opts {
		opt(o)
	}

	return &generator{
		typeMapper: o.typeMapper,
	}
}

// Generate 將 Entity 列表轉換為單一 YAML Schema
func (g *generator) Generate(entities []schema.Entity) ([]byte, error) {
	s := &outputSchema{
		Version:  CurrentVersion,
		Entities: []outputEntity{},
	}

	for _, entity := range entities {
		outputEnt, err := g.convertEntity(entity)
		if err != nil {
			return nil, fmt.Errorf("failed to convert entity %s: %w", entity.Name, err)
		}
		s.Entities = append(s.Entities, outputEnt)
	}

	data, err := yaml.Marshal(s)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal schema to yaml: %w", err)
	}
	return data, nil
}

// convertEntity 將 schema.Entity 轉換為 outputEntity
func (g *generator) convertEntity(entity schema.Entity) (outputEntity, error) {
	columns := make([]outputColumn, 0, len(entity.Columns))

	for _, col := range entity.Columns {
		outCol, err := g.convertColumn(col)
		if err != nil {
			return outputEntity{}, err
		}
		columns = append(columns, outCol)
	}

	return outputEntity{
		Name:    entity.Name,
		Columns: columns,
	}, nil
}

// convertColumn 將 schema.Column 轉換為 outputColumn
func (g *generator) convertColumn(col schema.Column) (outputColumn, error) {
	var normalizedType string

	// 如果是結構陣列欄位（如 Loot[n].Field 格式產生的欄位），直接使用 RawType 作為類型名稱
	if col.IsStruct {
		normalizedType = col.RawType
	} else {
		var ok bool
		normalizedType, ok = g.typeMapper.Map(col.RawType)
		if !ok {
			return outputColumn{}, fmt.Errorf("unknown type: %s for field: %s", col.RawType, col.Name)
		}
	}

	outCol := outputColumn{
		Name:     col.Name,
		Type:     normalizedType,
		IsArray:  col.IsArray,
		IsStruct: col.IsStruct,
	}

	// 處理參考類型
	if col.IsReference {
		// 構建 ref 表達式
		if col.RefField != "" {
			outCol.Ref = col.RefEntity + "." + col.RefField
		} else {
			outCol.Ref = col.RefEntity
		}
	}

	// 處理嵌套欄位（參考類型或結構陣列欄位）
	if col.IsStruct || len(col.NestedColumns) > 0 {
		nestedCols := make([]outputColumn, 0, len(col.NestedColumns))
		for _, nested := range col.NestedColumns {
			nestedOut, err := g.convertColumn(nested)
			if err != nil {
				return outputColumn{}, err
			}
			nestedCols = append(nestedCols, nestedOut)
		}
		outCol.NestedColumns = nestedCols
	}

	return outCol, nil
}
