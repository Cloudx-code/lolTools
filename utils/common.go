package utils

import (
	"encoding/json"
	"time"
)

func ToJson(i interface{}) string {
	data, _ := json.Marshal(i)
	return string(data)
}

// Retry 带重试间隔的重试
func Retry(retryTimes int, gap time.Duration, f func() error) error {
	var err error
	for i := 0; i < retryTimes; i++ {
		if err = f(); err == nil {
			return nil
		}
		time.Sleep(gap)
	}
	return err
}
