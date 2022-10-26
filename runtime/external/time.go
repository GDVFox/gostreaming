package external

import (
	"sync/atomic"
	"time"
)

var currentTime int64

func init() {
	ticker := time.NewTicker(time.Second)
	go func() {
		for {
			<-ticker.C
			atomic.StoreInt64(&currentTime, time.Now().Unix())
		}
	}()
}

// TimeNow возвращает текущее время в unix timestamp.
func TimeNow() time.Time {
	return time.Unix(0, atomic.LoadInt64(&currentTime))
}
