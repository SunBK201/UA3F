package statistics

import (
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/sunbk201/ua3f/internal/sniff"
)

const connStatsFile = "/var/log/ua3f/conn_stats"

type ConnectionRecord struct {
	Protocol  sniff.Protocol
	SrcAddr   string
	DestAddr  string
	StartTime time.Time
}

type ConnectionAction struct {
	Action Action
	Key    string
	Record ConnectionRecord
}

var connectionRecords = make(map[string]*ConnectionRecord)

// AddConnection adds or updates a connection record
func AddConnection(record *ConnectionRecord) {
	select {
	case connectionActionChan <- ConnectionAction{
		Action: Add,
		Key:    fmt.Sprintf("%s-%s", record.SrcAddr, record.DestAddr),
		Record: *record,
	}:
	default:
	}
}

// RemoveConnection removes a connection record
func RemoveConnection(srcAddr, destAddr string) {
	select {
	case connectionActionChan <- ConnectionAction{
		Action: Remove,
		Key:    fmt.Sprintf("%s-%s", srcAddr, destAddr),
	}:
	default:
	}
}

func dumpConnectionRecords() {
	f, err := os.Create(connStatsFile)
	if err != nil {
		logrus.Errorf("os.Create: %v", err)
		return
	}
	defer func() {
		if err := f.Close(); err != nil {
			logrus.Errorf("os.File.Close: %v", err)
		}
	}()

	var statList []ConnectionRecord
	for _, record := range connectionRecords {
		statList = append(statList, *record)
	}

	// Sort by start time (newest first)
	sort.SliceStable(statList, func(i, j int) bool {
		return statList[i].StartTime.After(statList[j].StartTime)
	})

	for _, record := range statList {
		duration := time.Since(record.StartTime)
		line := fmt.Sprintf("%s %s %s %d\n",
			record.Protocol, record.SrcAddr, record.DestAddr, int(duration.Seconds()))
		if _, err := f.WriteString(line); err != nil {
			logrus.Errorf("os.File.WriteString: %v", err)
			return
		}
	}
}
