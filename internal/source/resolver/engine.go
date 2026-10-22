package resolver

import (
	"fmt"
	"strings"

	"github.com/miskawu/formgen/internal/schema"
)

type engine struct {
	entities    []schema.Entity
	entityIndex map[string]*schema.Entity
	opts        *options
}

// Resolve 解析所有 Entity 中的參考類型
func (e *engine) Resolve() ([]schema.Entity, error) {
	result := make([]schema.Entity, len(e.entities))

	for i, entity := range e.entities {
		resolved, err := e.resolveEntity(entity)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve entity %s: %w", entity.Name, err)
		}
		result[i] = resolved
	}

	return result, nil
}

// resolveEntity 解析單個 Entity 中的參考類型
func (e *engine) resolveEntity(entity schema.Entity) (schema.Entity, error) {
	resolvedColumns := make([]schema.Column, len(entity.Columns))

	for i, col := range entity.Columns {
		resolved, err := e.resolveColumn(col)
		if err != nil {
			return schema.Entity{}, fmt.Errorf("failed to resolve column %s: %w", col.Name, err)
		}
		resolvedColumns[i] = resolved
	}

	return schema.Entity{
		Name:     entity.Name,
		Columns:  resolvedColumns,
		Metadata: entity.Metadata,
	}, nil
}

// resolveColumn 解析單個欄位的參考類型
func (e *engine) resolveColumn(col schema.Column) (schema.Column, error) {
	// 非參考類型，直接返回
	if !col.IsReference {
		return col, nil
	}

	// 解析參考表達式
	refEntity, refField, err := parseRefExpr(col.RefExpr)
	if err != nil {
		// 參考表達式解析失敗，照舊處理
		e.opts.warnHandler(fmt.Sprintf("invalid ref expression for column %s: %s", col.Name, col.RefExpr))
		return e.fallbackColumn(col), nil
	}

	col.RefEntity = refEntity
	col.RefField = refField

	// 查找參考的 Entity
	targetEntity, ok := e.entityIndex[refEntity]
	if !ok {
		// 找不到參考表單，發出警告並照舊處理
		e.opts.warnHandler(fmt.Sprintf("referenced entity %q not found for column %s", refEntity, col.Name))
		return e.fallbackColumn(col), nil
	}

	// 檢查目標 Entity 是否有 skip 標記（透過 Metadata 判斷）
	// 注意：這裡的 skip 是指整個 Entity 被 skip，不是單個欄位
	// 如果是單個欄位的 skip，在 Parser 階段就已經被過濾掉了

	// 如果指定了特定欄位（如 ref:Table.Field），收集該欄位的值
	if refField != "" {
		// 陣列參考：從指定欄位收集值
		// 這裡只設置參考資訊，實際的值收集由 Generator 或其他層處理
		col.RefField = refField
		return col, nil
	}

	// 嵌套整個 Entity 作為 message 類型
	// 複製目標 Entity 的所有欄位作為嵌套欄位
	col.NestedColumns = make([]schema.Column, len(targetEntity.Columns))
	copy(col.NestedColumns, targetEntity.Columns)

	return col, nil
}

// fallbackColumn 將參考類型欄位回退為普通欄位
func (e *engine) fallbackColumn(col schema.Column) schema.Column {
	result := schema.Column{
		Name:        col.Name,
		RawType:     col.RawType,
		IsArray:     col.IsArray,
		IsReference: false, // 回退為非參考類型
	}

	// 如果是陣列參考（有 RefField），fallback 時也應該把 IsArray 設為 false
	// 陣列參考的語義是「從 ref 表收集多個值組成陣列」
	// 如果 ref 表不存在，就沒有「收集多個值」的概念，應該存原始單一值
	if col.RefField != "" {
		result.IsArray = false
	}

	return result
}

// parseRefExpr 解析參考表達式
// 格式：ref:EntityName 或 ref:EntityName.FieldName
func parseRefExpr(expr string) (entityName, fieldName string, err error) {
	if expr == "" {
		return "", "", fmt.Errorf("empty ref expression")
	}

	// 移除 "ref:" 前綴
	if !strings.HasPrefix(expr, "ref:") {
		return "", "", fmt.Errorf("invalid ref expression format: %s", expr)
	}

	refPart := strings.TrimPrefix(expr, "ref:")
	refPart = strings.TrimSpace(refPart)

	if refPart == "" {
		return "", "", fmt.Errorf("empty ref target")
	}

	// 檢查是否有欄位指定（Entity.Field 格式）
	if idx := strings.Index(refPart, "."); idx != -1 {
		entityName = refPart[:idx]
		fieldName = refPart[idx+1:]
		if entityName == "" || fieldName == "" {
			return "", "", fmt.Errorf("invalid ref expression: %s", expr)
		}
		return entityName, fieldName, nil
	}

	// 只有 Entity 名稱
	return refPart, "", nil
}
