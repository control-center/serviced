package logri

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/Sirupsen/logrus"
)

const (
	rootLoggerName              = ""
	markerLevel    logrus.Level = 255
	nilLevel       logrus.Level = 254
)

var (
	// ErrInvalidRootLevel is returned when a nil level is set on the root logger
	ErrInvalidRootLevel = errors.New("The root logger must have a level")
)

// Logger is the wrapper of a Logrus logger. It holds references to its child
// loggers, and manages transactional application of new levels and output
// streams.
type Logger struct {
	mu           sync.Mutex
	Name         string
	parent       *Logger
	absLevel     logrus.Level
	tmpLevel     logrus.Level
	inherit      bool
	lastConfig   LogriConfig
	children     map[string]*Logger
	logger       *logrus.Logger
	outputs      []io.Writer
	localOutputs []io.Writer
}

// NewLoggerFromLogrus creates a new Logri logger tree rooted at a given Logrus
// logger.
func NewLoggerFromLogrus(base *logrus.Logger) *Logger {
	return &Logger{
		Name:         rootLoggerName,
		absLevel:     base.Level,
		tmpLevel:     markerLevel,
		inherit:      true,
		children:     make(map[string]*Logger),
		logger:       base,
		outputs:      []io.Writer{base.Out},
		localOutputs: []io.Writer{},
	}
}

// GetRoot returns the logger at the root of this logger's tree.
func (l *Logger) GetRoot() *Logger {
	next := l
	for next.parent != nil {
		next = next.parent
	}
	return next
}

// GetChild returns a logger that is a child of this logger, creating
// intervening loggers if they do not exist.  If the name given starts with the
// full name of the current logger (i.e., is "absolute"), then the logger
// returned will take that into account, rather than creating a duplicate tree
// below this one.
//
// Example:
//  logger := logri.Logger("a.b.c") // logger.name == "a.b.c"
//	l := logger.GetChild("a.b.c.d") // l.name == "a.b.c.d"
//	l = logger.GetChild("d") // l.name == "a.b.c.d"
//	l = logger.GetChild("b.c.d") // l.name == "a.b.c.b.c.d"
func (l *Logger) GetChild(name string) *Logger {
	if name == "" || name == "*" {
		return l.GetRoot()
	}
	relative := strings.TrimPrefix(name, l.Name+".")
	parent := l
	var (
		changed  bool
		localabs = l.Name
	)
	for _, part := range strings.Split(relative, ".") {
		if localabs == "" {
			localabs = part
		} else {
			localabs = fmt.Sprintf("%s.%s", localabs, part)
		}
		logger, ok := parent.children[part]
		if !ok {
			logger = &Logger{
				Name:     localabs,
				parent:   parent,
				absLevel: nilLevel,
				tmpLevel: markerLevel,
				inherit:  true,
				children: make(map[string]*Logger),
				logger: &logrus.Logger{
					Out:       parent.logger.Out,
					Formatter: parent.logger.Formatter,
					Hooks:     copyHooksExceptLoggerHook(parent.logger.Hooks),
					Level:     parent.logger.Level,
				},
			}
			logger.logger.Hooks.Add(LoggerHook{localabs})
			parent.children[part] = logger
			changed = true
		}
		parent = logger
	}
	if changed && l.GetRoot().lastConfig != nil {
		l.ApplyConfig(l.GetRoot().lastConfig)
	}
	return parent
}

// SetLevel sets the logging level for this logger and children inheriting
// their level from this logger. If inherit is false, the level will be set
// locally only.
func (l *Logger) SetLevel(level logrus.Level, inherit bool) error {
	if err := l.setLevel(level, inherit); err != nil {
		return err
	}
	l.applyTmpState()
	return nil
}

func (l *Logger) addOutput(w io.Writer, inherit bool) {
	if inherit {
		l.outputs = append(l.outputs, w)
	} else {
		l.localOutputs = append(l.localOutputs, w)
	}
}

// SetOutput sets the output to which this logger should write.
func (l *Logger) SetOutput(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logger.Out = w
}

// SetOutputs combines several output writers into one and configures this
// logger to write to that.
func (l *Logger) SetOutputs(writers ...io.Writer) {
	l.SetOutput(io.MultiWriter(writers...))
}

// GetEffectiveLevel returns the effective level of this logger. If this logger
// has no level set locally, it returns the level of its closest ancestor with
// an inheritable level.
func (l *Logger) GetEffectiveLevel() logrus.Level {
	if !l.inherit {
		return l.parent.GetEffectiveLevel()
	}
	if l.tmpLevel != markerLevel {
		return l.tmpLevel
	}
	return l.logger.Level
}

// ApplyConfig applies a Logrus config to a logger tree. Regardless of the
// logger within the tree to which the config is applied, it is treated as the
// root of the tree for purposes of configuring loggers.
func (l *Logger) ApplyConfig(config LogriConfig) error {
	root := l.GetRoot()
	origoutputs, origlocals := root.outputs, root.localOutputs
	root.outputs = []io.Writer{}
	root.localOutputs = []io.Writer{}
	root.resetChildren()
	root.lastConfig = config
	// Loggers are already sorted by hierarchy, so we can apply top down safely
	for _, loggerConfig := range config {
		logger := root.GetChild(loggerConfig.Logger)
		level, err := logrus.ParseLevel(loggerConfig.Level)
		if err != nil {
			// TODO: validate before it gets to this point
			return err
		}
		logger.setLevel(level, !loggerConfig.Local)

		for _, outputConfig := range loggerConfig.Out {
			w, err := GetOutputWriter(outputConfig.Type, outputConfig.Options)
			if err != nil {
				return err
			}
			logger.addOutput(w, !outputConfig.Local)
		}
	}
	if len(root.outputs) == 0 && len(root.localOutputs) == 0 {
		root.outputs = origoutputs
		root.localOutputs = origlocals
	}
	root.propagate()
	root.applyTmpState()
	return nil
}

func (l *Logger) resetChildren() {
	for _, child := range l.children {
		child.absLevel = nilLevel
		child.tmpLevel = markerLevel
		child.inherit = true
		child.outputs = []io.Writer{}
		child.localOutputs = []io.Writer{}
		child.resetChildren()
	}
}

func (l *Logger) setLevel(level logrus.Level, inherit bool) error {
	if level != l.absLevel || l.inherit != inherit {
		if level == nilLevel && l.Name == rootLoggerName {
			return ErrInvalidRootLevel
		}
		l.absLevel = level
		switch level {
		case nilLevel:
			l.tmpLevel = l.parent.GetEffectiveLevel()
			l.inherit = true
		default:
			l.tmpLevel = level
			l.inherit = inherit
		}
	}
	if l.inherit {
		l.propagate()
	}
	return nil
}

// AddHook adds a hook to this logger and all its children
func (l *Logger) AddHook(hook logrus.Hook) {
	l.logger.Hooks.Add(hook)
	for _, child := range l.children {
		child.AddHook(hook)
	}
}

func (l *Logger) propagate() {
	for _, child := range l.children {
		child.inheritLevel(l.GetEffectiveLevel())
		child.inheritOutputs(l.getInheritableOutputs())
		child.propagate()
	}
}

func (l *Logger) getInheritableOutputs() []io.Writer {
	var result []io.Writer
	if l.parent != nil {
		for _, out := range l.parent.getInheritableOutputs() {
			result = append(result, out)
		}
	}
	for _, out := range l.outputs {
		result = append(result, out)
	}
	return dedupeWriters(result...)
}

func (l *Logger) inheritOutputs(writers []io.Writer) {
	l.outputs = dedupeWriters(append(l.outputs, writers...)...)
}

func (l *Logger) inheritLevel(parentLevel logrus.Level) {
	if l.absLevel == nilLevel {
		l.tmpLevel = parentLevel
	}
}

func (l *Logger) applyTmpState() {
	if l.tmpLevel != markerLevel && l.tmpLevel != l.logger.Level {
		l.logger.Level = l.tmpLevel
	}
	l.tmpLevel = markerLevel
	allwriters := append(l.outputs, l.localOutputs...)
	l.SetOutputs(dedupeWriters(allwriters...)...)
	for _, child := range l.children {
		child.applyTmpState()
	}
}

func dedupeWriters(writers ...io.Writer) []io.Writer {
	var val struct{}
	m := map[io.Writer]struct{}{}
	for _, writer := range writers {
		m[writer] = val
	}
	var result []io.Writer
	for writer := range m {
		result = append(result, writer)
	}
	return result
}
