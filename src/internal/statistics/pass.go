package statistics

import (
	"fmt"
	"os"
	"sort"

	"github.com/sirupsen/logrus"
)

const passthroughStatsFile = "/var/log/ua3f/pass_stats"

type PassThroughRecord struct {
	SrcAddr  string
	DestAddr string
	UA       string
	Count    int
}

var passThroughRecords = make(map[string]*PassThroughRecord)

func AddPassThroughRecord(record *PassThroughRecord) {
	select {
	case passThroughRecordChan <- *record:
	default:
	}
}

func dumpPassThroughRecords() {
	f, err := os.Create(passthroughStatsFile)
	if err != nil {
		logrus.Errorf("os.Create: %v", err)
		return
	}
	defer func() {
		if err := f.Close(); err != nil {
			logrus.Errorf("os.File.Close: %v", err)
		}
	}()

	var statList []PassThroughRecord
	for _, record := range passThroughRecords {
		statList = append(statList, *record)
	}
	sort.SliceStable(statList, func(i, j int) bool {
		return statList[i].Count > statList[j].Count
	})

	for _, record := range statList {
		line := fmt.Sprintf("%s %s %d %s\n", record.SrcAddr, record.DestAddr, record.Count, record.UA)
		if _, err := f.WriteString(line); err != nil {
			logrus.Errorf("os.File.WriteString: %v", err)
			return
		}
	}
}
