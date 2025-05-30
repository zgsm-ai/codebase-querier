package job

type Job interface {
	Start()
	Close()
}

type Processor interface {
	Process() error
}
