package judger_test

import (
	"log"
	"os"
	"testing"

	"github.com/docker/docker/pkg/archive"
	"github.com/doveccl/DOJ/judger"
)

func TestData(*testing.T) {
	ptar, _ := archive.Tar("../testdata/a+b/data", archive.Gzip)
	utar, _ := archive.Tar("../testdata/a+b/code", archive.Gzip)
	log.Println(judger.Judge(judger.Config{3, 1000, 6}, ptar, utar, os.Stdout))
}

func TestInter(*testing.T) {
	ptar, _ := archive.Tar("../testdata/a+b/spj", archive.Gzip)
	utar, _ := archive.Tar("../testdata/a+b/code", archive.Gzip)
	log.Println(judger.Judge(judger.Config{10, 1000, 6}, ptar, utar, os.Stdout))
}

func TestQuine(*testing.T) {
	ptar, _ := archive.Tar("../testdata/quine/spj", archive.Gzip)
	utar, _ := archive.Tar("../testdata/quine/code", archive.Gzip)
	log.Println(judger.Judge(judger.Config{1, 1000, 6}, ptar, utar, os.Stdout))
}

func TestCE(*testing.T) {
	ptar, _ := archive.Tar("../testdata/a+b/data", archive.Gzip)
	utar, _ := archive.Tar("../testdata/a+b/ce", archive.Gzip)
	log.Println(judger.Judge(judger.Config{0, 1000, 6}, ptar, utar, os.Stdout))
}
