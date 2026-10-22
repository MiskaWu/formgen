package importer

import (
	"encoding/json"
	"testing"
)

func TestStructFieldCollector_Add_ToJSON(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*structFieldCollector)
		wantJSON map[string]string
		wantErr  bool
	}{
		{
			name: "基本結構陣列",
			setup: func(c *structFieldCollector) {
				c.Add("Loot", 0, "Kind", "1")
				c.Add("Loot", 0, "ItemId", "100")
				c.Add("Loot", 1, "Kind", "2")
				c.Add("Loot", 1, "ItemId", "200")
			},
			wantJSON: map[string]string{
				"Loot": `[{"item_id":"100","kind":"1"},{"item_id":"200","kind":"2"}]`,
			},
		},
		{
			name: "多個結構陣列",
			setup: func(c *structFieldCollector) {
				c.Add("Loot", 0, "Type", "1")
				c.Add("Reward", 0, "ItemId", "999")
			},
			wantJSON: map[string]string{
				"Loot":   `[{"type":"1"}]`,
				"Reward": `[{"item_id":"999"}]`,
			},
		},
		{
			name: "稀疏索引（跳過中間索引）",
			setup: func(c *structFieldCollector) {
				c.Add("Loot", 0, "Type", "1")
				c.Add("Loot", 2, "Type", "3") // 跳過 index 1
			},
			wantJSON: map[string]string{
				"Loot": `[{"type":"1"},{"type":"3"}]`, // index 1 被跳過
			},
		},
		{
			name:     "空收集器",
			setup:    func(c *structFieldCollector) {},
			wantJSON: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newStructFieldCollector()
			tt.setup(c)

			gotJSON, err := c.ToJSON()
			if (err != nil) != tt.wantErr {
				t.Errorf("ToJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(gotJSON) != len(tt.wantJSON) {
				t.Errorf("ToJSON() got %d arrays, want %d", len(gotJSON), len(tt.wantJSON))
				return
			}

			for arrayName, wantStr := range tt.wantJSON {
				gotStr, ok := gotJSON[arrayName]
				if !ok {
					t.Errorf("ToJSON() missing array %s", arrayName)
					continue
				}

				// 比較 JSON（忽略 key 順序）
				var gotData, wantData []map[string]any
				if err := json.Unmarshal([]byte(gotStr), &gotData); err != nil {
					t.Errorf("failed to parse got JSON: %v", err)
					continue
				}
				if err := json.Unmarshal([]byte(wantStr), &wantData); err != nil {
					t.Errorf("failed to parse want JSON: %v", err)
					continue
				}

				if len(gotData) != len(wantData) {
					t.Errorf("array %s: got %d elements, want %d", arrayName, len(gotData), len(wantData))
					continue
				}

				for i := range gotData {
					for k, wantV := range wantData[i] {
						if gotV, ok := gotData[i][k]; !ok || gotV != wantV {
							t.Errorf("array %s[%d].%s: got %v, want %v", arrayName, i, k, gotV, wantV)
						}
					}
				}
			}
		})
	}
}

func TestStructFieldCollector_ToSlice(t *testing.T) {
	c := newStructFieldCollector()
	c.Add("Loot", 0, "Kind", "1")
	c.Add("Loot", 0, "ItemId", "100")
	c.Add("Loot", 1, "Kind", "2")

	got := c.ToSlice()

	if len(got) != 1 {
		t.Fatalf("ToSlice() got %d arrays, want 1", len(got))
	}

	loot, ok := got["Loot"]
	if !ok {
		t.Fatal("ToSlice() missing Loot array")
	}

	if len(loot) != 2 {
		t.Fatalf("Loot got %d elements, want 2", len(loot))
	}

	// 驗證欄位名是 snake_case
	if loot[0]["kind"] != "1" {
		t.Errorf("loot[0].kind = %v, want 1", loot[0]["kind"])
	}
	if loot[0]["item_id"] != "100" {
		t.Errorf("loot[0].item_id = %v, want 100", loot[0]["item_id"])
	}
}

func TestParseStructField(t *testing.T) {
	tests := []struct {
		input         string
		wantArrayName string
		wantIndex     int
		wantFieldName string
		wantMatch     bool
	}{
		{"Loot[0].Kind", "Loot", 0, "Kind", true},
		{"Reward[1].ItemId", "Reward", 1, "ItemId", true},
		{"Items[10].Name", "Items", 10, "Name", true},
		{"normalField", "", 0, "", false},
		{"Loot[0]", "", 0, "", false},       // 缺少欄位名
		{"Loot.Field", "", 0, "", false},    // 缺少索引
		{"[0].Field", "", 0, "", false},     // 缺少陣列名
		{"Loot[a].Field", "", 0, "", false}, // 非數字索引
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			arrayName, index, fieldName, isMatch := parseStructField(tt.input)

			if isMatch != tt.wantMatch {
				t.Errorf("parseStructField(%q) match = %v, want %v", tt.input, isMatch, tt.wantMatch)
				return
			}

			if !isMatch {
				return
			}

			if arrayName != tt.wantArrayName {
				t.Errorf("arrayName = %q, want %q", arrayName, tt.wantArrayName)
			}
			if index != tt.wantIndex {
				t.Errorf("index = %d, want %d", index, tt.wantIndex)
			}
			if fieldName != tt.wantFieldName {
				t.Errorf("fieldName = %q, want %q", fieldName, tt.wantFieldName)
			}
		})
	}
}
