package judger

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/doveccl/doj/utils"
)

const (
	stateOnlineWindow = 10 * time.Second
	stateTTL          = 24 * time.Hour
)

type Status struct {
	Online        bool       `json:"online"`
	ConnectedAt   *time.Time `json:"connectedAt"`
	ActiveAt      *time.Time `json:"activeAt"`
	UptimeSeconds int        `json:"uptimeSeconds"`
}

type stateRecord struct {
	ConnectedAt time.Time
	ActiveAt    time.Time
}

var (
	stateMu     sync.Mutex
	memoryState = map[uint]stateRecord{}
)

func TouchStatus(ctx context.Context, id uint, now time.Time) {
	if id == 0 {
		return
	}
	if client := utils.Redis(ctx); client != nil {
		key := stateKey(id)
		values, _ := client.HGetAll(ctx, key).Result()
		activeAt, hasActive := parseStateTime(values["active_at"])
		connectedAt, hasConnected := parseStateTime(values["connected_at"])
		if !hasActive || activeAt.Before(now.Add(-stateOnlineWindow)) || !hasConnected {
			connectedAt = now
		}
		_ = client.HSet(ctx, key, map[string]string{
			"connected_at": connectedAt.Format(time.RFC3339Nano),
			"active_at":    now.Format(time.RFC3339Nano),
		}).Err()
		_ = client.Expire(ctx, key, stateTTL).Err()
		return
	}

	stateMu.Lock()
	record, ok := memoryState[id]
	if !ok || record.ActiveAt.Before(now.Add(-stateOnlineWindow)) {
		record.ConnectedAt = now
	}
	record.ActiveAt = now
	memoryState[id] = record
	stateMu.Unlock()
}

func ReadStatus(ctx context.Context, id uint, now time.Time) Status {
	if id == 0 {
		return Status{}
	}
	if client := utils.Redis(ctx); client != nil {
		values, err := client.HGetAll(ctx, stateKey(id)).Result()
		if err == nil {
			connectedAt, hasConnected := parseStateTime(values["connected_at"])
			activeAt, hasActive := parseStateTime(values["active_at"])
			if hasActive {
				return buildStatus(connectedAt, hasConnected, activeAt, now)
			}
		}
		return Status{}
	}

	stateMu.Lock()
	record, ok := memoryState[id]
	stateMu.Unlock()
	if !ok {
		return Status{}
	}
	return buildStatus(record.ConnectedAt, true, record.ActiveAt, now)
}

func buildStatus(connectedAt time.Time, hasConnected bool, activeAt time.Time, now time.Time) Status {
	online := activeAt.After(now.Add(-stateOnlineWindow))
	var connected *time.Time
	if hasConnected {
		connected = &connectedAt
	}
	active := activeAt
	status := Status{Online: online, ConnectedAt: connected, ActiveAt: &active}
	if online {
		since := activeAt
		if hasConnected {
			since = connectedAt
		}
		uptime := int(now.Sub(since).Seconds())
		if uptime < 0 {
			uptime = 0
		}
		status.UptimeSeconds = uptime
	}
	return status
}

func parseStateTime(raw string) (time.Time, bool) {
	if raw == "" {
		return time.Time{}, false
	}
	got, err := time.Parse(time.RFC3339Nano, raw)
	return got, err == nil
}

func stateKey(id uint) string {
	return "doj:judger:" + strconv.FormatUint(uint64(id), 10) + ":state"
}
