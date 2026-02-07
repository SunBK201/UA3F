package statistics

import (
	"bufio"
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

	dumpRecords []*RewriteRecord
	dumpFile    string
	dumpWriter  *bufio.Writer

	dumpInterval    time.Duration
	cleanupInterval time.Duration
}

type RewriteRecord struct {
	Host       string
	Count      int
	OriginalUA string
	MockedUA   string
	LastSeen   time.Time
}

func NewRewriteRecordList(dumpFile string) *RewriteRecordList {
	return &RewriteRecordList{
		recordAddChan:   make(chan *RewriteRecord, 100),
		records:         make(map[string]*RewriteRecord, 300),
		mu:              sync.RWMutex{},
		dumpRecords:     make([]*RewriteRecord, 0, 300),
		dumpFile:        dumpFile,
		dumpWriter:      bufio.NewWriter(nil),
		dumpInterval:    5 * time.Second,
		cleanupInterval: 24 * time.Hour,
	}
}

func (l *RewriteRecordList) Run() {
	go func() {
		dumpTicker := time.NewTicker(l.dumpInterval)
		cleanupTicker := time.NewTicker(l.cleanupInterval)
		defer dumpTicker.Stop()
		defer cleanupTicker.Stop()

		for {
			select {
			case record := <-l.recordAddChan:
				l.Add(record)
			case <-dumpTicker.C:
				l.Dump()
			case <-cleanupTicker.C:
				l.Cleanup()
			}
		}
	}()
}

func (l *RewriteRecordList) Add(record *RewriteRecord) {
	now := time.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	if r, exists := l.records[record.Host]; exists {
		r.Count++
		r.OriginalUA = record.OriginalUA
		r.MockedUA = record.MockedUA
		r.LastSeen = now
	} else {
		l.records[record.Host] = &RewriteRecord{
			Host:       record.Host,
			Count:      1,
			OriginalUA: record.OriginalUA,
			MockedUA:   record.MockedUA,
			LastSeen:   now,
		}
	}
}

func (l *RewriteRecordList) Cleanup() {
	cutoff := time.Now().Add(-l.cleanupInterval)

	l.mu.Lock()
	defer l.mu.Unlock()

	for host, record := range l.records {
		if record.LastSeen.Before(cutoff) {
			delete(l.records, host)
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

	l.dumpRecords = l.dumpRecords[:0]
	l.mu.RLock()
	for _, record := range l.records {
		l.dumpRecords = append(l.dumpRecords, record)
	}
	l.mu.RUnlock()

	sort.SliceStable(l.dumpRecords, func(i, j int) bool {
		return l.dumpRecords[i].Count > l.dumpRecords[j].Count
	})

	l.dumpWriter.Reset(f)
	defer func() {
		if err := l.dumpWriter.Flush(); err != nil {
			slog.Error("bufio.Writer.Flush", slog.Any("error", err))
		}
	}()

	for _, record := range l.dumpRecords {
		_, err := fmt.Fprintf(l.dumpWriter, "%s %d %sSEQSEQ%s\n",
			record.Host, record.Count, record.OriginalUA, record.MockedUA)
		if err != nil {
			slog.Error("Dump fmt.Fprintf", slog.Any("error", err))
		}
	}
}
