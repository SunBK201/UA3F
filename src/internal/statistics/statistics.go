package statistics

import (
	"strings"
	"time"
)

var (
	rewriteRecordChan     = make(chan RewriteRecord, 2000)
	passThroughRecordChan = make(chan PassThroughRecord, 2000)
)

func StartRecorder() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case record := <-rewriteRecordChan:
			if r, exists := rewriteRecords[record.Host]; exists {
				r.Count++
				r.OriginalUA = record.OriginalUA
				r.MockedUA = record.MockedUA
			} else {
				rewriteRecords[record.Host] = &RewriteRecord{
					Host:       record.Host,
					Count:      1,
					OriginalUA: record.OriginalUA,
					MockedUA:   record.MockedUA,
				}
			}
		case record := <-passThroughRecordChan:
			if strings.HasPrefix(record.UA, "curl/") {
				record.UA = "curl/*"
			}
			if r, exists := passThroughRecords[record.UA]; exists {
				r.Count++
				r.Host = record.Host
			} else {
				passThroughRecords[record.UA] = &PassThroughRecord{
					Host:  record.Host,
					UA:    record.UA,
					Count: 1,
				}
			}
		case <-ticker.C:
			dumpRewriteRecords()
			dumpPassThroughRecords()
		}
	}
}
