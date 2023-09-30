package judger

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/docker/docker/pkg/stdcopy"
)

func test(jid string, tid string, ttl time.Duration, env []string) (res Result) {
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
			case <-time.NewTimer(2 * time.Second).C:
				res.Status = SE
				fmt.Fprintln(buf, "Judger Timeout (2s)")
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
	defer func() { kill(jid, tid) }()
	utar, code = tgzPick(utar, "source")
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	buildCh := make(chan Result)
	go func() {
		buf := &bytes.Buffer{}
		r := start(ctx, ptar, buf, 0, 0, SE)
		jid, r.Detail = r.Detail, buf.String()
		buildCh <- r
	}()
	go func() {
		buf := &bytes.Buffer{}
		r := start(ctx, utar, buf, 1e9, conf.MemoryLimit<<20, CE)
		tid, r.Detail = r.Detail, buf.String()
		buildCh <- r
	}()
	for i := 0; i < 2; i++ {
		select {
		case <-ctx.Done():
			return Result{SE, 0, "Build Timeout (10s)"}, nil
		case r := <-buildCh:
			fmt.Fprintln(con, r.Detail)
			if r.Status != OK {
				return r, nil
			}
		}
	}
	ttl := time.Duration(conf.TimeLimit) * time.Millisecond
	for i := 0; i < conf.Points; i++ {
		p := test(jid, tid, ttl, []string{fmt.Sprintf("case=%v", i), "code=" + code})
		res.Status = max(res.Status, p.Status)
		res.Time = max(res.Time, p.Time)
		fmt.Fprintf(con, "Test #%v %v\n", i, p)
		ps = append(ps, p)
	}
	return
}
