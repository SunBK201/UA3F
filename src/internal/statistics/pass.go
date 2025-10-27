package statistics

import (
	"fmt"
	"os"
	"sort"

	"github.com/sirupsen/logrus"
)

const passthroughStatsFile = "/var/log/ua3f/passthrough_stats"

type PassThroughRecord struct {
	Host  string
	UA    string
	Count int
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
		logrus.Errorf("create stats file error: %v", err)
		return
	}
	defer f.Close()

	var statList []PassThroughRecord
	for _, record := range passThroughRecords {
		statList = append(statList, *record)
	}
	sort.SliceStable(statList, func(i, j int) bool {
		return statList[i].Count > statList[j].Count
	})

	for _, record := range statList {
		line := fmt.Sprintf("%s %d %s\n", record.Host, record.Count, record.UA)
		f.WriteString(line)
	}
}
