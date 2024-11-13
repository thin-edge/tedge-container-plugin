package c8y

import (
	"github.com/reubenmiller/go-c8y/pkg/logger"
)

// Logger used within the c8y client
var Logger logger.Logger

func init() {
	Logger = logger.NewLogger("c8y")
}

// SilenceLogger causes all log messages to be hidden
func SilenceLogger() {
	Logger = logger.NewDummyLogger("c8y")
}

// UnsilenceLogger enables the logger (opposite of Silence logger)
func UnsilenceLogger() {
	Logger = logger.NewLogger("c8y")
}
