package log

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/arr-ai/frozen"
	"github.com/sirupsen/logrus"
)

type logrusLevelConfig interface {
	getLogrusLevel() logrus.Level
}

type ioOutConfig interface {
	getIoOut() io.Writer
}

type standardLogger struct {
	internal *logrus.Logger
	fields   frozen.Map
}

func (sf standardFormat) Format(entry *LogEntry) (string, error) {
	message := strings.Builder{}
	message.WriteString(entry.Time.Format(time.RFC3339Nano))
	message.WriteByte(' ')

	if entry.Data.Count() != 0 {
		message.WriteString(getFormattedField(entry.Data))
		message.WriteByte(' ')
	}

	message.WriteString(strings.ToUpper(verboseToLogrusLevel(entry.Verbose).String()))
	message.WriteByte(' ')

	if entry.Message != "" {
		message.WriteString(entry.Message)
		message.WriteByte(' ')
	}

	// TODO: add codelinker's message here
	message.WriteByte('\n')
	return message.String(), nil
}

func (jf jsonFormat) Format(entry *LogEntry) (string, error) {
	jsonFile := make(map[string]interface{})
	jsonFile["timestamp"] = entry.Time.Format(time.RFC3339Nano)
	jsonFile["message"] = entry.Message
	jsonFile["level"] = strings.ToUpper(verboseToLogrusLevel(entry.Verbose).String())
	if entry.Data.Count() != 0 {
		fields := make(map[string]interface{})
		for i := entry.Data.Range(); i.Next(); {
			fields[i.Key().(string)] = i.Value()
		}
		jsonFile["fields"] = fields
	}
	data, err := json.Marshal(jsonFile)
	if err != nil {
		return "", err
	}
	return string(data) + "\n", nil
}

// NewStandardLogger returns a logger with a standard formater
func NewStandardLogger() Logger {
	logger := logrus.New()
	logger.SetFormatter(&pkgFormatterToLogrusFormatter{&standardFormat{}})

	return &standardLogger{internal: logger}
}

func (sl *standardLogger) Debug(args ...interface{}) {
	sl.log(true, args...)
}

func (sl *standardLogger) Debugf(format string, args ...interface{}) {
	sl.logf(true, format, args...)
}

func (sl *standardLogger) Error(errMsg error, args ...interface{}) {
	if msg, _ := sl.fields.Get(errMsgKey); msg != errMsg.Error() {
		sl.fields = sl.fields.With(errMsgKey, errMsg.Error())
	}
	sl.log(false, args...)
}

func (sl *standardLogger) Errorf(errMsg error, format string, args ...interface{}) {
	if msg, _ := sl.fields.Get(errMsgKey); msg != errMsg.Error() {
		sl.fields = sl.fields.With(errMsgKey, errMsg.Error())
	}
	sl.logf(false, format, args...)
}

func (sl *standardLogger) Info(args ...interface{}) {
	sl.log(false, args...)
}

func (sl *standardLogger) Infof(format string, args ...interface{}) {
	sl.logf(false, format, args...)
}

func (sl *standardLogger) PutFields(fields frozen.Map) Logger {
	sl.fields = fields
	return sl
}

func (sl *standardLogger) SetFormatter(formatter Config) error {
	switch f := formatter.(type) {
	case Formatter:
		sl.internal.SetFormatter(&pkgFormatterToLogrusFormatter{f})
		return nil
	case logrus.Formatter: // deprecated. provided for legacy support only
		sl.internal.SetFormatter(f)
		return nil
	default:
		return errors.New("formatter must be pkg.Formatter or logrus.Formatter")
	}
}

func (sl *standardLogger) SetVerbose(on bool) error {
	if on {
		sl.internal.SetLevel(logrus.DebugLevel)
	} else {
		sl.internal.SetLevel(logrus.InfoLevel)
	}
	return nil
}

func (sl *standardLogger) SetOutput(w io.Writer) error {
	sl.internal.SetOutput(w)
	return nil
}

func (sl *standardLogger) Copy() Logger {
	return &standardLogger{sl.getCopiedInternalLogger(), sl.fields}
}

func (sl *standardLogger) log(verbose bool, args ...interface{}) {
	logWithLogrus(sl.internal, &LogEntry{
		Time:    time.Now(),
		Message: fmt.Sprint(args...),
		Data:    sl.fields,
		Verbose: verbose,
	})
}

func (sl *standardLogger) logf(verbose bool, format string, args ...interface{}) {
	logWithLogrus(sl.internal, &LogEntry{
		Time:    time.Now(),
		Message: fmt.Sprintf(format, args...),
		Data:    sl.fields,
		Verbose: verbose,
	})
}

func getFormattedField(fields frozen.Map) string {
	if fields.Count() == 0 {
		return ""
	}

	formattedFields := strings.Builder{}
	i := fields.Range()
	i.Next()
	formattedFields.WriteString(fmt.Sprintf("%v=%v", i.Key(), i.Value()))
	for i.Next() {
		formattedFields.WriteString(fmt.Sprintf(" %v=%v", i.Key(), i.Value()))
	}
	return formattedFields.String()
}

func (sl *standardLogger) getCopiedInternalLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetFormatter(sl.internal.Formatter)
	logger.SetLevel(sl.internal.Level)
	logger.SetOutput(sl.internal.Out)

	return logger
}
