package logri

import "github.com/Sirupsen/logrus"

// LoggerHook is a Logrus hook that adds the logger name as a field to the entry
type LoggerHook struct {
	loggerName string
}

// Levels satisfies the logrus.Hook interface
func (hook LoggerHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// Fire satisfies the logrus.Hook interface
func (hook LoggerHook) Fire(entry *logrus.Entry) error {
	entry.SetField("logger", hook.loggerName)
	return nil
}

func copyHooksExceptLoggerHook(orig logrus.LevelHooks) logrus.LevelHooks {
	result := logrus.LevelHooks{}
	for level, hooks := range orig {
		for _, hook := range hooks {
			if _, ok := hook.(LoggerHook); !ok {
				result[level] = append(result[level], hook)
			}
		}
	}
	return result
}
