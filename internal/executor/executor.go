package executor

type Executor interface {
	Execute() (float32, error)
}