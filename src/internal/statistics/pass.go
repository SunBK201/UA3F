package statistics

import (
	"bufio"
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
		recordAddChan: make(chan *PassThroughRecord, 100),
		records:       make(map[string]*PassThroughRecord, 100),
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
	records := make([]*PassThroughRecord, 0, len(l.records))
	for _, r := range l.records {
		records = append(records, r)
	}
	l.mu.RUnlock()

	sort.SliceStable(records, func(i, j int) bool {
		return records[i].Count > records[j].Count
	})

	w := bufio.NewWriter(f)
	defer func() {
		if err := w.Flush(); err != nil {
			slog.Error("bufio.Writer.Flush", slog.Any("error", err))
		}
	}()

	for _, record := range records {
		_, err := fmt.Fprintf(w, "%s %s %d %s\n",
			record.SrcAddr, record.DestAddr, record.Count, record.UA)
		if err != nil {
			slog.Error("Dump fmt.Fprintf", slog.Any("error", err))
		}
	}
}
