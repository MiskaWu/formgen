package source

import (
	"testing"

	"github.com/miskawu/formgen/internal/schema"
)

func TestParser_Parse(t *testing.T) {
	tests := []struct {
		name    string
		input   Sheet
		want    schema.Entity
		wantErr bool
	}{
		{
			name: "基本欄位轉換",
			input: Sheet{
				Name: "TestSheet",
				Rows: [][]string{
					{"header0"},              // row 0
					{"id", "name", "age"},    // row 1: 欄位名稱
					{"int", "string", "int"}, // row 2: 欄位類型
					{"data0"},                // row 3
					{"", "", ""},             // row 4: skip 標記（都不 skip）
				},
			},
			want: schema.Entity{
				Name: "TestSheet",
				Columns: []schema.Column{
					{Name: "id", RawType: "int", IsArray: false},
					{Name: "name", RawType: "string", IsArray: false},
					{Name: "age", RawType: "int", IsArray: false},
				},
			},
			wantErr: false,
		},
		{
			name: "陣列類型轉換",
			input: Sheet{
				Name: "ArraySheet",
				Rows: [][]string{
					{"header0"},
					{"items", "tags", "scores"},      // row 1: 欄位名稱
					{"string[]", "int[]", "float[]"}, // row 2: 陣列類型
					{"data0"},
					{"", "", ""}, // row 4: skip 標記
				},
			},
			want: schema.Entity{
				Name: "ArraySheet",
				Columns: []schema.Column{
					{Name: "items", RawType: "string", IsArray: true},
					{Name: "tags", RawType: "int", IsArray: true},
					{Name: "scores", RawType: "float", IsArray: true},
				},
			},
			wantErr: false,
		},
		{
			name: "Skip 欄位",
			input: Sheet{
				Name: "SkipSheet",
				Rows: [][]string{
					{"header0"},
					{"id", "internal", "name", "debug"}, // row 1: 欄位名稱
					{"int", "string", "string", "bool"}, // row 2: 欄位類型
					{"data0"},
					{"", "skip", "", "skip"}, // row 4: skip 第 2 和第 4 個欄位
				},
			},
			want: schema.Entity{
				Name: "SkipSheet",
				Columns: []schema.Column{
					{Name: "id", RawType: "int", IsArray: false},
					{Name: "name", RawType: "string", IsArray: false},
				},
			},
			wantErr: false,
		},
		{
			name: "混合陣列與 skip",
			input: Sheet{
				Name: "MixedSheet",
				Rows: [][]string{
					{"header0"},
					{"id", "temp", "items", "count"},     // row 1: 欄位名稱
					{"int", "string", "string[]", "int"}, // row 2: 類型
					{"data0"},
					{"", "skip", "", ""}, // row 4: skip 第 2 個欄位
				},
			},
			want: schema.Entity{
				Name: "MixedSheet",
				Columns: []schema.Column{
					{Name: "id", RawType: "int", IsArray: false},
					{Name: "items", RawType: "string", IsArray: true},
					{Name: "count", RawType: "int", IsArray: false},
				},
			},
			wantErr: false,
		},
		{
			name: "空欄位名稱應該被跳過",
			input: Sheet{
				Name: "EmptyNameSheet",
				Rows: [][]string{
					{"header0"},
					{"id", "", "name"},          // row 1: 第 2 個欄位名稱為空
					{"int", "string", "string"}, // row 2: 類型
					{"data0"},
					{"", "", ""}, // row 4: skip 標記
				},
			},
			want: schema.Entity{
				Name: "EmptyNameSheet",
				Columns: []schema.Column{
					{Name: "id", RawType: "int", IsArray: false},
					{Name: "name", RawType: "string", IsArray: false},
				},
			},
			wantErr: false,
		},
		{
			name: "行數不足應該返回錯誤",
			input: Sheet{
				Name: "InvalidSheet",
				Rows: [][]string{
					{"header0"},
					{"id", "name"},
					{"int", "string"},
					// 少於 5 行
				},
			},
			want:    schema.Entity{},
			wantErr: true,
		},
		{
			name: "陣列欄位格式 Loot[n].Field",
			input: Sheet{
				Name: "RankRewards",
				Rows: [][]string{
					{"header0"},
					{"id", "Loot[0].Kind", "Loot[0].ItemId", "Loot[0].Amount", "Loot[1].Kind", "Loot[1].ItemId", "Loot[1].Amount"},
					{"int64", "int32", "int64", "int32", "int32", "int64", "int32"},
					{"", "", "", "", "", "", ""},
					{"", "", "", "", "", "", ""},
				},
			},
			want: schema.Entity{
				Name: "RankRewards",
				Columns: []schema.Column{
					{Name: "id", RawType: "int64", IsArray: false},
					{
						Name:     "Loot",
						RawType:  "LootStruct",
						IsArray:  true,
						IsStruct: true,
						NestedColumns: []schema.Column{
							{Name: "Kind", RawType: "int32", IsArray: false},
							{Name: "ItemId", RawType: "int64", IsArray: false},
							{Name: "Amount", RawType: "int32", IsArray: false},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "陣列欄位與參考類型混用",
			input: Sheet{
				Name: "RewardTable",
				Rows: [][]string{
					{"header0"},
					{"id", "Reward[0].ItemId", "Reward[0].Count", "Reward[1].ItemId", "Reward[1].Count"},
					{"int64", "*entry", "int32", "*entry", "int32"},
					{"", "ref:ItemInfo", "", "ref:ItemInfo", ""},
					{"", "", "", "", ""},
				},
			},
			want: schema.Entity{
				Name: "RewardTable",
				Columns: []schema.Column{
					{Name: "id", RawType: "int64", IsArray: false},
					{
						Name:     "Reward",
						RawType:  "RewardStruct",
						IsArray:  true,
						IsStruct: true,
						NestedColumns: []schema.Column{
							{Name: "ItemId", RawType: "entry", IsArray: false, IsReference: true, RefExpr: "ref:ItemInfo"},
							{Name: "Count", RawType: "int32", IsArray: false},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "陣列欄位類型不一致應該報錯",
			input: Sheet{
				Name: "InvalidArray",
				Rows: [][]string{
					{"header0"},
					{"id", "Loot[0].Value", "Loot[1].Value"},
					{"int64", "int32", "string"}, // Loot[0].Value 是 int32，Loot[1].Value 是 string
					{"", "", ""},
					{"", "", ""},
				},
			},
			want:    schema.Entity{},
			wantErr: true,
		},
		{
			name: "多個陣列欄位",
			input: Sheet{
				Name: "MultiArray",
				Rows: [][]string{
					{"header0"},
					{"id", "A[0].X", "A[0].Y", "B[0].M", "B[0].N", "A[1].X", "A[1].Y"},
					{"int64", "int32", "int32", "string", "string", "int32", "int32"},
					{"", "", "", "", "", "", ""},
					{"", "", "", "", "", "", ""},
				},
			},
			want: schema.Entity{
				Name: "MultiArray",
				Columns: []schema.Column{
					{Name: "id", RawType: "int64", IsArray: false},
					{
						Name:     "A",
						RawType:  "AStruct",
						IsArray:  true,
						IsStruct: true,
						NestedColumns: []schema.Column{
							{Name: "X", RawType: "int32", IsArray: false},
							{Name: "Y", RawType: "int32", IsArray: false},
						},
					},
					{
						Name:     "B",
						RawType:  "BStruct",
						IsArray:  true,
						IsStruct: true,
						NestedColumns: []schema.Column{
							{Name: "M", RawType: "string", IsArray: false},
							{Name: "N", RawType: "string", IsArray: false},
						},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewParser()
			got, err := parser.Parse(tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// 檢查 Entity Name
			if got.Name != tt.want.Name {
				t.Errorf("Parse() Entity.Name = %v, want %v", got.Name, tt.want.Name)
			}

			// 檢查 Columns 數量
			if len(got.Columns) != len(tt.want.Columns) {
				t.Errorf("Parse() Columns length = %d, want %d", len(got.Columns), len(tt.want.Columns))
				return
			}

			// 檢查每個 Column
			for i, gotCol := range got.Columns {
				wantCol := tt.want.Columns[i]
				if gotCol.Name != wantCol.Name {
					t.Errorf("Parse() Column[%d].Name = %v, want %v", i, gotCol.Name, wantCol.Name)
				}
				if gotCol.RawType != wantCol.RawType {
					t.Errorf("Parse() Column[%d].RawType = %v, want %v", i, gotCol.RawType, wantCol.RawType)
				}
				if gotCol.IsArray != wantCol.IsArray {
					t.Errorf("Parse() Column[%d].IsArray = %v, want %v", i, gotCol.IsArray, wantCol.IsArray)
				}
				if gotCol.IsReference != wantCol.IsReference {
					t.Errorf("Parse() Column[%d].IsReference = %v, want %v", i, gotCol.IsReference, wantCol.IsReference)
				}
				if gotCol.RefExpr != wantCol.RefExpr {
					t.Errorf("Parse() Column[%d].RefExpr = %v, want %v", i, gotCol.RefExpr, wantCol.RefExpr)
				}
				if gotCol.IsStruct != wantCol.IsStruct {
					t.Errorf("Parse() Column[%d].IsStruct = %v, want %v", i, gotCol.IsStruct, wantCol.IsStruct)
				}

				// 檢查 NestedColumns
				if len(gotCol.NestedColumns) != len(wantCol.NestedColumns) {
					t.Errorf("Parse() Column[%d].NestedColumns length = %d, want %d", i, len(gotCol.NestedColumns), len(wantCol.NestedColumns))
					continue
				}
				for j, gotNested := range gotCol.NestedColumns {
					wantNested := wantCol.NestedColumns[j]
					if gotNested.Name != wantNested.Name {
						t.Errorf("Parse() Column[%d].NestedColumns[%d].Name = %v, want %v", i, j, gotNested.Name, wantNested.Name)
					}
					if gotNested.RawType != wantNested.RawType {
						t.Errorf("Parse() Column[%d].NestedColumns[%d].RawType = %v, want %v", i, j, gotNested.RawType, wantNested.RawType)
					}
					if gotNested.IsArray != wantNested.IsArray {
						t.Errorf("Parse() Column[%d].NestedColumns[%d].IsArray = %v, want %v", i, j, gotNested.IsArray, wantNested.IsArray)
					}
					if gotNested.IsReference != wantNested.IsReference {
						t.Errorf("Parse() Column[%d].NestedColumns[%d].IsReference = %v, want %v", i, j, gotNested.IsReference, wantNested.IsReference)
					}
					if gotNested.RefExpr != wantNested.RefExpr {
						t.Errorf("Parse() Column[%d].NestedColumns[%d].RefExpr = %v, want %v", i, j, gotNested.RefExpr, wantNested.RefExpr)
					}
				}
			}
		})
	}
}
