package main

import (
	"fmt"
	"os"
	"time"

	"github.com/go-kit/kit/log"
)

////////////////////////////////////////////////////////////////////////////
// Constant and data type/structure definitions

////////////////////////////////////////////////////////////////////////////
// Global variables definitions

var (
	logger log.Logger
	debug  = 0
)

////////////////////////////////////////////////////////////////////////////
// Function definitions

//==========================================================================
// init

func init() {
	// https://godoc.org/github.com/go-kit/kit/log#TimestampFormat
	timestampFormat := log.TimestampFormat(time.Now, "0102T15:04:05") // 2006-01-02
	logger = log.NewLogfmtLogger(os.Stderr)
	logger = log.With(logger, "TS", timestampFormat)
	fmt.Println()
}

//==========================================================================
// support functions

// log level
// 0: log unconditionally
// 1: info, necessary output
// 2: verbose, optional output
// 3: debug, output for debug purpose
// 4: silly, unnecessary output
// 5: hidden, rarly unveil such output
// 6+: deep debug, hide from all above output cases
//     normally no longer for output control, but for parking
//     bump the visibility when things go wrong instead
func logIf(level int, message string, args ...interface{}) {
	if debug < level {
		return
	}
	p := make([]interface{}, 0)
	p = append(p, "Msg")
	p = append(p, message)
	p = append(p, args...)
	//fmt.Printf("%#v\n", p)
	logger.Log(p...)
}

// abortOn will quit on anticipated errors gracefully without stack trace
func abortOn(errCase string, e error) {
	_abortOn(errCase, e, 1)
}

func _abortOn(errCase string, e error, ret int) {
	if e != nil {
		logger.Log("Abort", errCase, "Err", e)
		os.Exit(ret)
	}
}
