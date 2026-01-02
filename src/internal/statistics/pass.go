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

	dumpRecords []*PassThroughRecord
	dumpFile    string
	dumpWriter  *bufio.Writer
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
		dumpRecords:   make([]*PassThroughRecord, 0, 100),
		dumpFile:      dumpFile,
		dumpWriter:    bufio.NewWriter(nil),
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
	if record.UA == "" {
		return
	}

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

	l.dumpRecords = l.dumpRecords[:0]
	l.mu.RLock()
	for _, r := range l.records {
		l.dumpRecords = append(l.dumpRecords, r)
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
		_, err := fmt.Fprintf(l.dumpWriter, "%s %s %d %s\n",
			record.SrcAddr, record.DestAddr, record.Count, record.UA)
		if err != nil {
			slog.Error("Dump fmt.Fprintf", slog.Any("error", err))
		}
	}
}
