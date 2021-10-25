package logmgr

type query interface {
	QueryBody() string
	QueryArgs() []interface{}
}

func tryCoerceQuery(o interface{}, action func(q query)) bool {
	if asQuery, ok := o.(query); ok {
		action(asQuery)
		return true
	}
	return false
}
