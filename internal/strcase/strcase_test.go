package strcase

import "testing"

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty string", input: "", want: ""},
		{name: "lowercase", input: "user", want: "user"},
		{name: "camelCase", input: "userId", want: "user_id"},
		{name: "PascalCase", input: "UserId", want: "user_id"},
		{name: "consecutive uppercase", input: "userID", want: "user_id"},
		{name: "all uppercase", input: "ID", want: "id"},
		{name: "with existing underscore", input: "DemoSide_Quest", want: "demo_side_quest"},
		{name: "multiple underscores", input: "Demo_Side_Quest", want: "demo_side_quest"},
		{name: "complex name", input: "Demo_RankRewards", want: "demo_rank_rewards"},
		{name: "createdAt", input: "createdAt", want: "created_at"},
		{name: "XMLParser", input: "XMLParser", want: "xmlparser"},
		{name: "getHTTPResponse", input: "getHTTPResponse", want: "get_httpresponse"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToSnakeCase(tt.input)
			if got != tt.want {
				t.Errorf("ToSnakeCase(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestToPascalCase(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty string", input: "", want: ""},
		{name: "lowercase", input: "user", want: "User"},
		{name: "already PascalCase", input: "User", want: "User"},
		{name: "snake_case", input: "user_id", want: "User_Id"},
		{name: "camelCase", input: "userId", want: "UserId"},
		{name: "with underscore preserved", input: "DemoSide_Quest", want: "DemoSide_Quest"},
		{name: "lowercase with underscore", input: "demo_side_quest", want: "Demo_Side_Quest"},
		{name: "leading underscore", input: "_leadingUnderscore", want: "_LeadingUnderscore"},
		{name: "multiple underscores", input: "a__b", want: "A__B"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToPascalCase(tt.input)
			if got != tt.want {
				t.Errorf("ToPascalCase(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestToPascalCase_WithRemoveUnderscore(t *testing.T) {
	// 示範調用者如何移除底線
	input := "DemoSide_Quest"
	got := ToPascalCase(input)

	// 調用者自行移除底線
	// result := strings.ReplaceAll(got, "_", "")

	if got != "DemoSide_Quest" {
		t.Errorf("ToPascalCase(%q) = %q, want %q", input, got, "DemoSide_Quest")
	}
}

func TestIsSQLReservedKeyword(t *testing.T) {
	tests := []struct {
		name string
		word string
		want bool
	}{
		{"select 是保留字", "select", true},
		{"SELECT 大寫也是保留字", "SELECT", true},
		{"from 是保留字", "from", true},
		{"table 是保留字", "table", true},
		{"username 不是保留字", "username", false},
		{"id 不是保留字", "id", false},
		{"name 不是保留字", "name", false},
		{"index 是保留字", "index", true},
		{"key 是保留字", "key", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsSQLReservedKeyword(tt.word); got != tt.want {
				t.Errorf("IsSQLReservedKeyword(%q) = %v, want %v", tt.word, got, tt.want)
			}
		})
	}
}
