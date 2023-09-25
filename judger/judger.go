package judger

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"github.com/docker/docker/pkg/stdcopy"
)

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
		_, e := stdcopy.StdCopy(judger.Conn, io.Discard, tester.Out)
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
			fmt.Fprintln(buf, "Judger Inspect", e)
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
				fmt.Fprintln(buf, "Judger Timeout (1s)")
			case <-jch:
				judgeDone()
			}
		}
	}
	res.Time = int64(time.Since(start) / time.Millisecond)
	res.Detail = buf.String()
	return
}

func Judge(conf Config, ptar io.Reader, utar io.Reader, con io.Writer) (res Result, ps []Result) {
	var code, jid, tid string
	buf := &bytes.Buffer{}
	con = io.MultiWriter(con, buf)
	utar, code = tgzPick(utar, "source")
	if jid, res.Status = start(ptar, con, 0, 0, SE); res.Status > 0 {
		res.Detail = buf.String()
		return
	}
	defer kill(jid)
	if tid, res.Status = start(utar, con, 1e9, conf.MemoryLimit<<20, CE); res.Status > 0 {
		res.Detail = buf.String()
		return
	}
	defer kill(tid)
	ttl := time.Duration(conf.TimeLimit) * time.Millisecond
	for i := 0; i < conf.Points; i++ {
		env := map[string]any{"case": i, "code": code}
		p := test(jid, tid, env, ttl)
		res.Status = max(res.Status, p.Status)
		res.Time = max(res.Time, p.Time)
		fmt.Fprintf(con, "#%v %v\n", i, p)
		ps = append(ps, p)
	}
	return
}
