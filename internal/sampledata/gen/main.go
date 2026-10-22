// Command gen 把 sampledata 的範例表寫成 xlsx 檔。
//
//	go run ./internal/sampledata/gen -out testdata
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/miskawu/formgen/internal/sampledata"
)

func main() {
	out := flag.String("out", "testdata", "輸出目錄")
	flag.Parse()

	paths, err := sampledata.WriteAll(*out)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for _, p := range paths {
		fmt.Println(p)
	}
}
