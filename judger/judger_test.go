package judger_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/docker/docker/pkg/archive"
	"github.com/doveccl/DOJ/judger"
)

func TestData(*testing.T) {
	ptar, _ := archive.Tar("../example/data", archive.Gzip)
	utar, _ := archive.Tar("../example/code", archive.Gzip)
	fmt.Println(judger.Judge(judger.Config{3, 1000, 6}, ptar, utar, os.Stdout))
}

func TestInter(*testing.T) {
	ptar, _ := archive.Tar("../example/interact", archive.Gzip)
	utar, _ := archive.Tar("../example/code", archive.Gzip)
	fmt.Println(judger.Judge(judger.Config{10, 1000, 6}, ptar, utar, os.Stdout))
}
