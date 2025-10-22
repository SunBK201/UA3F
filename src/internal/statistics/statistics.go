package statistics

import (
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/sirupsen/logrus"
)

type StatRecord struct {
	Host     string
	Count    int
	OriginUA string
	MockedUA string
}

var (
	statChan = make(chan StatRecord, 3000)
	stats    = make(map[string]*StatRecord)
)

const statsFile = "/var/log/ua3f/stats"

func StartStatWorker() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case dest := <-statChan:
			if record, exists := stats[dest.Host]; exists {
				record.Count++
				record.OriginUA = dest.OriginUA
				record.MockedUA = dest.MockedUA
			} else {
				stats[dest.Host] = &StatRecord{
					Host:     dest.Host,
					Count:    1,
					OriginUA: dest.OriginUA,
					MockedUA: dest.MockedUA,
				}
			}
		case <-ticker.C:
			dumpStatsToFile()
		}
	}
}

func AddStat(dest *StatRecord) {
	select {
	case statChan <- *dest:
	default:
	}
}

func dumpStatsToFile() {
	f, err := os.Create(statsFile)
	if err != nil {
		logrus.Errorf("create stats file error: %v", err)
		return
	}
	defer f.Close()

	var statList []StatRecord
	for _, record := range stats {
		statList = append(statList, *record)
	}
	sort.SliceStable(statList, func(i, j int) bool {
		return statList[i].Count > statList[j].Count
	})

	for _, record := range statList {
		line := fmt.Sprintf("%s %d %sSEQSEQ%s\n", record.Host, record.Count, record.OriginUA, record.MockedUA)
		f.WriteString(line)
	}
}
