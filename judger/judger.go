package judger

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/docker/docker/pkg/stdcopy"
)

type Status int

const (
	AC Status = iota
	WA
	PE
	TLE
	MLE
	OLE
	CE
	RE
	SE
)

type Config struct {
	Points      int
	TimeLimit   int64 // ms
	MemoryLimit int64 // mbytes
}

type Result struct {
	Status Status
	Time   int64
	Detail string
}

func (s Status) String() string {
	return []string{"AC", "WA", "PE", "TLE", "MLE", "OLE", "CE", "RE", "SE"}[s]
}

func (r Result) String() string {
	msg := strings.Trim(r.Detail, " \t\r\n")
	if msg == "" {
		return fmt.Sprintf("%v(%vms)", r.Status, r.Time)
	}
	return fmt.Sprintf("%v(%vms: %v)", r.Status, r.Time, msg)
}

func test(jid string, tid string, env map[string]any, ttl time.Duration) (res Result) {
	var err error
	var judger, tester *exec
	judger, err = newExec(jid, nil, false, env)
	if err != nil {
		return Result{SE, 0, err.Error()}
	}
	tester, err = newExec(tid, nil, false, nil)
	if err != nil {
		return Result{SE, 0, err.Error()}
	}
	buf := &bytes.Buffer{}
	jch, tch := make(chan error), make(chan error)
	go func() {
		defer tester.eof()
		_, e := stdcopy.StdCopy(tester.Conn, buf, judger.Out)
		jch <- e
	}()
	go func() {
		defer judger.eof()
		_, e := stdcopy.StdCopy(judger.Conn, buf, tester.Out)
		tch <- e
	}()
	if e := judger.start(); e != nil {
		return Result{SE, 0, e.Error()}
	}
	defer clean(jid)
	judgeDone := func() {
		if r, e := judger.inspect(); e == nil {
			res.Status = min(Status(r.ExitCode), SE)
		} else {
			res.Status = SE
			res.Detail = e.Error()
		}
	}
	start := time.Now()
	if e := tester.start(); e != nil {
		return Result{SE, 0, e.Error()}
	}
	defer clean(tid)
	select {
	case <-time.NewTimer(ttl).C:
		res.Status = TLE
	case <-jch:
		judgeDone()
	case <-tch:
		r, _ := tester.inspect()
		if r.ExitCode == 137 { // OOM kill
			res.Status = MLE
		} else if r.ExitCode != 0 {
			res.Status = RE
		} else {
			select {
			case <-time.NewTimer(time.Second).C:
				res.Status = SE
				res.Detail = "judger timeout"
			case <-jch:
				judgeDone()
			}
		}
	}
	res.Time = int64(time.Since(start) / time.Millisecond)
	if res.Detail == "" {
		res.Detail = buf.String()
	}
	return
}

func Judge(conf Config, ptar io.Reader, utar io.Reader, con io.Writer) (res Result, ps []Result) {
	var err error
	var code, jid, tid string
	buf := &bytes.Buffer{}
	con = io.MultiWriter(con, buf)
	utar, code = tgzPick(utar, "source")
	if jid, err = start(ptar, con, 0, 0); err != nil {
		return Result{SE, 0, buf.String()}, nil
	}
	defer kill(jid)
	if tid, err = start(utar, con, 1e9, conf.MemoryLimit<<20); err != nil {
		return Result{CE, 0, buf.String()}, nil
	}
	defer kill(tid)
	ttl := time.Duration(conf.TimeLimit) * time.Millisecond
	for i := 0; i < conf.Points; i++ {
		env := map[string]any{"case": i, "code": code}
		p := test(jid, tid, env, ttl)
		res.Status = max(res.Status, p.Status)
		res.Time = max(res.Time, p.Time)
		log.Println("Case", i, p)
		ps = append(ps, p)
	}
	return
}
