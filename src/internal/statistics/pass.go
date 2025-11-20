package statistics

import (
	"fmt"
	"log/slog"
	"os"
	"sort"
	"sync"
)

const passthroughStatsFile = "/var/log/ua3f/pass_stats"

type PassThroughRecord struct {
	SrcAddr  string
	DestAddr string
	UA       string
	Count    int
}

var (
	passThroughRecords   = make(map[string]*PassThroughRecord)
	passThroughRecordsMu sync.RWMutex
)

func AddPassThroughRecord(record *PassThroughRecord) {
	select {
	case passThroughRecordChan <- *record:
	default:
	}
}

func dumpPassThroughRecords() {
	f, err := os.Create(passthroughStatsFile)
	if err != nil {
		slog.Error("os.Create", slog.Any("error", err))
		return
	}
	defer func() {
		if err := f.Close(); err != nil {
			slog.Error("os.File.Close", slog.Any("error", err))
		}
	}()

	passThroughRecordsMu.RLock()
	var statList []PassThroughRecord
	for _, record := range passThroughRecords {
		statList = append(statList, *record)
	}
	passThroughRecordsMu.RUnlock()

	sort.SliceStable(statList, func(i, j int) bool {
		return statList[i].Count > statList[j].Count
	})

	for _, record := range statList {
		line := fmt.Sprintf("%s %s %d %s\n", record.SrcAddr, record.DestAddr, record.Count, record.UA)
		if _, err := f.WriteString(line); err != nil {
			slog.Error("os.File.WriteString", slog.Any("error", err))
			return
		}
	}
}
