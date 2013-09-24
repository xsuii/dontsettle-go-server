package xserver

import (
	"strconv"
)

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
