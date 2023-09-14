package main

import (
	"archive/tar"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

var docker *client.Client
var tag = "doj/test:1"
var ctx = context.Background()

func buildTar(dockerfile string) (*bytes.Buffer, error) {
	buf := &bytes.Buffer{}
	tw := tar.NewWriter(buf)
	defer tw.Close()
	if e := tw.WriteHeader(&tar.Header{Name: "Dockerfile", Size: int64(len(dockerfile))}); e != nil {
		return nil, e
	}
	if _, e := tw.Write([]byte(dockerfile)); e != nil {
		return nil, e
	}
	return buf, nil
}

func buildImg(tag string, tar io.Reader, src string) error {
	args := map[string]*string{"SRC": &src}
	opts := types.ImageBuildOptions{Tags: []string{tag}, Remove: true, ForceRemove: true, BuildArgs: args}
	r, e := docker.ImageBuild(ctx, tar, opts)
	if e != nil {
		return e
	}
	sc := bufio.NewScanner(r.Body)
	for sc.Scan() {
		var res map[string]any
		if json.Unmarshal(sc.Bytes(), &res) == nil {
			switch {
			case res["stream"] != nil:
				fmt.Print(res["stream"])
			case res["error"] != nil:
				return fmt.Errorf("%v", res["error"])
			}
		}
	}
	if _, e := docker.ImagesPrune(ctx, filters.Args{}); e != nil {
		return e
	}
	return nil
}

func newPod(img string, mem int64) (string, error) {
	c, e := docker.ContainerCreate(ctx, &container.Config{
		Image:           img,
		NetworkDisabled: true,
		OpenStdin:       true,
		AttachStdin:     true,
		AttachStdout:    true,
	}, &container.HostConfig{
		AutoRemove: true,
		Resources:  container.Resources{Memory: mem},
	}, nil, nil, "")
	if e != nil {
		return "", e
	}
	return c.ID, nil
}

func main() {
	cli, err := client.NewClientWithOpts(client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	} else {
		docker = cli
	}

	ctar, err := buildTar(`
		FROM gcc
		WORKDIR /src
		ARG SRC
		RUN echo "$SRC" > main.cc
		RUN g++ -Wall -static -O2 main.cc -o main
		FROM busybox
		WORKDIR /test
		COPY --from=0 /src/main .
		CMD ["./main"]
	`)
	if err != nil {
		panic(err)
	}

	if err := buildImg(tag, ctar, `
		#include <iostream>
		using namespace std;
		int main() {
			int a, b;
			cin >> a >> b;
			cout << a + b << endl;
		}
	`); err != nil {
		panic(err)
	}

	cid, err := newPod(tag, 64<<20) //64MB
	if err != nil {
		panic(err)
	}
	fmt.Println("Container ID", cid)

	hi, err := docker.ContainerAttach(ctx, cid, types.ContainerAttachOptions{Stdin: true, Stdout: true, Stream: true})
	if err != nil {
		panic(err)
	}

	go func() {
		if _, e := io.Copy(os.Stdout, hi.Reader); e != nil {
			panic(e)
		}
	}()

	if e := docker.ContainerStart(ctx, cid, types.ContainerStartOptions{}); e != nil {
		panic(e)
	}

	fmt.Println(hi.Conn.Write([]byte("123 456\n")))

	resCh, errCh := docker.ContainerWait(ctx, cid, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		fmt.Println(err)
	case res := <-resCh:
		fmt.Println(res.StatusCode, res.Error)
	}

	fmt.Println(docker.ImageRemove(ctx, tag, types.ImageRemoveOptions{Force: true}))
}
