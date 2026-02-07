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

	dumpRecords []*ConnectionRecord
	dumpFile    string
	dumpWriter  *bufio.Writer

	dumpInterval    time.Duration
	cleanupInterval time.Duration
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
		dumpRecords:      make([]*ConnectionRecord, 0, 500),
		dumpFile:         dumpFile,
		dumpWriter:       bufio.NewWriter(nil),
		dumpInterval:     5 * time.Second,
		cleanupInterval:  24 * time.Hour,
	}
}

func (l *ConnectionRecordList) Run() {
	go func() {
		dumpTicker := time.NewTicker(l.dumpInterval)
		cleanupTicker := time.NewTicker(l.cleanupInterval)
		defer dumpTicker.Stop()
		defer cleanupTicker.Stop()

		for {
			select {
			case record := <-l.recordAddChan:
				l.Add(record)
			case record := <-l.recordRemoveChan:
				l.Remove(record)
			case <-dumpTicker.C:
				l.Dump()
			case <-cleanupTicker.C:
				l.Cleanup()
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

func (l *ConnectionRecordList) Cleanup() {
	cutoff := time.Now().Add(-l.cleanupInterval)

	l.mu.Lock()
	defer l.mu.Unlock()

	for key, record := range l.records {
		if record.StartTime.Before(cutoff) {
			delete(l.records, key)
		}
	}
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

	l.dumpRecords = l.dumpRecords[:0]
	l.mu.RLock()
	for _, r := range l.records {
		l.dumpRecords = append(l.dumpRecords, r)
	}
	l.mu.RUnlock()

	// Sort by start time (newest first)
	sort.SliceStable(l.dumpRecords, func(i, j int) bool {
		return l.dumpRecords[i].StartTime.After(l.dumpRecords[j].StartTime)
	})

	l.dumpWriter.Reset(f)
	defer func() {
		if err := l.dumpWriter.Flush(); err != nil {
			slog.Error("bufio.Writer.Flush", slog.Any("error", err))
		}
	}()

	now := time.Now()
	for _, record := range l.dumpRecords {
		duration := now.Sub(record.StartTime)
		_, err := fmt.Fprintf(l.dumpWriter, "%s %s %s %d\n",
			record.Protocol, record.SrcAddr, record.DestAddr, int(duration.Seconds()))
		if err != nil {
			slog.Error("Dump fmt.Fprintf", slog.Any("error", err))
		}
	}
}
