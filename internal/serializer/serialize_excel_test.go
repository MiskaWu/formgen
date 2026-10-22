package serializer

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/miskawu/formgen/internal/source/excel"
	"github.com/miskawu/formgen/internal/strcase"

	"github.com/bufbuild/protocompile"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/dynamicpb"
)

// TestSerializeExcelToProto 測試將 Excel 資料序列化為 protobuf
// 使用合成範例 Demo_Category.xlsx 與 sample.proto 的 DemoCategoryList 進行測試
func TestSerializeExcelToProto(t *testing.T) {
	// 1. 讀取 Excel 檔案
	excelPath := "../../testdata/Demo_Category.xlsx"
	protoPath := "../../testdata/sample.proto"

	excelData, err := os.ReadFile(excelPath)
	if err != nil {
		t.Fatalf("無法讀取 Excel 檔案: %v", err)
	}

	// 2. 使用 Excel loader 載入資料
	loader := excel.NewLoader()
	sheets, err := loader.Load(excelData)
	if err != nil {
		t.Fatalf("無法載入 Excel: %v", err)
	}

	if len(sheets) == 0 {
		t.Fatal("Excel 沒有任何工作表")
	}

	// 取得第一個工作表
	sheet := sheets[0]
	t.Logf("工作表名稱: %s", sheet.Name)
	t.Logf("行數: %d", len(sheet.Rows))

	// 3. 顯示 Excel 結構（前幾行）
	t.Log("\n=== Excel 結構 ===")
	for i, row := range sheet.Rows {
		if i >= 6 { // 只顯示前 6 行（包含表頭和第一筆資料）
			break
		}
		t.Logf("Row %d: %v", i, row)
	}

	// 4. 解析 Excel 資料
	// Excel 格式：
	// Row 0: 註解
	// Row 1: 欄位名稱
	// Row 2: 欄位類型
	// Row 3: 註解
	// Row 4: skip 標記
	// Row 5+: 數據行
	if len(sheet.Rows) < 6 {
		t.Fatal("Excel 資料不足（需要至少 6 行）")
	}

	fieldNames := sheet.Rows[1]
	skipRow := sheet.Rows[4]
	dataRows := sheet.Rows[5:]

	// 建立有效欄位映射
	var validFieldIndices []int
	var validFieldNames []string
	for colIdx, fieldName := range fieldNames {
		fieldName = strings.TrimSpace(fieldName)
		if fieldName == "" {
			continue
		}
		if colIdx < len(skipRow) && strings.TrimSpace(skipRow[colIdx]) == "skip" {
			continue
		}
		validFieldIndices = append(validFieldIndices, colIdx)
		// 將欄位名轉為 snake_case 以匹配 proto 定義（與 importer 一致）
		validFieldNames = append(validFieldNames, strcase.ToSnakeCase(fieldName))
	}

	t.Logf("\n有效欄位: %v", validFieldNames)

	// 5. 轉換為 map 格式
	var dataList []map[string]any
	for _, dataRow := range dataRows {
		data := make(map[string]any)
		for i, colIdx := range validFieldIndices {
			if colIdx >= len(dataRow) {
				continue
			}
			cellValue := strings.TrimSpace(dataRow[colIdx])
			if cellValue == "" {
				continue
			}
			data[validFieldNames[i]] = cellValue
		}
		if len(data) > 0 {
			dataList = append(dataList, data)
		}
	}

	t.Logf("\n=== 原始資料 ===")
	t.Logf("共解析 %d 筆資料", len(dataList))
	for i, data := range dataList {
		t.Logf("  [%d] %v", i, data)
	}

	// 6. 建立 ProtoSerializer 並序列化
	serializer, err := NewProtoSerializer(protoPath)
	if err != nil {
		t.Fatalf("無法建立序列化器: %v", err)
	}

	// 使用 DemoCategoryList 和 DemoCategory 作為 wrapper 和 item 名稱
	wrapperName := "DemoCategoryList"
	itemName := "DemoCategory"

	protoBytes, err := serializer.Serialize(wrapperName, itemName, dataList)
	if err != nil {
		t.Fatalf("序列化失敗: %v", err)
	}

	t.Logf("\n=== 序列化後的 Binary ===")
	t.Logf("大小: %d bytes", len(protoBytes))
	t.Logf("完整 Hex:\n%x", protoBytes)

	// 7. 反序列化並顯示內容
	t.Log("\n=== 反序列化內容 ===")
	deserializeAndPrint(t, protoPath, wrapperName, protoBytes)

	// 8. 反序列化回 map 並核對內容（skip 欄位不應出現）
	items, err := serializer.Deserialize(wrapperName, itemName, protoBytes)
	if err != nil {
		t.Fatalf("反序列化失敗: %v", err)
	}
	if len(items) != len(dataList) {
		t.Fatalf("got %d items, want %d", len(items), len(dataList))
	}
	first := items[0]
	if first["id"] != int32(1) || first["name"] != "武器" || first["enabled"] != true {
		t.Errorf("first item = %v, want id=1 name=武器 enabled=true", first)
	}
	if _, ok := first["__extra"]; ok {
		t.Error("skip 欄位 __extra 不應被序列化")
	}
	// proto3 的 bool 預設值 false 不會出現在 map 裡
	if _, ok := items[2]["enabled"]; ok {
		t.Errorf("third item enabled should be default (absent), got %v", items[2]["enabled"])
	}
}

// deserializeAndPrint 反序列化 protobuf 並輸出 JSON 格式
func deserializeAndPrint(t *testing.T, protoPath, messageName string, data []byte) {
	// 讀取並編譯 proto 文件
	content, err := os.ReadFile(protoPath)
	if err != nil {
		t.Fatalf("無法讀取 proto 檔案: %v", err)
	}

	compiler := protocompile.Compiler{
		Resolver: &protocompile.SourceResolver{
			Accessor: protocompile.SourceAccessorFromMap(map[string]string{
				protoPath: string(content),
			}),
		},
	}

	fds, err := compiler.Compile(context.Background(), protoPath)
	if err != nil {
		t.Fatalf("編譯 proto 失敗: %v", err)
	}

	// 查找 message
	messages := fds[0].Messages()
	for i := 0; i < messages.Len(); i++ {
		md := messages.Get(i)
		if string(md.Name()) == messageName {
			// 創建動態消息
			msg := dynamicpb.NewMessage(md)

			// 使用 proto.Unmarshal 反序列化
			if err := proto.Unmarshal(data, msg); err != nil {
				t.Fatalf("反序列化失敗: %v", err)
			}

			// 轉為 JSON 輸出
			jsonBytes, err := protojson.MarshalOptions{
				Multiline: true,
				Indent:    "  ",
			}.Marshal(msg)
			if err != nil {
				t.Fatalf("轉 JSON 失敗: %v", err)
			}

			t.Logf("JSON 輸出:\n%s", string(jsonBytes))
			return
		}
	}

	t.Fatalf("找不到 message: %s", messageName)
}
