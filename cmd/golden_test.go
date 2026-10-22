package cmd

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/miskawu/formgen/internal/config"
)

var update = flag.Bool("update", false, "用目前的輸出覆寫 testdata/ 中的 golden 檔")

const testdataDir = "../testdata"

// TestGolden_SamplePipeline 用 testdata/Demo_*.xlsx 跑完整條生成流程，
// 並與 testdata/ 中的 golden 檔比對：
//
//	Excel → sample.yaml → sample.sql / sample.go / sample.proto / sample_envelope.go
//
// 修改生成器或範例表後，用 `go test ./cmd -run TestGolden -update` 更新 golden 檔。
func TestGolden_SamplePipeline(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "formgen.yaml")
	if err := os.WriteFile(cfgPath, []byte(config.DefaultConfig()), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := config.Init(cfgPath); err != nil {
		t.Fatalf("config.Init() error = %v", err)
	}

	files, err := filepath.Glob(filepath.Join(testdataDir, "Demo_*.xlsx"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no sample xlsx found (run `go generate ./internal/sampledata`): %v", err)
	}

	outDir := t.TempDir()
	if *update {
		outDir = testdataDir
	}
	out := func(name string) string { return filepath.Join(outDir, name) }

	// Excel → YAML Schema
	schemaOutputPath = out("sample.yaml")
	if err := runSchema(files); err != nil {
		t.Fatalf("runSchema() error = %v", err)
	}
	assertGolden(t, "sample.yaml", out("sample.yaml"))

	// YAML Schema → SQL DDL
	sqlInput, sqlOutput = out("sample.yaml"), out("sample.sql")
	if err := runSql(); err != nil {
		t.Fatalf("runSql() error = %v", err)
	}
	assertGolden(t, "sample.sql", out("sample.sql"))

	// YAML Schema → Proto（含 List wrapper 與 TableEnvelope / TableBundle）
	protoInput, protoOutput = out("sample.yaml"), out("sample.proto")
	protoPackageName, protoGoPackage = "sample", "example.com/sample/pb;pb"
	protoEnvelope, protoWrapper, protoPlugin = true, false, ""
	if err := runProto(); err != nil {
		t.Fatalf("runProto() error = %v", err)
	}
	assertGolden(t, "sample.proto", out("sample.proto"))

	// YAML Schema → GORM model（含 ToProto）
	ormInput, ormOutput = out("sample.yaml"), out("sample.go")
	ormPackageName, ormProtoPath, ormProtoAlias = "models", "example.com/sample/pb", "pb"
	if err := runOrm(); err != nil {
		t.Fatalf("runOrm() error = %v", err)
	}
	assertGolden(t, "sample.go", out("sample.go"))

	// YAML Schema → envelope 包裝函數
	wrapperInput, wrapperOutput = out("sample.yaml"), out("sample_envelope.go")
	wrapperPackageName, wrapperProtoPath, wrapperProtoAlias = "envelope", "example.com/sample/pb", "pb"
	wrapperFuncName, wrapperEnvelope = "", ""
	if err := runWrapper(); err != nil {
		t.Fatalf("runWrapper() error = %v", err)
	}
	assertGolden(t, "sample_envelope.go", out("sample_envelope.go"))
}

func assertGolden(t *testing.T, name, gotPath string) {
	t.Helper()
	if *update {
		return
	}
	got, err := os.ReadFile(gotPath)
	if err != nil {
		t.Fatalf("read output %s: %v", gotPath, err)
	}
	want, err := os.ReadFile(filepath.Join(testdataDir, name))
	if err != nil {
		t.Fatalf("read golden %s: %v (run `go test ./cmd -run TestGolden -update`)", name, err)
	}
	if string(got) != string(want) {
		t.Errorf("%s does not match golden file (run `go test ./cmd -run TestGolden -update` and review the diff)\n--- got ---\n%s", name, got)
	}
}
