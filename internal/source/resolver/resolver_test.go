package resolver

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/miskawu/formgen/internal/schema"
	"github.com/miskawu/formgen/internal/source"
	"github.com/miskawu/formgen/internal/source/excel"
)

func TestParseRefExpr(t *testing.T) {
	tests := []struct {
		name       string
		expr       string
		wantEntity string
		wantField  string
		wantErr    bool
	}{
		{
			name:       "簡單 Entity 參考",
			expr:       "ref:ItemInfo",
			wantEntity: "ItemInfo",
			wantField:  "",
			wantErr:    false,
		},
		{
			name:       "帶欄位的參考",
			expr:       "ref:Demo_ShopGoods.GoodsGroupId",
			wantEntity: "Demo_ShopGoods",
			wantField:  "GoodsGroupId",
			wantErr:    false,
		},
		{
			name:       "空表達式",
			expr:       "",
			wantEntity: "",
			wantField:  "",
			wantErr:    true,
		},
		{
			name:       "無 ref: 前綴",
			expr:       "ItemInfo",
			wantEntity: "",
			wantField:  "",
			wantErr:    true,
		},
		{
			name:       "只有 ref: 前綴",
			expr:       "ref:",
			wantEntity: "",
			wantField:  "",
			wantErr:    true,
		},
		{
			name:       "無效的格式（只有點）",
			expr:       "ref:.",
			wantEntity: "",
			wantField:  "",
			wantErr:    true,
		},
		{
			name:       "無效的格式（缺少欄位名）",
			expr:       "ref:Table.",
			wantEntity: "",
			wantField:  "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotEntity, gotField, err := parseRefExpr(tt.expr)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseRefExpr() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotEntity != tt.wantEntity {
				t.Errorf("parseRefExpr() entityName = %v, want %v", gotEntity, tt.wantEntity)
			}
			if gotField != tt.wantField {
				t.Errorf("parseRefExpr() fieldName = %v, want %v", gotField, tt.wantField)
			}
		})
	}
}

func TestResolver_Resolve(t *testing.T) {
	tests := []struct {
		name     string
		entities []schema.Entity
		want     []string // 預期解析後的特徵
		wantWarn []string // 預期的警告訊息
	}{
		{
			name: "非參考類型不變",
			entities: []schema.Entity{
				{
					Name: "User",
					Columns: []schema.Column{
						{Name: "id", RawType: "int64"},
						{Name: "name", RawType: "string"},
					},
				},
			},
			want: []string{
				"User.id: int64",
				"User.name: string",
			},
		},
		{
			name: "嵌套參考類型解析",
			entities: []schema.Entity{
				{
					Name: "ItemInfo",
					Columns: []schema.Column{
						{Name: "id", RawType: "int64"},
						{Name: "name", RawType: "string"},
						{Name: "price", RawType: "float"},
					},
				},
				{
					Name: "Character",
					Columns: []schema.Column{
						{Name: "charId", RawType: "int64"},
						{
							Name:        "equipment",
							RawType:     "entry",
							IsReference: true,
							RefExpr:     "ref:ItemInfo",
						},
					},
				},
			},
			want: []string{
				"Character.equipment: nested ItemInfo",
				"Character.equipment has 3 nested columns",
			},
		},
		{
			name: "找不到參考表單發出警告",
			entities: []schema.Entity{
				{
					Name: "Character",
					Columns: []schema.Column{
						{Name: "charId", RawType: "int64"},
						{
							Name:        "equipment",
							RawType:     "entry",
							IsReference: true,
							RefExpr:     "ref:NonExistent",
						},
					},
				},
			},
			want: []string{
				"Character.equipment: fallback to entry",
			},
			wantWarn: []string{
				"NonExistent",
				"not found",
			},
		},
		{
			name: "陣列參考類型（帶欄位指定）",
			entities: []schema.Entity{
				{
					Name: "ShopGoods",
					Columns: []schema.Column{
						{Name: "id", RawType: "int64"},
						{Name: "GoodsGroupId", RawType: "int32"},
					},
				},
				{
					Name: "GiftBox",
					Columns: []schema.Column{
						{Name: "id", RawType: "int64"},
						{
							Name:        "goodsIds",
							RawType:     "shortint",
							IsArray:     true,
							IsReference: true,
							RefExpr:     "ref:ShopGoods.GoodsGroupId",
						},
					},
				},
			},
			want: []string{
				"GiftBox.goodsIds: ref to ShopGoods.GoodsGroupId",
			},
		},
		{
			name: "陣列參考類型找不到參考表時應 fallback 為非陣列",
			entities: []schema.Entity{
				{
					Name: "Quest",
					Columns: []schema.Column{
						{Name: "id", RawType: "int64"},
						{
							Name:        "rewards",
							RawType:     "int",
							IsArray:     true,
							IsReference: true,
							RefExpr:     "ref:DropTable.GroupId",
						},
					},
				},
			},
			want: []string{
				"Quest.rewards: fallback to int, IsArray=false",
			},
			wantWarn: []string{
				"DropTable",
				"not found",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var warnings []string
			warnHandler := func(msg string) {
				warnings = append(warnings, msg)
			}

			resolver := New(tt.entities, WithWarnHandler(warnHandler))
			result, err := resolver.Resolve()
			if err != nil {
				t.Fatalf("Resolve() error = %v", err)
			}

			// 檢查結果特徵
			for _, wantFeature := range tt.want {
				found := checkFeature(result, wantFeature)
				if !found {
					t.Errorf("Resolve() missing expected feature: %s", wantFeature)
					t.Logf("Got entities: %+v", result)
				}
			}

			// 檢查警告訊息
			for _, wantWarn := range tt.wantWarn {
				found := false
				for _, w := range warnings {
					if strings.Contains(w, wantWarn) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Resolve() missing expected warning containing: %s", wantWarn)
					t.Logf("Got warnings: %v", warnings)
				}
			}
		})
	}
}

// checkFeature 檢查結果中是否包含預期的特徵
func checkFeature(entities []schema.Entity, wantFeature string) bool {
	for _, entity := range entities {
		for _, col := range entity.Columns {
			// 檢查基本欄位匹配：EntityName.colName: type
			basicMatch := fmt.Sprintf("%s.%s: %s", entity.Name, col.Name, col.RawType)
			if strings.Contains(basicMatch, wantFeature) {
				return true
			}

			// 檢查嵌套參考：nested EntityName
			if len(col.NestedColumns) > 0 {
				nestedMatch := fmt.Sprintf("%s.%s: nested %s", entity.Name, col.Name, col.RefEntity)
				if strings.Contains(nestedMatch, wantFeature) {
					return true
				}
				// 檢查嵌套欄位數量
				countMatch := fmt.Sprintf("%s.%s has %d nested columns", entity.Name, col.Name, len(col.NestedColumns))
				if strings.Contains(countMatch, wantFeature) {
					return true
				}
			}

			// 檢查帶欄位的參考：ref to Entity.Field
			if col.RefEntity != "" && col.RefField != "" {
				refMatch := fmt.Sprintf("%s.%s: ref to %s.%s", entity.Name, col.Name, col.RefEntity, col.RefField)
				if strings.Contains(refMatch, wantFeature) {
					return true
				}
			}

			// 檢查 fallback 情況
			if !col.IsReference && strings.Contains(wantFeature, "fallback") {
				fallbackMatch := fmt.Sprintf("%s.%s: fallback to %s", entity.Name, col.Name, col.RawType)
				if strings.Contains(fallbackMatch, wantFeature) {
					return true
				}
				// 檢查 fallback 後的 IsArray 狀態
				fallbackArrayMatch := fmt.Sprintf("%s.%s: fallback to %s, IsArray=%v", entity.Name, col.Name, col.RawType, col.IsArray)
				if strings.Contains(fallbackArrayMatch, wantFeature) {
					return true
				}
			}
		}
	}
	return false
}

func TestFindEmbeddedEntities(t *testing.T) {
	tests := []struct {
		name         string
		entities     []schema.Entity
		wantEmbedded []string // 預期被嵌入的 Entity 名稱
	}{
		{
			name: "無嵌入關係",
			entities: []schema.Entity{
				{
					Name: "User",
					Columns: []schema.Column{
						{Name: "id", RawType: "int64"},
						{Name: "name", RawType: "string"},
					},
				},
			},
			wantEmbedded: nil,
		},
		{
			name: "有嵌入關係",
			entities: []schema.Entity{
				{
					Name: "ItemInfo",
					Columns: []schema.Column{
						{Name: "id", RawType: "int64"},
						{Name: "name", RawType: "string"},
					},
				},
				{
					Name: "Character",
					Columns: []schema.Column{
						{Name: "charId", RawType: "int64"},
						{
							Name:        "equipment",
							RawType:     "entry",
							IsReference: true,
							RefEntity:   "ItemInfo",
							RefField:    "", // 嵌入整個 Entity
							NestedColumns: []schema.Column{
								{Name: "id", RawType: "int64"},
								{Name: "name", RawType: "string"},
							},
						},
					},
				},
			},
			wantEmbedded: []string{"ItemInfo"},
		},
		{
			name: "帶欄位指定的參考不算嵌入",
			entities: []schema.Entity{
				{
					Name: "ShopGoods",
					Columns: []schema.Column{
						{Name: "id", RawType: "int64"},
						{Name: "GoodsGroupId", RawType: "int32"},
					},
				},
				{
					Name: "GiftBox",
					Columns: []schema.Column{
						{Name: "id", RawType: "int64"},
						{
							Name:        "goodsIds",
							RawType:     "shortint",
							IsArray:     true,
							IsReference: true,
							RefEntity:   "ShopGoods",
							RefField:    "GoodsGroupId", // 指定欄位，不是嵌入整個 Entity
						},
					},
				},
			},
			wantEmbedded: nil, // ShopGoods 不算被嵌入
		},
		{
			name: "fallback 的參考不算嵌入",
			entities: []schema.Entity{
				{
					Name: "Character",
					Columns: []schema.Column{
						{Name: "charId", RawType: "int64"},
						{
							Name:          "equipment",
							RawType:       "entry",
							IsReference:   false, // fallback 後 IsReference 為 false
							NestedColumns: nil,   // fallback 後 NestedColumns 為空
						},
					},
				},
			},
			wantEmbedded: nil,
		},
		{
			name: "多個嵌入關係",
			entities: []schema.Entity{
				{
					Name: "ItemInfo",
					Columns: []schema.Column{
						{Name: "id", RawType: "int64"},
					},
				},
				{
					Name: "SkillInfo",
					Columns: []schema.Column{
						{Name: "id", RawType: "int64"},
					},
				},
				{
					Name: "Character",
					Columns: []schema.Column{
						{Name: "charId", RawType: "int64"},
						{
							Name:        "equipment",
							RawType:     "entry",
							IsReference: true,
							RefEntity:   "ItemInfo",
							NestedColumns: []schema.Column{
								{Name: "id", RawType: "int64"},
							},
						},
						{
							Name:        "skill",
							RawType:     "entry",
							IsReference: true,
							RefEntity:   "SkillInfo",
							NestedColumns: []schema.Column{
								{Name: "id", RawType: "int64"},
							},
						},
					},
				},
			},
			wantEmbedded: []string{"ItemInfo", "SkillInfo"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			embedded := FindEmbeddedEntities(tt.entities)

			// 檢查數量
			if len(embedded) != len(tt.wantEmbedded) {
				t.Errorf("FindEmbeddedEntities() got %d entities, want %d",
					len(embedded), len(tt.wantEmbedded))
				t.Logf("Got: %v", embedded)
			}

			// 檢查每個預期的 Entity 是否在結果中
			for _, want := range tt.wantEmbedded {
				if !embedded[want] {
					t.Errorf("FindEmbeddedEntities() missing expected entity: %s", want)
					t.Logf("Got: %v", embedded)
				}
			}
		})
	}
}

func TestFindReferencedEntities(t *testing.T) {
	tests := []struct {
		name           string
		entities       []schema.Entity
		wantReferenced []string // 預期被引用的 Entity 名稱
	}{
		{
			name: "無引用關係",
			entities: []schema.Entity{
				{
					Name: "User",
					Columns: []schema.Column{
						{Name: "id", RawType: "int64"},
						{Name: "name", RawType: "string"},
					},
				},
			},
			wantReferenced: nil,
		},
		{
			name: "完整嵌入（RefField==\"\" 且有 NestedColumns）",
			entities: []schema.Entity{
				{
					Name: "ItemInfo",
					Columns: []schema.Column{
						{Name: "id", RawType: "int64"},
					},
				},
				{
					Name: "Character",
					Columns: []schema.Column{
						{
							Name:        "equipment",
							IsReference: true,
							RefEntity:   "ItemInfo",
							RefField:    "", // 完整嵌入
							NestedColumns: []schema.Column{
								{Name: "id", RawType: "int64"},
							},
						},
					},
				},
			},
			wantReferenced: []string{"ItemInfo"},
		},
		{
			name: "字段引用（RefField!=\"\"，不需要 NestedColumns）",
			entities: []schema.Entity{
				{
					Name: "ShopGoods",
					Columns: []schema.Column{
						{Name: "id", RawType: "int64"},
						{Name: "GoodsGroupId", RawType: "int32"},
					},
				},
				{
					Name: "Shop",
					Columns: []schema.Column{
						{
							Name:        "goodsGroupId",
							IsReference: true,
							RefEntity:   "ShopGoods",
							RefField:    "GoodsGroupId", // 字段引用
							// 注意：字段引用不需要 NestedColumns
						},
					},
				},
			},
			wantReferenced: []string{"ShopGoods"},
		},
		{
			name: "混合引用（完整嵌入 + 字段引用）",
			entities: []schema.Entity{
				{
					Name: "RegionInfo",
					Columns: []schema.Column{
						{Name: "id", RawType: "int64"},
					},
				},
				{
					Name: "ShopGoods",
					Columns: []schema.Column{
						{Name: "id", RawType: "int64"},
						{Name: "GoodsGroupId", RawType: "int32"},
					},
				},
				{
					Name: "Mission",
					Columns: []schema.Column{
						{
							Name:        "regionId",
							IsReference: true,
							RefEntity:   "RegionInfo",
							RefField:    "", // 完整嵌入
							NestedColumns: []schema.Column{
								{Name: "id", RawType: "int64"},
							},
						},
						{
							Name:        "goodsGroupId",
							IsReference: true,
							RefEntity:   "ShopGoods",
							RefField:    "GoodsGroupId", // 字段引用
						},
					},
				},
			},
			wantReferenced: []string{"RegionInfo", "ShopGoods"},
		},
		{
			name: "完整嵌入但無 NestedColumns（fallback 情況不算引用）",
			entities: []schema.Entity{
				{
					Name: "Character",
					Columns: []schema.Column{
						{
							Name:          "equipment",
							IsReference:   true,
							RefEntity:     "ItemInfo",
							RefField:      "",  // 完整嵌入
							NestedColumns: nil, // 但沒有 NestedColumns（fallback）
						},
					},
				},
			},
			wantReferenced: nil, // 不算引用
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			referenced := FindReferencedEntities(tt.entities)

			// 檢查數量
			if len(referenced) != len(tt.wantReferenced) {
				t.Errorf("FindReferencedEntities() got %d entities, want %d",
					len(referenced), len(tt.wantReferenced))
				t.Logf("Got: %v", referenced)
			}

			// 檢查每個預期的 Entity 是否在結果中
			for _, want := range tt.wantReferenced {
				if !referenced[want] {
					t.Errorf("FindReferencedEntities() missing expected entity: %s", want)
					t.Logf("Got: %v", referenced)
				}
			}
		})
	}
}

func TestResolver_WithSampleExcelFiles(t *testing.T) {
	// 載入合成範例表（由 internal/sampledata 產生）
	files := []string{
		"../../../testdata/Demo_Mission.xlsx",
		"../../../testdata/Demo_Region.xlsx",
		"../../../testdata/Demo_Shop.xlsx",
		"../../../testdata/Demo_ShopGoods.xlsx",
		"../../../testdata/Demo_ShopLabel.xlsx",
	}

	loader := excel.NewLoader()
	parser := source.NewParser()

	var allEntities []schema.Entity

	// 每個檔案只取第一個工作表（第二個是欄位說明）
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("failed to read %s: %v", file, err)
		}

		sheets, err := loader.Load(data)
		if err != nil {
			t.Fatalf("Failed to load %s: %v", file, err)
		}

		entity, err := parser.Parse(sheets[0])
		if err != nil {
			t.Fatalf("failed to parse %s: %v", file, err)
		}
		allEntities = append(allEntities, entity)
	}

	// 使用 Resolver 解析參考
	var warnings []string
	warnHandler := func(msg string) {
		warnings = append(warnings, msg)
	}

	res := New(allEntities, WithWarnHandler(warnHandler))
	resolved, err := res.Resolve()
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	columnOf := func(entityName, colName string) schema.Column {
		t.Helper()
		for _, entity := range resolved {
			if entity.Name != entityName {
				continue
			}
			for _, col := range entity.Columns {
				if col.Name == colName {
					return col
				}
			}
		}
		t.Fatalf("column %s.%s not found", entityName, colName)
		return schema.Column{}
	}

	// 整筆嵌入：Demo_Mission.RegionId → Demo_Region
	regionID := columnOf("Demo_Mission", "RegionId")
	if regionID.RefEntity != "Demo_Region" || regionID.RefField != "" || len(regionID.NestedColumns) == 0 {
		t.Errorf("Demo_Mission.RegionId should embed Demo_Region, got %+v", regionID)
	}

	// 欄位引用：Demo_Shop.GoodsGroupId → Demo_ShopGoods.GroupId（保持陣列）
	goods := columnOf("Demo_Shop", "GoodsGroupId")
	if goods.RefEntity != "Demo_ShopGoods" || goods.RefField != "GroupId" || !goods.IsArray {
		t.Errorf("Demo_Shop.GoodsGroupId should reference Demo_ShopGoods.GroupId as array, got %+v", goods)
	}

	// 找不到參考表：Demo_Mission.Drops fallback 為非陣列的一般欄位
	drops := columnOf("Demo_Mission", "Drops")
	if drops.IsReference || drops.IsArray {
		t.Errorf("Demo_Mission.Drops should fall back to plain non-array column, got %+v", drops)
	}

	// 被嵌入／被引用的 Entity
	embedded := FindEmbeddedEntities(resolved)
	if !embedded["Demo_Region"] || !embedded["Demo_ShopLabel"] || len(embedded) != 2 {
		t.Errorf("FindEmbeddedEntities() = %v, want Demo_Region and Demo_ShopLabel", embedded)
	}
	referenced := FindReferencedEntities(resolved)
	if !referenced["Demo_ShopGoods"] || len(referenced) != 3 {
		t.Errorf("FindReferencedEntities() = %v, want Demo_Region, Demo_ShopLabel, Demo_ShopGoods", referenced)
	}

	// 範例中故意引用了不存在的 TextInfo、DropTable、ItemInfo，應產生警告
	if len(warnings) == 0 {
		t.Error("Expected warnings for unresolved references")
	}
	for _, w := range warnings {
		if !strings.Contains(w, "not found") {
			t.Errorf("unexpected warning: %s", w)
		}
	}
}
