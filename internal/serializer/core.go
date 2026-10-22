package serializer

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/bufbuild/protocompile"
	"github.com/bufbuild/protocompile/parser"
	"github.com/bufbuild/protocompile/reporter"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

// Serialize 將多筆數據序列化為包裝 message 的 protobuf bytes
// wrapperName: 包裝 message 名稱（如 "EventInfos"）
// itemName: 項目 message 名稱（如 "EventInfo"）
// dataList: 多筆數據的映射列表
func (s *ProtoSerializer) Serialize(wrapperName string, itemName string, dataList []map[string]any) ([]byte, error) {
	// 獲取包裝 message descriptor
	wrapperDesc, err := s.getMessageDescriptor(wrapperName)
	if err != nil {
		return nil, fmt.Errorf("failed to get wrapper message descriptor for %s: %w", wrapperName, err)
	}

	// 獲取項目 message descriptor
	itemDesc, err := s.getMessageDescriptor(itemName)
	if err != nil {
		return nil, fmt.Errorf("failed to get item message descriptor for %s: %w", itemName, err)
	}

	// 創建包裝消息
	wrapperMsg := dynamicpb.NewMessage(wrapperDesc)

	// 找到 repeated 字段（通常是 "items"）
	fields := wrapperDesc.Fields()
	var repeatedField protoreflect.FieldDescriptor
	for i := 0; i < fields.Len(); i++ {
		f := fields.Get(i)
		if f.IsList() && f.Message() != nil && f.Message().FullName() == itemDesc.FullName() {
			repeatedField = f
			break
		}
	}

	if repeatedField == nil {
		return nil, fmt.Errorf("no repeated field of type %s found in %s", itemName, wrapperName)
	}

	// 獲取 list 並填充項目
	list := wrapperMsg.Mutable(repeatedField).List()
	for i, data := range dataList {
		// 創建項目消息
		itemMsg := dynamicpb.NewMessage(itemDesc)

		// 對 keys 排序以確保序列化結果穩定
		keys := make([]string, 0, len(data))
		for k := range data {
			keys = append(keys, k)
		}
		slices.Sort(keys)

		// 填充字段（按排序後的順序）
		for _, fieldName := range keys {
			value := data[fieldName]
			if err := s.setField(itemMsg, fieldName, value); err != nil {
				return nil, fmt.Errorf("failed to set field %s in item %d: %w", fieldName, i, err)
			}
		}

		// 添加到 list
		list.Append(protoreflect.ValueOfMessage(itemMsg))
	}

	// 序列化（使用 Deterministic 確保相同輸入產生相同輸出）
	bytes, err := proto.MarshalOptions{Deterministic: true}.Marshal(wrapperMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal wrapper message: %w", err)
	}

	return bytes, nil
}

// getMessageDescriptor 獲取指定消息的描述符
func (s *ProtoSerializer) getMessageDescriptor(messageName string) (protoreflect.MessageDescriptor, error) {
	// 檢查緩存
	if desc, ok := s.messageCache[messageName]; ok {
		return desc, nil
	}

	// 讀取 proto 文件內容
	content, err := os.ReadFile(s.protoFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read proto file: %w", err)
	}

	// 編譯 proto 文件
	compiler := protocompile.Compiler{
		Resolver: &protocompile.SourceResolver{
			Accessor: protocompile.SourceAccessorFromMap(map[string]string{
				s.protoFilePath: string(content),
			}),
		},
	}

	fds, err := compiler.Compile(context.Background(), s.protoFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to compile proto file: %w", err)
	}

	if len(fds) == 0 {
		return nil, fmt.Errorf("no file descriptors found")
	}

	// 查找消息
	messages := fds[0].Messages()
	var msgDesc protoreflect.MessageDescriptor
	for i := 0; i < messages.Len(); i++ {
		md := messages.Get(i)
		if string(md.Name()) == messageName {
			msgDesc = md
			break
		}
	}

	if msgDesc == nil {
		return nil, fmt.Errorf("message %s not found in proto file", messageName)
	}

	// 緩存
	s.messageCache[messageName] = msgDesc

	return msgDesc, nil
}

// setField 設置動態消息的字段值
func (s *ProtoSerializer) setField(msg *dynamicpb.Message, fieldName string, value any) error {
	msgDesc := msg.Descriptor()
	fields := msgDesc.Fields()

	var fd protoreflect.FieldDescriptor
	for i := 0; i < fields.Len(); i++ {
		f := fields.Get(i)
		if string(f.Name()) == fieldName {
			fd = f
			break
		}
	}

	if fd == nil {
		// 字段不存在，跳過（允許 Excel 有額外欄位）
		return nil
	}

	// 處理 repeated 欄位
	if fd.IsList() {
		return s.setRepeatedField(msg, fd, value)
	}

	// 處理單個 message 類型欄位（嵌入的 Entity）
	if fd.Kind() == protoreflect.MessageKind {
		return s.setSingleMessageField(msg, fd, value)
	}

	// 根據字段類型轉換值
	convertedValue, err := s.convertValue(fd, value)
	if err != nil {
		return err
	}

	// 設置字段
	msg.Set(fd, convertedValue)

	return nil
}

// setSingleMessageField 設置單個 message 類型欄位的值（嵌入的 Entity）
func (s *ProtoSerializer) setSingleMessageField(msg *dynamicpb.Message, fd protoreflect.FieldDescriptor, value any) error {
	// value 應該是 map[string]any（從嵌入數據查找而來）
	data, ok := value.(map[string]any)
	if !ok {
		return fmt.Errorf("message field value must be map[string]any, got %T", value)
	}

	// 獲取嵌套 message 的描述符
	nestedMsgDesc := fd.Message()

	// 創建嵌套 message
	nestedMsg := dynamicpb.NewMessage(nestedMsgDesc)

	// 對 keys 排序以確保序列化結果穩定
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	// 設置每個欄位
	for _, fieldName := range keys {
		fieldValue := data[fieldName]
		if err := s.setField(nestedMsg, fieldName, fieldValue); err != nil {
			return fmt.Errorf("failed to set nested field %s: %w", fieldName, err)
		}
	}

	// 設置到父 message
	msg.Set(fd, protoreflect.ValueOfMessage(nestedMsg))

	return nil
}

// setRepeatedField 設置 repeated 欄位的值
func (s *ProtoSerializer) setRepeatedField(msg *dynamicpb.Message, fd protoreflect.FieldDescriptor, value any) error {
	// 檢查是否為 repeated message 類型（如 repeated LootStruct）
	if fd.Kind() == protoreflect.MessageKind {
		return s.setRepeatedMessageField(msg, fd, value)
	}

	// 以下處理 repeated scalar 類型
	// 獲取字串值
	strValue, ok := value.(string)
	if !ok {
		return fmt.Errorf("repeated field value must be a string, got %T", value)
	}

	strValue = strings.TrimSpace(strValue)
	if strValue == "" {
		return nil // 空值，不設置
	}

	// 解析 JSON 陣列
	var elements []json.RawMessage
	if err := json.Unmarshal([]byte(strValue), &elements); err != nil {
		return fmt.Errorf("failed to parse JSON array: %w", err)
	}

	// 獲取 list 並填充元素
	list := msg.Mutable(fd).List()
	for i, elem := range elements {
		// 將 RawMessage 轉換為字串（去除引號如果是字串類型）
		elemStr := strings.TrimSpace(string(elem))

		// 如果是 JSON 字串（有引號），去除引號
		if len(elemStr) >= 2 && elemStr[0] == '"' && elemStr[len(elemStr)-1] == '"' {
			var unquoted string
			if err := json.Unmarshal(elem, &unquoted); err != nil {
				return fmt.Errorf("failed to parse string element at index %d: %w", i, err)
			}
			elemStr = unquoted
		}

		// 轉換單一元素值
		convertedValue, err := s.convertScalarValue(fd.Kind(), elemStr)
		if err != nil {
			return fmt.Errorf("failed to convert element at index %d: %w", i, err)
		}

		list.Append(convertedValue)
	}

	return nil
}

// setRepeatedMessageField 設置 repeated message 欄位的值（如 repeated LootStruct）
func (s *ProtoSerializer) setRepeatedMessageField(msg *dynamicpb.Message, fd protoreflect.FieldDescriptor, value any) error {
	// value 應該是 []map[string]any（從 structFieldCollector.ToSlice() 來）
	// 或者是 JSON 字串
	var items []map[string]any

	switch v := value.(type) {
	case []map[string]any:
		items = v
	case string:
		// 嘗試解析 JSON 字串
		v = strings.TrimSpace(v)
		if v == "" {
			return nil
		}
		if err := json.Unmarshal([]byte(v), &items); err != nil {
			return fmt.Errorf("failed to parse repeated message value as JSON: %w", err)
		}
	default:
		return fmt.Errorf("repeated message field value must be []map[string]any or JSON string, got %T", value)
	}

	if len(items) == 0 {
		return nil
	}

	// 獲取嵌套 message 的描述符
	nestedMsgDesc := fd.Message()

	// 獲取 list 並填充元素
	list := msg.Mutable(fd).List()
	for i, item := range items {
		// 創建嵌套 message
		nestedMsg := dynamicpb.NewMessage(nestedMsgDesc)

		// 對 keys 排序以確保序列化結果穩定
		keys := make([]string, 0, len(item))
		for k := range item {
			keys = append(keys, k)
		}
		slices.Sort(keys)

		// 設置每個欄位
		for _, fieldName := range keys {
			fieldValue := item[fieldName]
			if err := s.setField(nestedMsg, fieldName, fieldValue); err != nil {
				return fmt.Errorf("failed to set nested field %s in item %d: %w", fieldName, i, err)
			}
		}

		list.Append(protoreflect.ValueOfMessage(nestedMsg))
	}

	return nil
}

// convertScalarValue 將字串轉換為指定類型的 protoreflect.Value
func (s *ProtoSerializer) convertScalarValue(kind protoreflect.Kind, strValue string) (protoreflect.Value, error) {
	switch kind {
	case protoreflect.DoubleKind:
		v, err := strconv.ParseFloat(strValue, 64)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfFloat64(v), nil

	case protoreflect.FloatKind:
		v, err := strconv.ParseFloat(strValue, 32)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfFloat32(float32(v)), nil

	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		v, err := strconv.ParseInt(strValue, 10, 32)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfInt32(int32(v)), nil

	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		v, err := strconv.ParseInt(strValue, 10, 64)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfInt64(v), nil

	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		v, err := strconv.ParseUint(strValue, 10, 32)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfUint32(uint32(v)), nil

	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		v, err := strconv.ParseUint(strValue, 10, 64)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfUint64(v), nil

	case protoreflect.BoolKind:
		v, err := strconv.ParseBool(strValue)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfBool(v), nil

	case protoreflect.StringKind:
		return protoreflect.ValueOfString(strValue), nil

	case protoreflect.BytesKind:
		return protoreflect.ValueOfBytes([]byte(strValue)), nil

	default:
		return protoreflect.ValueOfString(strValue), nil
	}
}

// convertValue 將 any 值轉換為 protobuf 字段需要的類型
func (s *ProtoSerializer) convertValue(fd protoreflect.FieldDescriptor, value any) (protoreflect.Value, error) {
	// 如果是字符串，需要根據目標類型轉換
	strValue, ok := value.(string)
	if !ok {
		// 如果不是字符串，嘗試轉換為 protoreflect.Value
		return protoreflect.ValueOf(value), nil
	}

	strValue = strings.TrimSpace(strValue)

	// 根據字段類型轉換
	switch fd.Kind() {
	case protoreflect.DoubleKind:
		v, err := strconv.ParseFloat(strValue, 64)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfFloat64(v), nil

	case protoreflect.FloatKind:
		v, err := strconv.ParseFloat(strValue, 32)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfFloat32(float32(v)), nil

	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		v, err := strconv.ParseInt(strValue, 10, 32)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfInt32(int32(v)), nil

	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		v, err := strconv.ParseInt(strValue, 10, 64)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfInt64(v), nil

	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		v, err := strconv.ParseUint(strValue, 10, 32)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfUint32(uint32(v)), nil

	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		v, err := strconv.ParseUint(strValue, 10, 64)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfUint64(v), nil

	case protoreflect.BoolKind:
		v, err := strconv.ParseBool(strValue)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfBool(v), nil

	case protoreflect.StringKind:
		return protoreflect.ValueOfString(strValue), nil

	case protoreflect.BytesKind:
		return protoreflect.ValueOfBytes([]byte(strValue)), nil

	default:
		// 對於其他類型，返回字符串
		return protoreflect.ValueOfString(strValue), nil
	}
}

// ValidateProtoFile 驗證 proto 文件是否存在且可解析
func (s *ProtoSerializer) ValidateProtoFile() error {
	if _, err := os.Stat(s.protoFilePath); os.IsNotExist(err) {
		return fmt.Errorf("proto file not found: %s", s.protoFilePath)
	}

	content, err := os.ReadFile(s.protoFilePath)
	if err != nil {
		return fmt.Errorf("failed to read proto file: %w", err)
	}

	// parser.Parse 需要非 nil 的 reporter.Handler，傳 nil 會在遇到第一個 token 時 panic
	_, err = parser.Parse(s.protoFilePath, strings.NewReader(string(content)), reporter.NewHandler(nil))
	if err != nil {
		return fmt.Errorf("failed to parse proto file: %w", err)
	}

	return nil
}

// Deserialize 將 protobuf bytes 反序列化為 []map[string]any
// wrapperName: 包裝 message 名稱（如 "EventInfos"）
// itemName: 項目 message 名稱（如 "EventInfo"）
// data: protobuf bytes（支援 gzip 壓縮，會自動檢測並解壓）
func (s *ProtoSerializer) Deserialize(wrapperName string, itemName string, data []byte) ([]map[string]any, error) {
	// 檢測並解壓 gzip
	if isGzipped(data) {
		var err error
		data, err = gunzip(data)
		if err != nil {
			return nil, fmt.Errorf("failed to decompress gzip data: %w", err)
		}
	}

	// 獲取包裝 message descriptor
	wrapperDesc, err := s.getMessageDescriptor(wrapperName)
	if err != nil {
		return nil, fmt.Errorf("failed to get wrapper message descriptor for %s: %w", wrapperName, err)
	}

	// 獲取項目 message descriptor
	itemDesc, err := s.getMessageDescriptor(itemName)
	if err != nil {
		return nil, fmt.Errorf("failed to get item message descriptor for %s: %w", itemName, err)
	}

	// 創建包裝消息並反序列化
	wrapperMsg := dynamicpb.NewMessage(wrapperDesc)
	if err := proto.Unmarshal(data, wrapperMsg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal protobuf data: %w", err)
	}

	// 找到 repeated 字段
	fields := wrapperDesc.Fields()
	var repeatedField protoreflect.FieldDescriptor
	for i := 0; i < fields.Len(); i++ {
		f := fields.Get(i)
		if f.IsList() && f.Message() != nil && f.Message().FullName() == itemDesc.FullName() {
			repeatedField = f
			break
		}
	}

	if repeatedField == nil {
		return nil, fmt.Errorf("no repeated field of type %s found in %s", itemName, wrapperName)
	}

	// 遍歷 list，轉換為 []map[string]any
	list := wrapperMsg.Get(repeatedField).List()
	result := make([]map[string]any, 0, list.Len())

	for i := 0; i < list.Len(); i++ {
		itemMsg := list.Get(i).Message()
		itemMap := s.messageToMap(itemMsg)
		result = append(result, itemMap)
	}

	return result, nil
}

// isGzipped 檢測數據是否為 gzip 壓縮格式
func isGzipped(data []byte) bool {
	// gzip magic bytes: 0x1f, 0x8b
	return len(data) >= 2 && data[0] == 0x1f && data[1] == 0x8b
}

// gunzip 解壓 gzip 數據
func gunzip(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return io.ReadAll(reader)
}

// messageToMap 將 protoreflect.Message 轉換為 map[string]any
func (s *ProtoSerializer) messageToMap(msg protoreflect.Message) map[string]any {
	result := make(map[string]any)

	msg.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		fieldName := string(fd.Name())
		result[fieldName] = s.convertFieldValue(fd, v)
		return true
	})

	return result
}

// convertFieldValue 將 protoreflect.Value 轉換為 Go 原生類型
func (s *ProtoSerializer) convertFieldValue(fd protoreflect.FieldDescriptor, v protoreflect.Value) any {
	// 處理 repeated 欄位
	if fd.IsList() {
		return s.convertListValue(fd, v.List())
	}

	// 處理 map 欄位
	if fd.IsMap() {
		return s.convertMapValue(fd, v.Map())
	}

	// 處理單一值
	return s.convertSingleValue(fd, v)
}

// convertSingleValue 轉換單一值
func (s *ProtoSerializer) convertSingleValue(fd protoreflect.FieldDescriptor, v protoreflect.Value) any {
	switch fd.Kind() {
	case protoreflect.BoolKind:
		return v.Bool()
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		return int32(v.Int())
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return v.Int()
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		return uint32(v.Uint())
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return v.Uint()
	case protoreflect.FloatKind:
		return float32(v.Float())
	case protoreflect.DoubleKind:
		return v.Float()
	case protoreflect.StringKind:
		return v.String()
	case protoreflect.BytesKind:
		return v.Bytes()
	case protoreflect.EnumKind:
		return int32(v.Enum())
	case protoreflect.MessageKind:
		return s.messageToMap(v.Message())
	default:
		return v.Interface()
	}
}

// convertListValue 轉換 repeated 欄位
func (s *ProtoSerializer) convertListValue(fd protoreflect.FieldDescriptor, list protoreflect.List) any {
	result := make([]any, list.Len())

	for i := 0; i < list.Len(); i++ {
		v := list.Get(i)
		if fd.Kind() == protoreflect.MessageKind {
			result[i] = s.messageToMap(v.Message())
		} else {
			result[i] = s.convertSingleValue(fd, v)
		}
	}

	return result
}

// convertMapValue 轉換 map 欄位
func (s *ProtoSerializer) convertMapValue(fd protoreflect.FieldDescriptor, m protoreflect.Map) any {
	result := make(map[string]any)

	m.Range(func(k protoreflect.MapKey, v protoreflect.Value) bool {
		keyStr := fmt.Sprintf("%v", k.Interface())
		if fd.MapValue().Kind() == protoreflect.MessageKind {
			result[keyStr] = s.messageToMap(v.Message())
		} else {
			result[keyStr] = v.Interface()
		}
		return true
	})

	return result
}
