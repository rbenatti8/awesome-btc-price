package web

type invalidParamError struct {
	message string
	details map[string]string
}

func (e invalidParamError) Error() string {
	return e.message
}

func (e invalidParamError) Details() map[string]string {
	return e.details
}
