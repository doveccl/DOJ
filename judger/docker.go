package judger

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
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
	Status string              `json:"status"`
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

func start(tar io.Reader, con io.Writer, cpu, mem int64) (string, error) {
	iid, opt := "", types.ImageBuildOptions{Remove: true, ForceRemove: true}
	res, err := cli.ImageBuild(ctx, tar, opt)
	if err != nil {
		return "", err
	}

	for sc := bufio.NewScanner(res.Body); sc.Scan(); {
		o := &buildOut{}
		if json.Unmarshal(sc.Bytes(), o) == nil {
			switch {
			case o.Aux.ID != "":
				iid = o.Aux.ID
			case o.Stream != "":
				if _, e := con.Write([]byte(o.Stream)); e != nil {
					log.Print(o.Stream)
				}
			case o.Error != "":
				return "", errors.New(o.Error)
			}
		}
	}

	var c container.CreateResponse
	cconf := &container.Config{Image: iid, NetworkDisabled: true, Cmd: []string{"sleep", "infinity"}}
	hconf := &container.HostConfig{Resources: container.Resources{NanoCPUs: cpu, Memory: mem}}
	if c, err = cli.ContainerCreate(ctx, cconf, hconf, nil, nil, ""); err == nil {
		if err = cli.ContainerStart(ctx, c.ID, types.ContainerStartOptions{}); err == nil {
			return c.ID, nil
		}
	}

	return "", err
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
