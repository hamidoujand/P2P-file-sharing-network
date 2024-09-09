package cmd

// Runner representa all required behaviors that a command needs to have.
type Runner interface {
	Run() error
	Init([]string) error
	Name() string
}
