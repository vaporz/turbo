package turbo

import (
	"github.com/apsdehal/go-logger"
	"os"
)

var log *logger.Logger

func initLogger()  {
	//log instance init
	var err error
	log, err = logger.New("turbo", 1, os.Stdout)
	if err != nil {
		panic(err)
	}
}