package sampledata

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/miskawu/formgen/internal/source"
	"github.com/miskawu/formgen/internal/source/excel"
)

const testdataDir = "../../testdata"

// trimRow 去掉列尾的空字串（excelize 讀回時不會保留列尾空白儲存格）
func trimRow(row []string) []string {
	end := len(row)
	for end > 0 && row[end-1] == "" {
		end--
	}
	return row[:end]
}

func trimRows(rows [][]string) [][]string {
	out := make([][]string, len(rows))
	for i, row := range rows {
		out[i] = trimRow(row)
	}
	return out
}

func TestTables_HeaderIsWellFormed(t *testing.T) {
	seen := map[string]bool{}
	for _, table := range Tables() {
		if seen[table.Name] {
			t.Errorf("duplicate table name %s", table.Name)
		}
		seen[table.Name] = true

		if len(table.Rows) <= source.DefaultLayout().DataRow {
			t.Errorf("%s: need header rows plus at least one data row", table.Name)
			continue
		}
		width := len(table.Rows[1])
		for r := 0; r < 5; r++ {
			if len(table.Rows[r]) != width {
				t.Errorf("%s: header row %d has %d cells, want %d", table.Name, r, len(table.Rows[r]), width)
			}
		}
		for r, row := range table.Rows[5:] {
			if len(row) > width {
				t.Errorf("%s: data row %d is wider than header", table.Name, r)
			}
		}

		if _, err := source.NewParser().Parse(source.Sheet{Name: table.Name, Rows: table.Rows}); err != nil {
			t.Errorf("%s: parse failed: %v", table.Name, err)
		}
	}
}

// TestBuild_IsReproducible 確認同樣的表格每次產生的 xlsx 位元組完全相同
func TestBuild_IsReproducible(t *testing.T) {
	table := Tables()[0]
	render := func() []byte {
		f, err := Build(table)
		if err != nil {
			t.Fatalf("Build() error = %v", err)
		}
		defer f.Close()
		buf, err := f.WriteToBuffer()
		if err != nil {
			t.Fatalf("WriteToBuffer() error = %v", err)
		}
		return buf.Bytes()
	}
	if !bytes.Equal(render(), render()) {
		t.Error("Build() output is not byte-for-byte reproducible")
	}
}

// TestCommittedFilesUpToDate 確認 testdata/ 裡的 xlsx 與 Tables() 一致。
// 失敗時請執行 `go generate ./internal/sampledata` 重新產生。
func TestCommittedFilesUpToDate(t *testing.T) {
	loader := excel.NewLoader()
	for _, table := range Tables() {
		t.Run(table.Name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(testdataDir, table.FileName()))
			if err != nil {
				t.Fatalf("read %s: %v (run `go generate ./internal/sampledata`)", table.FileName(), err)
			}
			sheets, err := loader.Load(data)
			if err != nil {
				t.Fatalf("load: %v", err)
			}
			if len(sheets) != 2 {
				t.Fatalf("got %d sheets, want 2", len(sheets))
			}
			if sheets[0].Name != table.Name || sheets[1].Name != NotesSheetName {
				t.Errorf("sheet names = [%s %s], want [%s %s]", sheets[0].Name, sheets[1].Name, table.Name, NotesSheetName)
			}
			if got, want := trimRows(sheets[0].Rows), trimRows(table.Rows); !reflect.DeepEqual(got, want) {
				t.Errorf("sheet %s is out of date (run `go generate ./internal/sampledata`)\ngot:  %q\nwant: %q", table.Name, got, want)
			}
			if got, want := trimRows(sheets[1].Rows), trimRows(table.NotesRows()); !reflect.DeepEqual(got, want) {
				t.Errorf("notes sheet is out of date (run `go generate ./internal/sampledata`)")
			}
		})
	}
}

func TestWriteAll(t *testing.T) {
	dir := t.TempDir()
	paths, err := WriteAll(dir)
	if err != nil {
		t.Fatalf("WriteAll() error = %v", err)
	}
	if len(paths) != len(Tables()) {
		t.Fatalf("WriteAll() wrote %d files, want %d", len(paths), len(Tables()))
	}
	for _, p := range paths {
		got, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		want, err := os.ReadFile(filepath.Join(testdataDir, filepath.Base(p)))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("%s differs from committed copy (run `go generate ./internal/sampledata`)", filepath.Base(p))
		}
	}
}
