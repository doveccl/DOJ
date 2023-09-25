package judger

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

type buildOut struct {
	Aux    struct{ ID string } `json:"aux"`
	Error  string              `json:"error"`
	Stream string              `json:"stream"`
}

type exec struct {
	ID   string
	Conn net.Conn
	Out  *bufio.Reader
}

var cli *client.Client
var ctx = context.Background()

func init() {
	if c, e := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation()); e == nil {
		cli = c
	} else {
		panic(e)
	}
}

func start(tar io.Reader, con io.Writer, cpu, mem int64, ceflag Status) (id string, s Status) {
	r, e := cli.ImageBuild(ctx, tar, types.ImageBuildOptions{Remove: true, ForceRemove: true})
	if e != nil {
		fmt.Fprintln(con, e)
		return "", SE
	}

	for sc := bufio.NewScanner(r.Body); sc.Scan(); {
		o := &buildOut{}
		if json.Unmarshal(sc.Bytes(), o) == nil {
			switch {
			case o.Aux.ID != "":
				id = o.Aux.ID
			case o.Stream != "":
				fmt.Fprint(con, o.Stream)
			case o.Error != "":
				fmt.Fprintln(con, o.Error)
				return "", ceflag
			}
		}
	}

	var c container.CreateResponse
	cconf := &container.Config{Image: id, NetworkDisabled: true, Cmd: []string{"sleep", "infinity"}}
	hconf := &container.HostConfig{Resources: container.Resources{NanoCPUs: cpu, Memory: mem}}
	if c, e = cli.ContainerCreate(ctx, cconf, hconf, nil, nil, ""); e == nil {
		if e = cli.ContainerStart(ctx, c.ID, types.ContainerStartOptions{}); e == nil {
			return c.ID, AC
		}
	}

	fmt.Fprintln(con, e)
	return "", SE
}

func newExec(cid string, cmd []string, root bool, env map[string]any) (x *exec, e error) {
	conf := types.ExecConfig{AttachStdin: true, AttachStdout: true, AttachStderr: true, Env: envList(env)}
	if !root {
		conf.User = "65534"
	}
	if len(cmd) > 0 {
		conf.Cmd = cmd
	} else if c, e := cli.ContainerInspect(ctx, cid); e == nil {
		if i, _, e := cli.ImageInspectWithRaw(ctx, c.Image); e == nil {
			conf.Cmd = i.Config.Cmd
		}
	}

	var r types.IDResponse
	var h types.HijackedResponse
	if r, e = cli.ContainerExecCreate(ctx, cid, conf); e == nil {
		if h, e = cli.ContainerExecAttach(ctx, r.ID, types.ExecStartCheck{}); e == nil {
			x = &exec{ID: r.ID, Out: h.Reader, Conn: h.Conn}
		}
	}
	return
}

func (e *exec) start() error {
	return cli.ContainerExecStart(ctx, e.ID, types.ExecStartCheck{})
}

func (e *exec) inspect() (types.ContainerExecInspect, error) {
	return cli.ContainerExecInspect(ctx, e.ID)
}

func (e *exec) eof() {
	if v, ok := e.Conn.(interface{ CloseWrite() error }); ok {
		_ = v.CloseWrite()
	} else { // will close output either
		e.Conn.Close()
	}
}

func clean(cid string) {
	cmd := []string{"sh", "-c", "ps -o pid= | grep -v 1 | xargs kill -9"}
	if x, e := newExec(cid, cmd, true, nil); e == nil {
		if e = x.start(); e != nil {
			log.Println(e)
		}
	}
}

func kill(cid string) {
	if cli.ContainerRemove(ctx, cid, types.ContainerRemoveOptions{Force: true}) == nil {
		if _, e := cli.ImagesPrune(ctx, filters.Args{}); e != nil {
			log.Println(e)
		}
	}
}
