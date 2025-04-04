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

// SetGlobalOptions sets the global options, which will be used by all kuysor instances.
// This should be called at the beginning of the application.
func SetGlobalOptions(opt Options) {
	options = &opt
}

// getGlobalOptions returns global options
// if the global options never setted, it will set default options
func getGlobalOptions() *Options {
	if options == nil {
		options = &Options{
			PlaceHolderType: Question,
			DefaultLimit:    defaulLimit,
			StructTag:       defaultStructTag,
			NullSortMethod:  defaultNullSortMethod,
		}
	}

	return options
}
