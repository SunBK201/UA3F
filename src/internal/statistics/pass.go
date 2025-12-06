package statistics

import (
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

type PassThroughRecordList struct {
	recordAddChan chan *PassThroughRecord
	records       map[string]*PassThroughRecord
	mu            sync.RWMutex
	dumpFile      string
}

type PassThroughRecord struct {
	SrcAddr  string
	DestAddr string
	UA       string
	Count    int
}

func NewPassThroughRecordList(dumpFile string) *PassThroughRecordList {
	return &PassThroughRecordList{
		recordAddChan: make(chan *PassThroughRecord, 500),
		records:       make(map[string]*PassThroughRecord, 500),
		mu:            sync.RWMutex{},
		dumpFile:      dumpFile,
	}
}

func (l *PassThroughRecordList) Run() {
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

func (l *PassThroughRecordList) Add(record *PassThroughRecord) {
	if strings.HasPrefix(record.UA, "curl/") {
		record.UA = "curl/*"
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if r, exists := l.records[record.UA]; exists {
		r.Count++
		r.SrcAddr = record.SrcAddr
		r.DestAddr = record.DestAddr
	} else {
		l.records[record.UA] = &PassThroughRecord{
			SrcAddr:  record.SrcAddr,
			DestAddr: record.DestAddr,
			UA:       record.UA,
			Count:    1,
		}
	}
}

func (l *PassThroughRecordList) Dump() {
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
	var statList []PassThroughRecord
	for _, record := range l.records {
		statList = append(statList, *record)
	}
	l.mu.RUnlock()

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
