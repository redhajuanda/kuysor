package kuysor

type Instance struct {
	options *Options
}

// NewInstance creates a new Kuysor instance with the given options.
// It is useful for grouping multiple queries with the same options.
// For example, if you manage multiple databases with different options, let's say you have Postgres and MySQL databases in the same application,
// you can create an instance for each database with different options, one with PlaceHolderType=Dollar and the other with PlaceHolderType=Question.
func NewInstance(opt ...Options) *Instance {

	i := &Instance{}
	if len(opt) > 0 { // override the options
		i.options = &opt[0]
	} else {
		i.options = getGlobalOptions()
	}

	return i

}

// NewQuery creates a new Kuysor instance.
// It accepts the SQL query.
func (i *Instance) NewQuery(query string) *Kuysor {

	p := &Kuysor{
		sql: query,
	}

	p.options = i.options

	return p

}
