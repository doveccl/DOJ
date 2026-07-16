package judger

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	common "github.com/doveccl/doj/contract/judger"
)

func TestProgressReporterThrottlesUpdatesAndKeepsProgressAlive(t *testing.T) {
	var got []common.HeartbeatRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req common.HeartbeatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		got = append(got, req)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	reporter := &progressReporter{client: server.Client(), cfg: WorkerConfig{Server: server.URL}, taskID: 7, submissionID: 11, attempt: 2}
	reporter.Set(t.Context(), "download", 0, nil)
	reporter.Update("download", 10, nil)
	reporter.Flush(t.Context())
	if len(got) != 1 {
		t.Fatalf("early progress update sent %d heartbeats", len(got))
	}

	reporter.sentAt = time.Now().Add(-progressUpdateInterval)
	reporter.Flush(t.Context())
	if len(got) != 2 || got[1].Stage != "download" || got[1].Done != 10 {
		t.Fatalf("progress heartbeat = %+v", got)
	}

	reporter.sentAt = time.Now().Add(-progressKeepaliveInterval)
	reporter.Flush(t.Context())
	if len(got) != 3 || got[2].Stage != "download" || got[2].Done != 10 {
		t.Fatalf("keepalive heartbeat = %+v", got)
	}
}
