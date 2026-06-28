package judger

import (
	"archive/tar"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	dockerbuild "github.com/docker/docker/api/types/build"
	dockercontainer "github.com/docker/docker/api/types/container"
	dockerimage "github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/docker/client"
	"github.com/docker/docker/errdefs"
	"github.com/docker/docker/pkg/stdcopy"
)

type dockerHostConfig struct {
	Binds       []string
	NetworkMode string
	SecurityOpt []string
	CapDrop     []string
	CapAdd      []string
}

type dockerCreateRequest struct {
	Image      string
	Cmd        []string
	WorkingDir string
	HostConfig dockerHostConfig
}

func dockerBuildImage(ctx context.Context, dir string, dockerfile string, outputLimit int64) (string, string, error) {
	return dockerBuildImageTimed(ctx, dir, dockerfile, outputLimit, dockerBuildTiming{})
}

type dockerBuildTiming struct {
	Logf         func(format string, args ...any)
	SubmissionID uint
	Attempt      int
}

func dockerBuildImageTimed(ctx context.Context, dir string, dockerfile string, outputLimit int64, timing dockerBuildTiming) (string, string, error) {
	clientStartedAt := time.Now()
	cli, err := newDockerClient()
	logStep(timing.Logf, timing.SubmissionID, timing.Attempt, "docker_build_client", clientStartedAt)
	if err != nil {
		return "", "", err
	}
	defer cli.Close()

	tarStartedAt := time.Now()
	body, err := tarDirectory(dir)
	logTask(timing.Logf, timing.SubmissionID, timing.Attempt, "docker_build_context_tar=%s bytes=%d", formatDuration(time.Since(tarStartedAt)), len(body))
	if err != nil {
		return "", "", err
	}
	tag := tempDockerTag()
	requestStartedAt := time.Now()
	resp, err := cli.ImageBuild(ctx, bytes.NewReader(body), dockerbuild.ImageBuildOptions{
		Tags:        []string{tag},
		Dockerfile:  dockerfile,
		NetworkMode: "none",
		Remove:      true,
		ForceRemove: true,
	})
	logStep(timing.Logf, timing.SubmissionID, timing.Attempt, "docker_build_request", requestStartedAt)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	streamStartedAt := time.Now()
	output, err := readBuildStream(resp.Body, outputLimit)
	logTask(timing.Logf, timing.SubmissionID, timing.Attempt, "docker_build_stream=%s output_bytes=%d", formatDuration(time.Since(streamStartedAt)), len(output))
	if err != nil {
		return "", output, err
	}
	return tag, output, nil
}

func dockerCreateContainer(ctx context.Context, req dockerCreateRequest) (string, error) {
	cli, err := newDockerClient()
	if err != nil {
		return "", err
	}
	defer cli.Close()

	got, err := cli.ContainerCreate(ctx, &dockercontainer.Config{
		Image:      req.Image,
		Cmd:        strslice.StrSlice(req.Cmd),
		WorkingDir: req.WorkingDir,
	}, &dockercontainer.HostConfig{
		Binds:       req.HostConfig.Binds,
		NetworkMode: dockercontainer.NetworkMode(req.HostConfig.NetworkMode),
		SecurityOpt: req.HostConfig.SecurityOpt,
		CapDrop:     strslice.StrSlice(req.HostConfig.CapDrop),
		CapAdd:      strslice.StrSlice(req.HostConfig.CapAdd),
	}, nil, nil, "")
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(got.ID) == "" {
		return "", fmt.Errorf("docker create returned an empty container id")
	}
	return strings.TrimSpace(got.ID), nil
}

func dockerImageCmd(ctx context.Context, image string) ([]string, error) {
	cli, err := newDockerClient()
	if err != nil {
		return nil, err
	}
	defer cli.Close()
	got, _, err := cli.ImageInspectWithRaw(ctx, image)
	if err != nil {
		return nil, err
	}
	return append([]string(nil), got.Config.Cmd...), nil
}

func dockerEnsureImage(ctx context.Context, image string) (bool, error) {
	cli, err := newDockerClient()
	if err != nil {
		return false, err
	}
	defer cli.Close()
	if _, _, err := cli.ImageInspectWithRaw(ctx, image); err != nil {
		if !errdefs.IsNotFound(err) {
			return false, err
		}
		reader, err := cli.ImagePull(ctx, image, dockerimage.PullOptions{})
		if err != nil {
			return false, err
		}
		defer reader.Close()
		if _, err := io.Copy(io.Discard, reader); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func dockerStartContainer(ctx context.Context, id string) error {
	cli, err := newDockerClient()
	if err != nil {
		return err
	}
	defer cli.Close()
	return cli.ContainerStart(ctx, id, dockercontainer.StartOptions{})
}

func dockerWaitContainer(ctx context.Context, id string) (int, error) {
	cli, err := newDockerClient()
	if err != nil {
		return 0, err
	}
	defer cli.Close()

	statusCh, errCh := cli.ContainerWait(ctx, id, dockercontainer.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return 0, err
		}
		return 0, fmt.Errorf("docker wait ended without a status")
	case status := <-statusCh:
		return int(status.StatusCode), nil
	case <-ctx.Done():
		return 0, ctx.Err()
	}
}

func dockerRemoveContainer(ctx context.Context, id string) {
	cleanupCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	cli, err := newDockerClient()
	if err != nil {
		return
	}
	defer cli.Close()
	err = cli.ContainerRemove(cleanupCtx, id, dockercontainer.RemoveOptions{Force: true})
	if err != nil && !errdefs.IsNotFound(err) {
		return
	}
}

func dockerRemoveImage(ctx context.Context, id string) {
	cleanupCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	cli, err := newDockerClient()
	if err != nil {
		return
	}
	defer cli.Close()
	_, err = cli.ImageRemove(cleanupCtx, id, dockerimage.RemoveOptions{Force: true, PruneChildren: true})
	if err != nil && !errdefs.IsNotFound(err) {
		return
	}
}

func dockerInspectPID(ctx context.Context, id string) (int, error) {
	cli, err := newDockerClient()
	if err != nil {
		return 0, err
	}
	defer cli.Close()
	got, err := cli.ContainerInspect(ctx, id)
	if err != nil {
		return 0, err
	}
	if got.State == nil || got.State.Pid <= 0 {
		return 0, fmt.Errorf("invalid container pid")
	}
	return got.State.Pid, nil
}

func dockerLogs(ctx context.Context, id string, outputLimit int64) string {
	cli, err := newDockerClient()
	if err != nil {
		return ""
	}
	defer cli.Close()
	reader, err := cli.ContainerLogs(ctx, id, dockercontainer.LogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		return ""
	}
	defer reader.Close()
	output := &limitBuffer{limit: outputLimit + 1}
	_, _ = stdcopy.StdCopy(output, output, reader)
	return strings.TrimSpace(output.String())
}

func dockerCopyFile(ctx context.Context, id string, source string, target string) error {
	cli, err := newDockerClient()
	if err != nil {
		return err
	}
	defer cli.Close()
	reader, _, err := cli.CopyFromContainer(ctx, id, source)
	if err != nil {
		return err
	}
	defer reader.Close()
	return extractSingleFile(reader, target, source)
}

func dockerPing(ctx context.Context) error {
	cli, err := newDockerClient()
	if err != nil {
		return err
	}
	defer cli.Close()
	_, err = cli.Ping(ctx)
	return err
}

func newDockerClient() (*client.Client, error) {
	return client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
}

func tempDockerTag() string {
	return fmt.Sprintf("doj-build:%d-%d", os.Getpid(), time.Now().UnixNano())
}

func readBuildStream(reader io.Reader, outputLimit int64) (string, error) {
	output := &limitBuffer{limit: outputLimit + 1}
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64*1024), int(outputLimit)+64*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		var item struct {
			Stream      string `json:"stream"`
			Status      string `json:"status"`
			Progress    string `json:"progress"`
			Error       string `json:"error"`
			ErrorDetail *struct {
				Message string `json:"message"`
			} `json:"errorDetail"`
		}
		if err := json.Unmarshal(line, &item); err != nil {
			_, _ = output.Write(line)
			_, _ = output.Write([]byte("\n"))
			continue
		}
		if item.ErrorDetail != nil && item.ErrorDetail.Message != "" {
			_, _ = output.Write([]byte(item.ErrorDetail.Message))
			return output.String(), fmt.Errorf("%s", item.ErrorDetail.Message)
		}
		if item.Error != "" {
			_, _ = output.Write([]byte(item.Error))
			return output.String(), fmt.Errorf("%s", item.Error)
		}
		for _, value := range []string{item.Stream, item.Status, item.Progress} {
			if value != "" {
				_, _ = output.Write([]byte(value))
			}
		}
		if output.overflow || int64(output.Len()) > outputLimit {
			return output.String(), fmt.Errorf("docker build output limit exceeded")
		}
	}
	if err := scanner.Err(); err != nil {
		return output.String(), err
	}
	return output.String(), nil
}

func tarDirectory(dir string) ([]byte, error) {
	var buffer bytes.Buffer
	writer := tar.NewWriter(&buffer)
	err := filepath.WalkDir(dir, func(file string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(dir, file)
		if err != nil {
			return err
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(rel)
		if err := writer.WriteHeader(header); err != nil {
			return err
		}
		input, err := os.Open(file)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(writer, input)
		closeErr := input.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
	if err != nil {
		_ = writer.Close()
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func extractSingleFile(reader io.Reader, target string, source string) error {
	tarReader := tar.NewReader(reader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeRegA {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		file, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o700)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(file, tarReader)
		closeErr := file.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	}
	return fmt.Errorf("docker archive did not contain %s", source)
}
