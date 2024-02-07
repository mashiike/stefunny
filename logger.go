package stefunny

import (
	"bytes"
	"io"
	"log"

	"github.com/fatih/color"
	"github.com/fujiwara/logutils"
)

func LoggerSetup(w io.Writer, minLevel string) func() {
	beforeOutput := log.Writer()
	cleanup := func() {
		log.SetOutput(beforeOutput)
	}
	filter := &logutils.LevelFilter{
		Levels:   []logutils.LogLevel{"debug", "info", "notice", "warn", "error"},
		MinLevel: "info",
		ModifierFuncs: []logutils.ModifierFunc{
			logutils.Color(color.FgHiBlack),
			logutils.Color(color.FgWhite),
			logutils.Color(color.FgHiBlue),
			logutils.Color(color.FgYellow),
			logutils.Color(color.FgRed, color.Bold),
		},
		Writer: w,
	}
	if minLevel != "" {
		filter.MinLevel = logutils.LogLevel(minLevel)
	}
	if minLevel == "debug" {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
		log.SetOutput(
			writerFunc(func(b []byte) (int, error) {
				//For align the logs
				x := bytes.IndexByte(b, '[')
				if x >= 0 {
					pos := x - 1
					n := ((pos/4)+1)*4 - pos - 1

					b = append(b[:pos+n], b[pos:]...)
					for i := 0; i < n; i++ {
						b[pos+i] = ' '
					}
				}
				return filter.Write(b)
			}),
		)
		log.Println("[debug] Setting log level to", minLevel)
		return cleanup
	}
	log.SetOutput(filter)
	return cleanup
}

type writerFunc func([]byte) (int, error)

func (f writerFunc) Write(bs []byte) (int, error) {
	return f(bs)
}
