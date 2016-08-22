package logri

import (
	"io/ioutil"
	"path/filepath"

	"github.com/Sirupsen/logrus"
	"github.com/fsnotify/fsnotify"
)

var (
	// RootLogger is the default logger tree.
	RootLogger = NewLoggerFromLogrus(logrus.New())
	// Verify we implement the right interface
	_ logrus.FieldLogger = RootLogger
)

// GetLogger returns a logger from the default tree with the given name.
func GetLogger(name string) *Logger {
	return RootLogger.GetChild(name)
}

// SetLevel sets the level of the root logger and its descendants
func SetLevel(level logrus.Level) {
	RootLogger.SetLevel(level, true)
}

// ApplyConfig applies configuration to the default tree.
func ApplyConfig(config LogriConfig) error {
	return RootLogger.ApplyConfig(config)
}

// ApplyConfigFromFile reads logging configuration from a file and applies it
// to the default tree.
func ApplyConfigFromFile(file string) error {
	bytes, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	cfg, err := ConfigFromBytes(bytes)
	if err != nil {
		return err
	}
	return ApplyConfig(cfg)
}

// WatchConfigFile watches a given config file, applying the config on change
func WatchConfigFile(file string) error {
	// Set up an fsnotify watcher
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer w.Close()

	// Get clean versions of the filename and its directory
	cleanPath := filepath.Clean(file)
	cleanDir, _ := filepath.Split(cleanPath)

	// What event operations do we care about
	ops := fsnotify.Write | fsnotify.Create

	// Start watching the directory
	w.Add(cleanDir)

	err = nil
	for {
		select {
		case e := <-w.Events:
			// See if the event is for our config file
			if filepath.Clean(e.Name) == cleanPath {
				// It is, so check the operation. If it's a write or create, update.
				if e.Op&ops > 0 {
					err = ApplyConfigFromFile(cleanPath)
				}
			}
		case err = <-w.Errors:
		}
		if err != nil {
			WithError(err).WithFields(logrus.Fields{
				"file": cleanPath,
				"dir":  cleanDir,
			}).Warning("Unable to read logging config file")
		}
		err = nil
	}
}

// Adds the given hook to every logger in the tree
func AddHook(hook logrus.Hook) {
	RootLogger.AddHook(hook)
}

// WithField creates an entry from the root logger and adds a field to
// it. If you want multiple fields, use `WithFields`.
//
// Note that it doesn't log until you call Debug, Print, Info, Warn, Fatal
// or Panic on the Entry it returns.
func WithField(key string, value interface{}) *logrus.Entry {
	return RootLogger.WithField(key, value)
}

// WithFields creates an entry from the root logger and adds multiple
// fields to it. This is simply a helper for `WithField`, invoking it
// once for each field.
//
// Note that it doesn't log until you call Debug, Print, Info, Warn, Fatal
// or Panic on the Entry it returns.
func WithFields(fields logrus.Fields) *logrus.Entry {
	return RootLogger.WithFields(fields)
}

// WithError creates an entry from the root logger and adds an error to it,
// using the value defined in ErrorKey as key.
func WithError(err error) *logrus.Entry {
	return RootLogger.WithError(err)
}

// Debugf logs a message at level Debug on the standard logger.
func Debugf(format string, args ...interface{}) {
	RootLogger.Debugf(format, args...)
}

// Infof logs a message at level Info on the standard logger.
func Infof(format string, args ...interface{}) {
	RootLogger.Infof(format, args...)
}

// Printf logs a message at level Info on the standard logger.
func Printf(format string, args ...interface{}) {
	RootLogger.Printf(format, args...)
}

// Warnf logs a message at level Warn on the standard logger.
func Warnf(format string, args ...interface{}) {
	RootLogger.Warnf(format, args...)
}

// Warningf logs a message at level Warn on the standard logger.
func Warningf(format string, args ...interface{}) {
	RootLogger.Warningf(format, args...)
}

// Errorf logs a message at level Error on the standard logger.
func Errorf(format string, args ...interface{}) {
	RootLogger.Errorf(format, args...)
}

// Fatalf logs a message at level Fatal on the standard logger.
func Fatalf(format string, args ...interface{}) {
	RootLogger.Fatalf(format, args...)
}

// Panicf logs a message at level Panic on the standard logger.
func Panicf(format string, args ...interface{}) {
	RootLogger.Panicf(format, args...)
}

// Debug logs a message at level Debug on the standard logger.
func Debug(args ...interface{}) {
	RootLogger.Debug(args...)
}

// Info logs a message at level Info on the standard logger.
func Info(args ...interface{}) {
	RootLogger.Info(args...)
}

// Print logs a message at level Info on the standard logger.
func Print(args ...interface{}) {
	RootLogger.Print(args...)
}

// Warn logs a message at level Warn on the standard logger.
func Warn(args ...interface{}) {
	RootLogger.Warn(args...)
}

// Warning logs a message at level Warn on the standard logger.
func Warning(args ...interface{}) {
	RootLogger.Warning(args...)
}

// Error logs a message at level Error on the standard logger.
func Error(args ...interface{}) {
	RootLogger.Error(args...)
}

// Fatal logs a message at level Fatal on the standard logger.
func Fatal(args ...interface{}) {
	RootLogger.Fatal(args...)
}

// Panic logs a message at level Panic on the standard logger.
func Panic(args ...interface{}) {
	RootLogger.Panic(args...)
}

// Debugln logs a message at level Debug on the standard logger.
func Debugln(args ...interface{}) {
	RootLogger.Debugln(args...)
}

// Infoln logs a message at level Info on the standard logger.
func Infoln(args ...interface{}) {
	RootLogger.Infoln(args...)
}

// Println logs a message at level Info on the standard logger.
func Println(args ...interface{}) {
	RootLogger.Println(args...)
}

// Warnln logs a message at level Warn on the standard logger.
func Warnln(args ...interface{}) {
	RootLogger.Warnln(args...)
}

// Warningln logs a message at level Warn on the standard logger.
func Warningln(args ...interface{}) {
	RootLogger.Warningln(args...)
}

// Errorln logs a message at level Error on the standard logger.
func Errorln(args ...interface{}) {
	RootLogger.Errorln(args...)
}

// Fatalln logs a message at level Fatal on the standard logger.
func Fatalln(args ...interface{}) {
	RootLogger.Fatalln(args...)
}

// Panicln logs a message at level Panic on the standard logger.
func Panicln(args ...interface{}) {
	RootLogger.Panicln(args...)
}
