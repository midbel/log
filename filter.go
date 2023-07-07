package log

type filterfunc func(Entry) bool

func parseFilter(str string) (filterfunc, error) {
	if str == "" {
		return func(_ Entry) bool { return true }, nil
	}
	return nil, nil
}
