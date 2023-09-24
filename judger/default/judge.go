package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

const (
	WA = 1
	SE = 8
)

var cas = os.Getenv("case")
var num = regexp.MustCompile(`\d+`)

func quit(s int, v ...any) {
	fmt.Fprintln(os.Stderr, v...)
	os.Exit(s)
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
		quit(SE, "unable to get data")
	}
	if _, e := io.Copy(os.Stdout, in); e != nil {
		quit(SE, e)
	}
	asc, osc := bufio.NewScanner(ans), bufio.NewScanner(os.Stdin)
	for i := 0; ; i++ { // do not use `asc.Scan() || osc.Scan()`
		ok1, ok2 := asc.Scan(), osc.Scan()
		if !ok1 && !ok2 {
			break
		}
		if strings.TrimRight(asc.Text(), " \r\n") != strings.TrimRight(osc.Text(), " \r\n") {
			quit(WA, "wrong answer at line", i+1)
		}
	}
}
