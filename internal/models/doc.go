// Package models 定義 formgen 在資料庫中自行維護的資料表。
//
// 啟用 `db import --snapshot` 時，importer 會把每張表序列化成 protobuf（gzip 壓縮）
// 存進 Snapshot，並在內容變動時追加一筆 Version，方便下游服務比對 checksum
// 決定是否需要重新拉取資料。未啟用快照時不會用到這些表。
package models
