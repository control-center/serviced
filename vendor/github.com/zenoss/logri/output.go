package logri

import (
	"bytes"
	"errors"
	"io"
	"os"
	"runtime"
	"sync"
)

type OutputType string

const (
	FileOutput   OutputType = "file"
	StdoutOutput            = "stdout"
	StderrOutput            = "stderr"
	TestOutput              = "test" // Used for tests only
)

var (
	ErrInvalidOutputOptions = errors.New("Insufficient or invalid options were given for an output")

	// Registry of file outputs
	fileOutputRegistry = make(map[string]io.Writer)

	// Registry of test outputs
	testOutputRegistry = make(map[string]*bytes.Buffer)
	mu                 sync.Mutex
)

func GetOutputWriter(outtype OutputType, options map[string]string) (io.Writer, error) {
	switch outtype {
	case FileOutput:

		// FileOutput type requires an option called "file," specifying the
		// file to be logged to. If it doesn't exist, it's invalid config.
		file, ok := options["file"]
		if !ok {
			return nil, ErrInvalidOutputOptions
		}

		// Look to see if we have a writer open already
		mu.Lock()
		defer mu.Unlock()
		if writer, ok := fileOutputRegistry[file]; ok {
			return writer, nil
		}

		// Open the file for appending, creating if it exists, and save the
		// writer for later access by other loggers.
		writer, err := os.OpenFile(file, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			return nil, err
		}
		fileOutputRegistry[file] = writer

		// Close the file if it gets GCed
		runtime.SetFinalizer(writer, finalizeFile)

		return writer, nil

	case StdoutOutput:
		return os.Stdout, nil

	case StderrOutput:
		return os.Stderr, nil

	case TestOutput:
		name, ok := options["name"]
		if !ok {
			return nil, ErrInvalidOutputOptions
		}
		mu.Lock()
		defer mu.Unlock()
		if writer, ok := testOutputRegistry[name]; ok {
			return writer, nil
		}
		var writer bytes.Buffer
		testOutputRegistry[name] = &writer
		return &writer, nil
	}
	return nil, ErrInvalidOutputOptions
}

func finalizeFile(f *os.File) {
	mu.Lock()
	defer mu.Unlock()
	delete(fileOutputRegistry, f.Name())
	f.Close()
}
