package sg

import (
	"go.uber.org/zap"
)

var lg *zap.SugaredLogger

func init() {
	lg = zap.NewNop().Sugar()
}

// SetLogger sets the package-level logger. If l is nil, a no-op logger is used.
func SetLogger(l *zap.SugaredLogger) {
	if l != nil {
		lg = l
	} else {
		lg = zap.NewNop().Sugar()
	}
}
