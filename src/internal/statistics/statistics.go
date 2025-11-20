package statistics

import (
	"strings"
	"time"
)

var (
	rewriteRecordChan     = make(chan RewriteRecord, 2000)
	passThroughRecordChan = make(chan PassThroughRecord, 2000)
	connectionActionChan  = make(chan ConnectionAction, 2000)
)

// Actions for recording connection statistics
type Action int

const (
	Add Action = iota
	Remove
)

func StartRecorder() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case record := <-rewriteRecordChan:
			rewriteRecordsMu.Lock()
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
			rewriteRecordsMu.Unlock()
		case record := <-passThroughRecordChan:
			if strings.HasPrefix(record.UA, "curl/") {
				record.UA = "curl/*"
			}
			passThroughRecordsMu.Lock()
			if r, exists := passThroughRecords[record.UA]; exists {
				r.Count++
				r.DestAddr = record.DestAddr
				r.SrcAddr = record.SrcAddr
			} else {
				passThroughRecords[record.UA] = &PassThroughRecord{
					SrcAddr:  record.SrcAddr,
					DestAddr: record.DestAddr,
					UA:       record.UA,
					Count:    1,
				}
			}
			passThroughRecordsMu.Unlock()
		case action := <-connectionActionChan:
			connectionRecordsMu.Lock()
			switch action.Action {
			case Add:
				if r, exists := connectionRecords[action.Key]; exists {
					r.Protocol = action.Record.Protocol
				} else {
					connectionRecords[action.Key] = &ConnectionRecord{
						Protocol:  action.Record.Protocol,
						SrcAddr:   action.Record.SrcAddr,
						DestAddr:  action.Record.DestAddr,
						StartTime: action.Record.StartTime,
					}
				}
			case Remove:
				delete(connectionRecords, action.Key)
			}
			connectionRecordsMu.Unlock()
		case <-ticker.C:
			dumpRewriteRecords()
			dumpPassThroughRecords()
			dumpConnectionRecords()
		}
	}
}
