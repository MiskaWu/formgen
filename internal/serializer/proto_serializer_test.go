package serializer

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestProtoSerializer(t *testing.T) {
	// 創建測試 proto 文件（包含包裝 message）
	protoContent := `syntax = "proto3";

package test;

message TestTable {
  int32 id = 1;
  string name = 2;
  int32 age = 3;
  string email = 4;
}

message TestTables {
  repeated TestTable items = 1;
}
`
	tmpFile := "test_temp.proto"
	if err := os.WriteFile(tmpFile, []byte(protoContent), 0644); err != nil {
		t.Fatalf("Failed to create temp proto file: %v", err)
	}
	defer os.Remove(tmpFile)

	// 創建序列化器
	serializer, err := NewProtoSerializer(tmpFile)
	if err != nil {
		t.Fatalf("Failed to create serializer: %v", err)
	}

	// 測試序列化（使用包裝 message）
	dataList := []map[string]any{
		{
			"id":    "1",
			"name":  "Alice",
			"age":   "30",
			"email": "alice@example.com",
		},
		{
			"id":    "2",
			"name":  "Bob",
			"age":   "25",
			"email": "bob@example.com",
		},
	}

	bytes, err := serializer.Serialize("TestTables", "TestTable", dataList)
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	if len(bytes) == 0 {
		t.Error("Serialized bytes should not be empty")
	}

	t.Logf("Serialized %d bytes for %d items", len(bytes), len(dataList))
}

func TestProtoSerializer_MissingField(t *testing.T) {
	// 創建測試 proto 文件
	protoContent := `syntax = "proto3";

package test;

message TestTable {
  int32 id = 1;
  string name = 2;
}

message TestTables {
  repeated TestTable items = 1;
}
`
	tmpFile := "test_temp2.proto"
	if err := os.WriteFile(tmpFile, []byte(protoContent), 0644); err != nil {
		t.Fatalf("Failed to create temp proto file: %v", err)
	}
	defer os.Remove(tmpFile)

	serializer, err := NewProtoSerializer(tmpFile)
	if err != nil {
		t.Fatalf("Failed to create serializer: %v", err)
	}

	// 包含額外字段（proto 中不存在）
	dataList := []map[string]any{
		{
			"id":    "1",
			"name":  "Bob",
			"extra": "should be ignored",
		},
	}

	bytes, err := serializer.Serialize("TestTables", "TestTable", dataList)
	if err != nil {
		t.Fatalf("Failed to serialize with extra fields: %v", err)
	}

	if len(bytes) == 0 {
		t.Error("Serialized bytes should not be empty")
	}
}

func TestProtoSerializer_TypeConversion(t *testing.T) {
	protoContent := `syntax = "proto3";

package test;

message Numbers {
  int32 int_val = 1;
  float float_val = 2;
  bool bool_val = 3;
}

message NumbersList {
  repeated Numbers items = 1;
}
`
	tmpFile := "test_temp3.proto"
	if err := os.WriteFile(tmpFile, []byte(protoContent), 0644); err != nil {
		t.Fatalf("Failed to create temp proto file: %v", err)
	}
	defer os.Remove(tmpFile)

	serializer, err := NewProtoSerializer(tmpFile)
	if err != nil {
		t.Fatalf("Failed to create serializer: %v", err)
	}

	// 測試字符串到各種類型的轉換
	dataList := []map[string]any{
		{
			"int_val":   "42",
			"float_val": "3.14",
			"bool_val":  "true",
		},
	}

	bytes, err := serializer.Serialize("NumbersList", "Numbers", dataList)
	if err != nil {
		t.Fatalf("Failed to serialize with type conversions: %v", err)
	}

	if len(bytes) == 0 {
		t.Error("Serialized bytes should not be empty")
	}

	t.Logf("Successfully serialized with type conversions: %d bytes", len(bytes))
}

func TestProtoSerializer_Deserialize(t *testing.T) {
	// 創建測試 proto 文件
	protoContent := `syntax = "proto3";

package test;

message TestTable {
  int32 id = 1;
  string name = 2;
  int32 age = 3;
  string email = 4;
}

message TestTables {
  repeated TestTable items = 1;
}
`
	tmpFile := "test_deserialize.proto"
	if err := os.WriteFile(tmpFile, []byte(protoContent), 0644); err != nil {
		t.Fatalf("Failed to create temp proto file: %v", err)
	}
	defer os.Remove(tmpFile)

	serializer, err := NewProtoSerializer(tmpFile)
	if err != nil {
		t.Fatalf("Failed to create serializer: %v", err)
	}

	// 原始數據
	originalData := []map[string]any{
		{
			"id":    "1",
			"name":  "Alice",
			"age":   "30",
			"email": "alice@example.com",
		},
		{
			"id":    "2",
			"name":  "Bob",
			"age":   "25",
			"email": "bob@example.com",
		},
	}

	// 序列化
	protoBytes, err := serializer.Serialize("TestTables", "TestTable", originalData)
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	// 反序列化
	result, err := serializer.Deserialize("TestTables", "TestTable", protoBytes)
	if err != nil {
		t.Fatalf("Failed to deserialize: %v", err)
	}

	// 驗證結果
	if len(result) != 2 {
		t.Errorf("Expected 2 items, got %d", len(result))
	}

	// 驗證第一筆數據
	if result[0]["id"] != int32(1) {
		t.Errorf("Expected id=1, got %v (type: %T)", result[0]["id"], result[0]["id"])
	}
	if result[0]["name"] != "Alice" {
		t.Errorf("Expected name=Alice, got %v", result[0]["name"])
	}
	if result[0]["age"] != int32(30) {
		t.Errorf("Expected age=30, got %v", result[0]["age"])
	}

	// 驗證第二筆數據
	if result[1]["id"] != int32(2) {
		t.Errorf("Expected id=2, got %v", result[1]["id"])
	}
	if result[1]["name"] != "Bob" {
		t.Errorf("Expected name=Bob, got %v", result[1]["name"])
	}

	t.Logf("Successfully deserialized %d items", len(result))
}

func TestProtoSerializer_Deserialize_Gzip(t *testing.T) {
	// 創建測試 proto 文件
	protoContent := `syntax = "proto3";

package test;

message TestTable {
  int32 id = 1;
  string name = 2;
  int32 age = 3;
}

message TestTables {
  repeated TestTable items = 1;
}
`
	tmpFile := "test_deserialize_gzip.proto"
	if err := os.WriteFile(tmpFile, []byte(protoContent), 0644); err != nil {
		t.Fatalf("Failed to create temp proto file: %v", err)
	}
	defer os.Remove(tmpFile)

	serializer, err := NewProtoSerializer(tmpFile)
	if err != nil {
		t.Fatalf("Failed to create serializer: %v", err)
	}

	// 原始數據
	originalData := []map[string]any{
		{"id": "1", "name": "Alice", "age": "30"},
		{"id": "2", "name": "Bob", "age": "25"},
	}

	// 序列化
	protoBytes, err := serializer.Serialize("TestTables", "TestTable", originalData)
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	// Gzip 壓縮
	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)
	if _, err := gzipWriter.Write(protoBytes); err != nil {
		t.Fatalf("Failed to gzip: %v", err)
	}
	gzipWriter.Close()
	gzippedBytes := buf.Bytes()

	t.Logf("Original size: %d bytes, Gzipped size: %d bytes", len(protoBytes), len(gzippedBytes))

	// 反序列化（應自動檢測並解壓 gzip）
	result, err := serializer.Deserialize("TestTables", "TestTable", gzippedBytes)
	if err != nil {
		t.Fatalf("Failed to deserialize gzipped data: %v", err)
	}

	// 驗證結果
	if len(result) != 2 {
		t.Errorf("Expected 2 items, got %d", len(result))
	}

	if result[0]["id"] != int32(1) {
		t.Errorf("Expected id=1, got %v", result[0]["id"])
	}
	if result[0]["name"] != "Alice" {
		t.Errorf("Expected name=Alice, got %v", result[0]["name"])
	}

	t.Logf("Successfully deserialized gzipped data: %d items", len(result))
}

func TestProtoSerializer_Deserialize_NestedMessage(t *testing.T) {
	// 創建包含嵌套 message 的 proto 文件
	protoContent := `syntax = "proto3";

package test;

message Address {
  string city = 1;
  string street = 2;
}

message Person {
  int32 id = 1;
  string name = 2;
  Address address = 3;
}

message Persons {
  repeated Person items = 1;
}
`
	tmpFile := "test_deserialize_nested.proto"
	if err := os.WriteFile(tmpFile, []byte(protoContent), 0644); err != nil {
		t.Fatalf("Failed to create temp proto file: %v", err)
	}
	defer os.Remove(tmpFile)

	serializer, err := NewProtoSerializer(tmpFile)
	if err != nil {
		t.Fatalf("Failed to create serializer: %v", err)
	}

	// 原始數據（包含嵌套結構）
	originalData := []map[string]any{
		{
			"id":   "1",
			"name": "Alice",
			"address": map[string]any{
				"city":   "Taipei",
				"street": "Main St",
			},
		},
	}

	// 序列化
	protoBytes, err := serializer.Serialize("Persons", "Person", originalData)
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	// 反序列化
	result, err := serializer.Deserialize("Persons", "Person", protoBytes)
	if err != nil {
		t.Fatalf("Failed to deserialize: %v", err)
	}

	// 驗證結果
	if len(result) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(result))
	}

	// 驗證嵌套結構
	address, ok := result[0]["address"].(map[string]any)
	if !ok {
		t.Fatalf("Expected address to be map[string]any, got %T", result[0]["address"])
	}

	if address["city"] != "Taipei" {
		t.Errorf("Expected city=Taipei, got %v", address["city"])
	}
	if address["street"] != "Main St" {
		t.Errorf("Expected street=Main St, got %v", address["street"])
	}

	t.Logf("Successfully deserialized nested message")
}

// TestProtoSerializer_RoundTrip_SampleShop 用 testdata/sample.proto 序列化一份
// 含「整筆嵌入」與「欄位引用」的商店資料，gzip 後再反序列化，確認巢狀結構完整保留。
func TestProtoSerializer_RoundTrip_SampleShop(t *testing.T) {
	protoFile := filepath.Join("..", "..", "testdata", "sample.proto")

	serializer, err := NewProtoSerializer(protoFile)
	if err != nil {
		t.Fatalf("Failed to create serializer: %v", err)
	}
	if err := serializer.ValidateProtoFile(); err != nil {
		t.Fatalf("ValidateProtoFile() error = %v", err)
	}

	// 與 importer 產生的快照資料同形：值為字串，巢狀表為 map / []map
	dataList := []map[string]any{
		{
			"id":         "1",
			"label":      map[string]any{"id": "1", "label_text": "7001", "label_order": "1"},
			"sort_order": "1",
			"goods_group_id": []map[string]any{
				{"id": "101", "group_id": "11", "price": "100", "discount": "1"},
				{"id": "102", "group_id": "11", "price": "250", "discount": "0.9"},
			},
			"display_type": "3",
			"name_text":    "7101",
		},
		{
			"id":             "3",
			"label":          map[string]any{"id": "3", "label_text": "7003", "label_order": "3"},
			"goods_group_id": []map[string]any{},
			"pick_count":     "1",
		},
	}

	protoBytes, err := serializer.Serialize("DemoShopList", "DemoShop", dataList)
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	if _, err := gw.Write(protoBytes); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}

	// 反序列化（自動解壓 gzip）
	result, err := serializer.Deserialize("DemoShopList", "DemoShop", buf.Bytes())
	if err != nil {
		t.Fatalf("Failed to deserialize: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("Expected 2 shops, got %d", len(result))
	}

	label, ok := result[0]["label"].(map[string]any)
	if !ok || label["label_text"] != int32(7001) {
		t.Errorf("label = %v, want embedded DemoShopLabel with label_text=7001", result[0]["label"])
	}

	goods, ok := result[0]["goods_group_id"].([]any)
	if !ok || len(goods) != 2 {
		t.Fatalf("goods_group_id = %v, want 2 embedded DemoShopGoods", result[0]["goods_group_id"])
	}
	second, _ := goods[1].(map[string]any)
	if second["price"] != int32(250) || second["discount"] != float32(0.9) {
		t.Errorf("second goods = %v, want price=250 discount=0.9", second)
	}

	if _, ok := result[1]["goods_group_id"]; ok {
		t.Errorf("empty goods list should be omitted, got %v", result[1]["goods_group_id"])
	}
	if result[1]["pick_count"] != int32(1) {
		t.Errorf("pick_count = %v, want 1", result[1]["pick_count"])
	}

	jsonBytes, _ := json.MarshalIndent(result, "", "  ")
	t.Logf("Deserialized:\n%s", jsonBytes)
}

func TestProtoSerializer_ValidateProtoFile(t *testing.T) {
	dir := t.TempDir()
	valid := filepath.Join(dir, "valid.proto")
	invalid := filepath.Join(dir, "invalid.proto")
	if err := os.WriteFile(valid, []byte("syntax = \"proto3\";\nmessage A { int32 id = 1; }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(invalid, []byte("syntax = \"proto3\";\nmessage A { int32 id = ; }\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{name: "合法的 proto", path: valid},
		{name: "語法錯誤", path: invalid, wantErr: true},
		{name: "檔案不存在", path: filepath.Join(dir, "missing.proto"), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := NewProtoSerializer(tt.path)
			if err != nil {
				t.Fatal(err)
			}
			if err := s.ValidateProtoFile(); (err != nil) != tt.wantErr {
				t.Errorf("ValidateProtoFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
