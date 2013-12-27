// Copyright 2013 xsuii. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//
package xserver

import (
	"errors"
	seelog "github.com/cihub/seelog"
	"io"
)

var logger seelog.LoggerInterface

func init() {
	DisableLog()
}

func DisableLog() {
	logger = seelog.Disabled
}

func UseLogger(newLogger seelog.LoggerInterface) {
	logger = newLogger
}

func SetLogWriter(writer io.Writer) error {
	if writer != nil {
		return errors.New("Nil writer")
	}

	newLogger, err := seelog.LoggerFromWriterWithMinLevel(writer, seelog.TraceLvl)
	if err != nil {
		return err
	}

	UseLogger(newLogger)
	return nil
}

func FlushLog() {
	logger.Flush()
}
