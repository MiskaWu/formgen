package source

import "testing"

func TestLayout_Validate(t *testing.T) {
	tests := []struct {
		name    string
		layout  Layout
		wantErr bool
	}{
		{name: "預設版面", layout: DefaultLayout()},
		{
			name:   "精簡版面（無註解列）",
			layout: Layout{NameRow: 0, TypeRow: 1, MetaRow: 2, OutputRow: 3, DataRow: 4, SkipMarker: "skip"},
		},
		{
			name:    "負數列索引",
			layout:  Layout{NameRow: -1, TypeRow: 1, MetaRow: 2, OutputRow: 3, DataRow: 4, SkipMarker: "skip"},
			wantErr: true,
		},
		{
			name:    "表頭列重複",
			layout:  Layout{NameRow: 1, TypeRow: 1, MetaRow: 2, OutputRow: 3, DataRow: 4, SkipMarker: "skip"},
			wantErr: true,
		},
		{
			name:    "資料列在表頭之前",
			layout:  Layout{NameRow: 1, TypeRow: 2, MetaRow: 3, OutputRow: 4, DataRow: 3, SkipMarker: "skip"},
			wantErr: true,
		},
		{
			name:    "空的 skip 標記",
			layout:  Layout{NameRow: 1, TypeRow: 2, MetaRow: 3, OutputRow: 4, DataRow: 5, SkipMarker: " "},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.layout.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLayout_HeaderRows(t *testing.T) {
	if got := DefaultLayout().HeaderRows(); got != 5 {
		t.Errorf("DefaultLayout().HeaderRows() = %d, want 5", got)
	}
	l := Layout{NameRow: 0, TypeRow: 3, MetaRow: 1, OutputRow: 2, DataRow: 4, SkipMarker: "skip"}
	if got := l.HeaderRows(); got != 4 {
		t.Errorf("HeaderRows() = %d, want 4", got)
	}
}

func TestCell(t *testing.T) {
	rows := [][]string{{" a ", "b"}, {"c"}}
	tests := []struct {
		row, col int
		want     string
	}{
		{0, 0, "a"},
		{0, 1, "b"},
		{1, 1, ""},
		{2, 0, ""},
		{-1, 0, ""},
		{0, -1, ""},
	}
	for _, tt := range tests {
		if got := Cell(rows, tt.row, tt.col); got != tt.want {
			t.Errorf("Cell(%d, %d) = %q, want %q", tt.row, tt.col, got, tt.want)
		}
	}
}

func TestParser_Parse_WithCustomLayout(t *testing.T) {
	// 自訂版面：沒有註解列，類型在名稱之前，輸出類型用 "-" 表示略過
	layout := Layout{TypeRow: 0, NameRow: 1, OutputRow: 2, MetaRow: 3, DataRow: 4, SkipMarker: "-"}
	sheet := Sheet{
		Name: "Custom",
		Rows: [][]string{
			{"int", "string", "*entry", "string"},
			{"id", "name", "ownerId", "note"},
			{"", "", "", "-"},
			{"", "", "ref:Owner", ""},
			{"1", "foo", "7", "ignored"},
		},
	}

	got, err := NewParser(WithLayout(layout)).Parse(sheet)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(got.Columns) != 3 {
		t.Fatalf("Parse() got %d columns, want 3: %+v", len(got.Columns), got.Columns)
	}
	owner := got.Columns[2]
	if owner.Name != "ownerId" || !owner.IsReference || owner.RefExpr != "ref:Owner" || owner.RawType != "entry" {
		t.Errorf("ownerId column = %+v, want reference to ref:Owner", owner)
	}
}

func TestParser_Parse_InvalidLayout(t *testing.T) {
	bad := DefaultLayout()
	bad.DataRow = 2
	_, err := NewParser(WithLayout(bad)).Parse(Sheet{Name: "X", Rows: make([][]string, 10)})
	if err == nil {
		t.Error("Parse() with invalid layout should return error")
	}
}
