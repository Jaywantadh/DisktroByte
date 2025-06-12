package logging

import (
	"os"

	"github.com/sirupsen/logrus"
)

var Log *logrus.Logger

func InitLogger(debug bool){
	Log = logrus.New()
	Log.Out = os.Stdout

	if debug {
		Log.SetLevel(logrus.DebugLevel)
		Log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
	} else {
		Log.SetLevel(logrus.InfoLevel)
		Log.SetFormatter(&logrus.JSONFormatter{})
	}
}