package statistics

import (
	"fmt"
	"time"
)

type TestResult struct {
	ChanId  int
	Nonce   uint64        // id
	ReqTime time.Time     // request time
	Cost    time.Duration // total cost
	Success bool          // success
}

func (tr *TestResult) String() string {
	return fmt.Sprintf("Nonce:%d ReqTime:%s Success:%v Cost:%.3fs", tr.Nonce, tr.ReqTime.Format("04:05.000"), tr.Success, tr.Cost.Seconds())
}
