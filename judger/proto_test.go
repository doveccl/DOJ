package judger

import (
	"net"
	"testing"
)

func TestCodecRoundTrip(t *testing.T) {
	left, right := net.Pipe()
	defer left.Close()
	defer right.Close()

	sender := NewCodec(left)
	receiver := NewCodec(right)
	want := Message{
		Kind: MsgCompile,
		Compile: &CompileRequest{
			TaskID:      "task-1",
			UserCommand: "./main",
			Limits:      Limits{TimeMS: 1000, MemoryKB: 262144},
		},
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- sender.Send(want)
	}()

	got, err := receiver.Recv()
	if err != nil {
		t.Fatal(err)
	}
	if err := <-errCh; err != nil {
		t.Fatal(err)
	}
	if got.Kind != want.Kind || got.Compile == nil || got.Compile.UserCommand != "./main" {
		t.Fatalf("unexpected message: %#v", got)
	}
}
