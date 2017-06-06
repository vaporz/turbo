package turbo

import (
	logger "github.com/sirupsen/logrus"
	"os"
	"io"
)

var log *logger.Logger

func init() {
	//// Log as JSON instead of the default ASCII formatter.
	//logger.SetFormatter(&logger.TextFormatter{})
	//// Output to stdout instead of the default stderr
	//logger.SetOutput(os.Stdout)
	////Log the Debug level and above.
	//logger.SetLevel(logger.DebugLevel)
	//
	log = logger.StandardLogger()
	log = &logger.Logger{
		Out:       os.Stderr,
		Formatter: new(logger.TextFormatter),
		Hooks:     make(logger.LevelHooks),
		Level:     logger.DebugLevel,
	}
}

func SetOutput(out io.Writer) {
	log.Out = out
}