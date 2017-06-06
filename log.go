package turbo

import (
	logger "github.com/sirupsen/logrus"
	"os"
	"io"
	"runtime"
	"strings"
	"path"
)

var log *logger.Logger

type ContextHook struct{}

func (hook ContextHook) Levels() []logger.Level {
	return logger.AllLevels
}

func (hook ContextHook) Fire(entry *logger.Entry) error {
	pc := make([]uintptr, 3, 3)
	cnt := runtime.Callers(7, pc)
	
	for i := 0; i < cnt; i++ {
		pc_i := pc[i] - 1
		fu := runtime.FuncForPC(pc_i)
		name := fu.Name()
		if !strings.Contains(name, "github.com/Sirupsen/logrus") {
			file, line := fu.FileLine(pc_i)
			entry.Data["file"] = path.Base(file)
			//entry.Data["func"] = path.Base(name)
			entry.Data["line"] = line
			break
		}
	}
	return nil
}

func init() {
	//log file, func and line in dep env
	if Environment == "production" {
		// Log as JSON instead of the default ASCII formatter.
		logger.SetFormatter(&logger.JSONFormatter{})
		logger.SetOutput(os.Stderr)
		logger.SetLevel(logger.InfoLevel)
	} else {
		logger.SetFormatter(&logger.TextFormatter{})
		logger.SetOutput(os.Stderr)
		// set logger with debug level in development environment.
		logger.SetLevel(logger.DebugLevel)
		logger.AddHook(ContextHook{})
	}
	
	log = logger.StandardLogger()
}

func SetOutput(out io.Writer) {
	log.Out = out
}