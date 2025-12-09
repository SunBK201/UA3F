package statistics

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/sunbk201/ua3f/internal/sniff"
)

type ConnectionRecordList struct {
	recordAddChan    chan *ConnectionRecord
	recordRemoveChan chan *ConnectionRecord
	records          map[string]*ConnectionRecord
	mu               sync.RWMutex
	dumpFile         string
}

type ConnectionRecord struct {
	Protocol  sniff.Protocol
	SrcAddr   string
	DestAddr  string
	StartTime time.Time
}

func NewConnectionRecordList(dumpFile string) *ConnectionRecordList {
	return &ConnectionRecordList{
		recordAddChan:    make(chan *ConnectionRecord, 100),
		recordRemoveChan: make(chan *ConnectionRecord, 100),
		records:          make(map[string]*ConnectionRecord, 500),
		mu:               sync.RWMutex{},
		dumpFile:         dumpFile,
	}
}

func (l *ConnectionRecordList) Run() {
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case record := <-l.recordAddChan:
				l.Add(record)
			case record := <-l.recordRemoveChan:
				l.Remove(record)
			case <-ticker.C:
				l.Dump()
			}
		}
	}()
}

func (l *ConnectionRecordList) Add(record *ConnectionRecord) {
	l.mu.Lock()
	defer l.mu.Unlock()

	key := fmt.Sprintf("%s-%s", record.SrcAddr, record.DestAddr)
	if r, exists := l.records[key]; exists {
		r.Protocol = record.Protocol
	} else {
		startTime := record.StartTime
		if startTime.IsZero() {
			startTime = time.Now()
		}
		l.records[key] = &ConnectionRecord{
			Protocol:  record.Protocol,
			SrcAddr:   record.SrcAddr,
			DestAddr:  record.DestAddr,
			StartTime: startTime,
		}
	}
}

func (l *ConnectionRecordList) Remove(record *ConnectionRecord) {
	l.mu.Lock()
	defer l.mu.Unlock()

	key := fmt.Sprintf("%s-%s", record.SrcAddr, record.DestAddr)
	delete(l.records, key)
}

func (l *ConnectionRecordList) Dump() {
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
	records := make([]*ConnectionRecord, 0, len(l.records))
	for _, r := range l.records {
		records = append(records, r)
	}
	l.mu.RUnlock()

	// Sort by start time (newest first)
	sort.SliceStable(records, func(i, j int) bool {
		return records[i].StartTime.After(records[j].StartTime)
	})

	w := bufio.NewWriter(f)
	defer func() {
		if err := w.Flush(); err != nil {
			slog.Error("bufio.Writer.Flush", slog.Any("error", err))
		}
	}()

	now := time.Now()
	for _, record := range records {
		duration := now.Sub(record.StartTime)
		_, err := fmt.Fprintf(w, "%s %s %s %d\n",
			record.Protocol, record.SrcAddr, record.DestAddr, int(duration.Seconds()))
		if err != nil {
			slog.Error("Dump fmt.Fprintf", slog.Any("error", err))
		}
	}
}
