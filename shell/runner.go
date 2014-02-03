package shell

type Runner interface {
        Reader(size_t int)
        Write(data []byte) (int, error)
        StdoutPipe() chan string
        StderrPipe() chan string
        ExitedPipe() chan bool
        Error() error
        Signal(signal os.Signal) error
        Kill() error
        Close()
}
