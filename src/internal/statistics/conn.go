package statistics

import (
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
		recordAddChan:    make(chan *ConnectionRecord, 500),
		recordRemoveChan: make(chan *ConnectionRecord, 500),
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
	var statList []ConnectionRecord
	for _, record := range l.records {
		statList = append(statList, *record)
	}
	l.mu.RUnlock()

	// Sort by start time (newest first)
	sort.SliceStable(statList, func(i, j int) bool {
		return statList[i].StartTime.After(statList[j].StartTime)
	})

	for _, record := range statList {
		duration := time.Since(record.StartTime)
		line := fmt.Sprintf("%s %s %s %d\n",
			record.Protocol, record.SrcAddr, record.DestAddr, int(duration.Seconds()))
		if _, err := f.WriteString(line); err != nil {
			slog.Error("os.File.WriteString", slog.Any("error", err))
			return
		}
	}
}
