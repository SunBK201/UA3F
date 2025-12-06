package statistics

type Recorder struct {
	RewriteRecordList     *RewriteRecordList
	PassThroughRecordList *PassThroughRecordList
	ConnectionRecordList  *ConnectionRecordList
}

func New() *Recorder {
	return &Recorder{
		RewriteRecordList:     NewRewriteRecordList("/var/log/ua3f/rewrite_stats"),
		PassThroughRecordList: NewPassThroughRecordList("/var/log/ua3f/pass_stats"),
		ConnectionRecordList:  NewConnectionRecordList("/var/log/ua3f/conn_stats"),
	}
}

func (r *Recorder) Start() {
	r.RewriteRecordList.Run()
	r.PassThroughRecordList.Run()
	r.ConnectionRecordList.Run()
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
