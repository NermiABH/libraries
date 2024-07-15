package inform

import (
	"fmt"
	"log"
	"runtime"
	"strconv"
	"sync/atomic"
)

type LogLevel string

const (
	LogLevelLog   LogLevel = "LOG"
	LogLevelWarn  LogLevel = "WARN"
	LogLevelCrit  LogLevel = "CRIT"
	LogLevelPanic LogLevel = "PANIC"
	LogLevelFatal LogLevel = "FATAL"
)

type LogMessage struct {
	Level    LogLevel
	Caller   string
	IsPrintf bool
	Format   string
	Params   []any
}

func (l LogMessage) formattedMessage() string {
	if l.IsPrintf {
		return fmt.Sprintf(l.Caller+" "+l.Format, l.Params...)
	}
	return fmt.Sprintln(append([]interface{}{l.Caller}, l.Params...)...)
}
func (l LogMessage) coloredMessage(msg string) string {
	switch l.Level {
	case LogLevelLog:
		return msg
	case LogLevelWarn:
		return Yellow + msg + Reset
	case LogLevelCrit:
		return Red + msg + Reset
	case LogLevelPanic:
		return Cyan + msg + Reset
	case LogLevelFatal:
		return Magenta + msg + Reset
	default:
		return msg
	}
}

type LogConfig struct {
	logChan           chan LogMessage
	done              chan struct{}
	tg                *Sender
	tgEnabled         atomic.Bool
	tgLogEnabled      atomic.Bool
	tgWarnEnabled     atomic.Bool
	tgCritEnabled     atomic.Bool
	tgPanicEnabled    atomic.Bool
	tgFatalEnabled    atomic.Bool
	errMetrics        *ErrMetrics
	errMetricsEnabled atomic.Bool
}

var logConfig = &LogConfig{}

func EnableLogProcessor(queue int) {
	if logConfig.logChan != nil {
		panic("log processor already running")
	}
	logConfig.logChan = make(chan LogMessage, queue)
	logConfig.done = make(chan struct{})
	go logProcessor()
}

func StopLogProcessor() {
	if logConfig.logChan != nil {
		close(logConfig.logChan)
		<-logConfig.done
	}
}

func logProcessor() {
	for msg := range logConfig.logChan {
		processLogMessage(msg)
	}
	close(logConfig.done)
}

func processLogMessage(lm LogMessage) {
	msg := lm.formattedMessage()
	log.Print(lm.coloredMessage(msg))
	if lm.Level != LogLevelLog {
		if logConfig.errMetricsEnabled.Load() {
			logConfig.errMetrics.ErrGaugeInc(ErrLevel(lm.Level))
		}
		if logConfig.tgEnabled.Load() {
			switch {
			case lm.Level == LogLevelWarn && logConfig.tgWarnEnabled.Load():
				logConfig.tg.ToQueue(fmt.Sprintf("[WARN] %s", msg))
			case lm.Level == LogLevelCrit && logConfig.tgCritEnabled.Load():
				logConfig.tg.ToQueue(fmt.Sprintf("[CRIT] %s", msg))
			}
		}
	}
}

func EnableErrMetrics(namespace, subsystem string) error {
	m, err := errMetricsInit(namespace, subsystem)
	if err != nil {
		return err
	}
	logConfig.errMetrics = m
	logConfig.errMetricsEnabled.Store(true)
	return nil
}

func EnableTelegram(subsystem, token, chatID, threadId string, queue int) error {
	logConfig.tg = NewSender(subsystem, token, chatID, threadId, queue)
	logConfig.tgEnabled.Store(true)
	return nil
}

func EnableLogToTelegram()   { logConfig.tgLogEnabled.Store(true) }
func EnableWarnToTelegram()  { logConfig.tgWarnEnabled.Store(true) }
func EnableCritToTelegram()  { logConfig.tgCritEnabled.Store(true) }
func EnablePanicToTelegram() { logConfig.tgPanicEnabled.Store(true) }
func EnableFatalToTelegram() { logConfig.tgFatalEnabled.Store(true) }

func getCallerInfo() string {
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		return "unknown file"
	}
	short := file
	for i, j := len(file)-1, 0; i > 0 && j < 2; i-- {
		if file[i] == '/' {
			short = file[i+1:]
			j++
		}
	}
	file = short
	return short + ":" + strconv.Itoa(line)
}

func Log(v ...any) {
	lm := LogMessage{LogLevelLog, getCallerInfo(), false, "", v}
	logMessage(lm)
}

func Logf(format string, v ...any) {
	lm := LogMessage{LogLevelLog, getCallerInfo(), true, format, v}
	logMessage(lm)
}

func Warn(v ...any) {
	lm := LogMessage{LogLevelWarn, getCallerInfo(), false, "", v}
	logMessage(lm)
}

func Warnf(format string, v ...any) {
	lm := LogMessage{LogLevelWarn, getCallerInfo(), true, format, v}
	logMessage(lm)
}

func Crit(v ...any) {
	lm := LogMessage{LogLevelCrit, getCallerInfo(), false, "", v}
	logMessage(lm)
}

func Critf(format string, v ...any) {
	lm := LogMessage{LogLevelCrit, getCallerInfo(), true, format, v}
	logMessage(lm)
}

func Panic(v ...any) {
	lm := LogMessage{LogLevelPanic, getCallerInfo(), false, "", v}
	handlePanicFatal(lm)
}

func Panicf(format string, v ...any) {
	lm := LogMessage{LogLevelPanic, getCallerInfo(), true, format, v}
	handlePanicFatal(lm)
}

func Fatal(v ...any) {
	lm := LogMessage{LogLevelFatal, getCallerInfo(), false, "", v}
	handlePanicFatal(lm)
}

func Fatalf(format string, v ...any) {
	lm := LogMessage{LogLevelFatal, getCallerInfo(), true, format, v}
	handlePanicFatal(lm)
}

func logMessage(lm LogMessage) {
	if logConfig.logChan != nil {
		logConfig.logChan <- lm
	} else {
		processLogMessage(lm)
	}
}

func handlePanicFatal(lm LogMessage) {
	if logConfig.logChan != nil {
		close(logConfig.logChan)
	}
	if logConfig.errMetricsEnabled.Load() {
		logConfig.errMetrics.ErrGaugeInc(ErrLevel(lm.Level))
	}
	msg := lm.formattedMessage()
	if logConfig.tgEnabled.Load() {
		switch lm.Level {
		case LogLevelPanic:
			if logConfig.tgPanicEnabled.Load() {
				logConfig.tg.ToQueue(fmt.Sprintf("[PANIC] %s", msg))
			}
		case LogLevelFatal:
			if logConfig.tgFatalEnabled.Load() {
				logConfig.tg.ToQueue(fmt.Sprintf("[FATAL] %s", msg))
			}
		}
		logConfig.tg.Stop()
	}
	if lm.Level == LogLevelFatal {
		log.Fatal(lm.coloredMessage(msg))
	} else {
		panic(lm.coloredMessage(msg))
	}
}
