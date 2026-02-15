package statistics

import (
	"sync"

	"github.com/sunbk201/ua3f/internal/log"
)

type Recorder struct {
	RewriteRecordList     *RewriteRecordList
	PassThroughRecordList *PassThroughRecordList
	ConnectionRecordList  *ConnectionRecordList
	once                  sync.Once
}

func New() *Recorder {
	return &Recorder{
		RewriteRecordList:     NewRewriteRecordList(log.GetStatsFilePath("rewrite_stats")),
		PassThroughRecordList: NewPassThroughRecordList(log.GetStatsFilePath("pass_stats")),
		ConnectionRecordList:  NewConnectionRecordList(log.GetStatsFilePath("conn_stats")),
	}
}

func (r *Recorder) Start() {
	r.once.Do(func() {
		r.RewriteRecordList.Run()
		r.PassThroughRecordList.Run()
		r.ConnectionRecordList.Run()
	})
}

func (r *Recorder) AddRecord(record any) {
	switch rec := record.(type) {
	case *RewriteRecord:
		select {
		case r.RewriteRecordList.recordAddChan <- rec:
		default:
		}
	case *PassThroughRecord:
		select {
		case r.PassThroughRecordList.recordAddChan <- rec:
		default:
		}
	case *ConnectionRecord:
		select {
		case r.ConnectionRecordList.recordAddChan <- rec:
		default:
		}
	}
}

func (r *Recorder) RemoveRecord(record any) {
	switch rec := record.(type) {
	case *ConnectionRecord:
		select {
		case r.ConnectionRecordList.recordRemoveChan <- rec:
		default:
		}
	}
}
