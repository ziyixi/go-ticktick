package ticktick

const (
	TemplateTime = "2006-01-02T15:04:05.000+0000"
)

// list contains
func Contains[T comparable](s []T, e T) bool {
	for _, v := range s {
		if v == e {
			return true
		}
	}
	return false
}
