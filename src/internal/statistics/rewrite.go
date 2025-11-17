package statistics

import (
	"fmt"
	"log/slog"
	"os"
	"sort"
)

const rewriteStatsFile = "/var/log/ua3f/rewrite_stats"

type RewriteRecord struct {
	Host       string
	Count      int
	OriginalUA string
	MockedUA   string
}

var rewriteRecords = make(map[string]*RewriteRecord)

func AddRewriteRecord(record *RewriteRecord) {
	select {
	case rewriteRecordChan <- *record:
	default:
	}
}

func dumpRewriteRecords() {
	f, err := os.Create(rewriteStatsFile)
	if err != nil {
		slog.Error("os.Create", slog.Any("error", err))
		return
	}
	defer func() {
		if err := f.Close(); err != nil {
			slog.Error("os.File.Close", slog.Any("error", err))
		}
	}()

	var statList []RewriteRecord
	for _, record := range rewriteRecords {
		statList = append(statList, *record)
	}
	sort.SliceStable(statList, func(i, j int) bool {
		return statList[i].Count > statList[j].Count
	})

	for _, record := range statList {
		line := fmt.Sprintf("%s %d %sSEQSEQ%s\n", record.Host, record.Count, record.OriginalUA, record.MockedUA)
		if _, err := f.WriteString(line); err != nil {
			slog.Error("os.File.WriteString", slog.Any("error", err))
		}
	}
}
