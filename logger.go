package sg

import (
	"flag"

	"go.uber.org/zap"
)

var lg *zap.SugaredLogger

var (
	logs = flag.Bool("logs", false, "enable logging")
)

func logging() {
	if lg != nil {
		return
	}
	if *logs {
		t, _ := zap.NewProduction()
		lg = t.Sugar()
	} else {
		lg = zap.NewNop().Sugar()
	}
}
