package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/doveccl/DOJ/judger"
)

var cas = os.Getenv("case")
var num = regexp.MustCompile(`\d+`)

func quit(s judger.Status, v ...any) {
	fmt.Fprintln(os.Stderr, v...)
	os.Exit(int(s))
}

func main() {
	var in, ans *os.File
	if ds, e := os.ReadDir("."); e == nil {
		for _, d := range ds {
			if !d.IsDir() && num.FindString(d.Name()) == cas {
				switch {
				case strings.Contains(d.Name(), "in"):
					in, _ = os.Open(d.Name())
				case strings.Contains(d.Name(), "ou") || strings.Contains(d.Name(), "an"):
					ans, _ = os.Open(d.Name())
				}
			}
		}
	}
	if in == nil || ans == nil {
		quit(judger.SE, "unable to get data")
	}
	if _, e := io.Copy(os.Stdout, in); e != nil {
		quit(judger.SE, e)
	}
	asc, osc := bufio.NewScanner(ans), bufio.NewScanner(os.Stdin)
	for i := 0; asc.Scan() || osc.Scan(); i++ {
		if strings.TrimRight(asc.Text(), " \r\n") != strings.TrimRight(osc.Text(), " \r\n") {
			quit(judger.WA, "wrong answer at line", i+1)
		}
	}
}
