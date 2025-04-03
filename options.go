package kuysor

type Options struct {
	PlaceHolderType PlaceHolderType
	DefaultLimit    int
	StructTag       string
	NullSortMethod  NullSortMethod
}

var (
	options *Options
)

// SetOptions sets the global options, which will be used by all kuysor instances.
// This should be called at the beginning of the application.
func SetOptions(opt Options) {
	options = &opt
}
