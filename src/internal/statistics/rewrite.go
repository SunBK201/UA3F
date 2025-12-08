package statistics

import (
	"fmt"
	"log/slog"
	"os"
	"sort"
	"sync"
	"time"
)

type RewriteRecordList struct {
	recordAddChan chan *RewriteRecord
	records       map[string]*RewriteRecord
	mu            sync.RWMutex
	dumpFile      string
}

type RewriteRecord struct {
	Host       string
	Count      int
	OriginalUA string
	MockedUA   string
}

func NewRewriteRecordList(dumpFile string) *RewriteRecordList {
	return &RewriteRecordList{
		recordAddChan: make(chan *RewriteRecord, 100),
		records:       make(map[string]*RewriteRecord, 100),
		mu:            sync.RWMutex{},
		dumpFile:      dumpFile,
	}
}

func (l *RewriteRecordList) Run() {
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case record := <-l.recordAddChan:
				l.Add(record)
			case <-ticker.C:
				l.Dump()
			}
		}
	}()
}

func (l *RewriteRecordList) Add(record *RewriteRecord) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if r, exists := l.records[record.Host]; exists {
		r.Count++
		r.OriginalUA = record.OriginalUA
		r.MockedUA = record.MockedUA
	} else {
		l.records[record.Host] = &RewriteRecord{
			Host:       record.Host,
			Count:      1,
			OriginalUA: record.OriginalUA,
			MockedUA:   record.MockedUA,
		}
	}
}

func (l *RewriteRecordList) Dump() {
	f, err := os.Create(l.dumpFile)
	if err != nil {
		slog.Error("os.Create", slog.Any("error", err))
		return
	}
	defer func() {
		if err := f.Close(); err != nil {
			slog.Error("os.File.Close", slog.Any("error", err))
		}
	}()

	l.mu.RLock()
	var statList []RewriteRecord
	for _, record := range l.records {
		statList = append(statList, *record)
	}
	l.mu.RUnlock()

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
