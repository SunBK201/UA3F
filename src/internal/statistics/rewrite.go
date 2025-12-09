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
	dumpFile      string
	dumpWriter    *bufio.Writer
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
		dumpWriter:    bufio.NewWriter(nil),
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
	records := make([]*RewriteRecord, 0, len(l.records))
	for _, record := range l.records {
		records = append(records, record)
	}
	l.mu.RUnlock()

	sort.SliceStable(records, func(i, j int) bool {
		return records[i].Count > records[j].Count
	})

	l.dumpWriter.Reset(f)
	defer func() {
		if err := l.dumpWriter.Flush(); err != nil {
			slog.Error("bufio.Writer.Flush", slog.Any("error", err))
		}
	}()

	for _, record := range records {
		_, err := fmt.Fprintf(l.dumpWriter, "%s %d %sSEQSEQ%s\n",
			record.Host, record.Count, record.OriginalUA, record.MockedUA)
		if err != nil {
			slog.Error("Dump fmt.Fprintf", slog.Any("error", err))
		}
	}
}
