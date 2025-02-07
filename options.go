package kuysor

type Options struct {
	Dialect Dialect
}

type Dialect string

const (
	MySQL Dialect = "mysql"
)

var (
	options *Options
)

// SetOptions sets the global options, which will be used by all kuysor instances.
// This should be called at the beginning of the application.
func SetOptions(opt Options) {
	options = &opt
}
