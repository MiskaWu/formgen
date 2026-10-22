package schema

type Column struct {
	Name    string
	RawType string
	IsArray bool

	// 參考類型支援（用於 *entry 等修飾符）
	IsReference bool   // 是否為參考類型（帶 * 修飾符）
	RefExpr     string // 原始參考表達式（如 "ref:ItemInfo" 或 "ref:Table.Field"）

	// 解析後的參考資訊（由 Resolver 填充）
	RefEntity     string   // 參考的 Entity 名稱
	RefField      string   // 參考的欄位名稱（可為空表示整個 Entity）
	NestedColumns []Column // 嵌套欄位（解析後填充，用於嵌套 message）

	// 結構陣列標記（用於 Loot[n].Field 格式的陣列欄位）
	IsStruct bool // 是否為結構陣列欄位（由 Parser 設置）
}

// EntityMetadata 包含 Entity 的元數據資訊
type EntityMetadata struct {
	// IsWrapper 標記此 Entity 是否為包裝類型（如 repeated message 的容器）
	// 包裝類型不應被 SQL/ORM 生成器處理
	IsWrapper bool

	// SourceEntity 指向此包裝類型所包裝的原始 Entity 名稱
	// 僅當 IsWrapper 為 true 時有效
	SourceEntity string
}

type Entity struct {
	Name     string
	Columns  []Column
	Metadata EntityMetadata // 元數據，用於標記特殊屬性
}

// Loader 定義從不同格式載入 schema 的介面
type Loader interface {
	Load(data []byte) ([]Entity, error)
}
