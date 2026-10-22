package models

import (
	"gorm.io/gorm"
)

// Snapshot 是某張資料表在某個命名空間下的最新序列化內容。
//
// (Namespace, Format, Name) 唯一決定一份快照；內容相同（Checksum 不變）時不會改寫。
type Snapshot struct {
	Namespace string `gorm:"type:varchar(32);index:idx_snapshot_lookup,priority:1"` // 命名空間，用來區分不同資料集或環境
	Format    string `gorm:"type:varchar(16);index:idx_snapshot_lookup,priority:2"` // 序列化格式，例如 "protobuf+gzip"
	Name      string `gorm:"type:varchar(32);index:idx_snapshot_lookup,priority:3"` // 資料表（Entity）名稱
	Checksum  string `gorm:"type:varchar(32)"`                                      // Data 的 MD5
	Data      []byte `gorm:"type:blob"`                                             // 序列化後的內容
	gorm.Model
}
