package xserver

import (
	"strconv"
)

var sync = make(chan bool)

func idToString(id uint64) string {
	return strconv.FormatUint(id, 10)
}

func stringToId(sid string) uint64 {
	i, err := strconv.ParseUint(sid, 10, 64)
	if err != nil {
		logger.Error(err.Error())
		return NullId
	}
	return i
}

func int64ToString(i int64) string {
	return strconv.FormatInt(i, 10)
}

// Pause for debug
func Pause() {
	for {
	}
}
