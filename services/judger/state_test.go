package judger

import (
	"testing"
	"time"

	"github.com/doveccl/doj/utils"
)

func TestJudgerStatusWindow(t *testing.T) {
	startRedis(t)
	utils.ResetCacheForTest()
	t.Cleanup(utils.ResetCacheForTest)
	now := time.Now()
	id := uint(now.UnixNano()%1_000_000_000) + 1_000_000_000

	TouchStatus(t.Context(), id, now)
	status := ReadStatus(t.Context(), id, now.Add(3*time.Second))
	if !status.Online || status.UptimeSeconds != 3 {
		t.Fatalf("fresh status = %+v", status)
	}

	status = ReadStatus(t.Context(), id, now.Add(11*time.Second))
	if status.Online || status.UptimeSeconds != 0 {
		t.Fatalf("expired status = %+v", status)
	}

	TouchStatus(t.Context(), id, now.Add(12*time.Second))
	status = ReadStatus(t.Context(), id, now.Add(13*time.Second))
	if !status.Online || status.UptimeSeconds != 1 {
		t.Fatalf("reconnected status = %+v", status)
	}
}
