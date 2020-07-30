package errors

type CustomError struct {
	Msg string
}

func (e *CustomError) Error() string {
	return e.Msg
}

type MalformedRequestError struct {
	Status int
	Msg    string
}

func (mr *MalformedRequestError) Error() string {
	return mr.Msg
}
