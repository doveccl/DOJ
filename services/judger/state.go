package judger

import (
	"context"
	"strconv"
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

func TouchStatus(ctx context.Context, id uint, now time.Time) {
	if id == 0 {
		return
	}
	key := stateKey(id)
	var record stateRecord
	found, err := utils.CacheGet(ctx, key, &record)
	if err != nil {
		return
	}
	if !found || record.ActiveAt.Before(now.Add(-stateOnlineWindow)) {
		record.ConnectedAt = now
	}
	record.ActiveAt = now
	_ = utils.CacheSet(ctx, key, record, stateTTL)
}

func ReadStatus(ctx context.Context, id uint, now time.Time) Status {
	if id == 0 {
		return Status{}
	}
	var record stateRecord
	found, err := utils.CacheGet(ctx, stateKey(id), &record)
	if err != nil || !found || record.ActiveAt.IsZero() {
		return Status{}
	}
	return buildStatus(record.ConnectedAt, !record.ConnectedAt.IsZero(), record.ActiveAt, now)
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

func stateKey(id uint) string {
	return "doj:judger:" + strconv.FormatUint(uint64(id), 10) + ":state"
}
