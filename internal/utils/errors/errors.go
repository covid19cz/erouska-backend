package errors

//CustomError Custom error (who would guess)
type CustomError struct {
	Msg string
}

func (e *CustomError) Error() string {
	return e.Msg
}

//MalformedRequestError Error for malformed request
type MalformedRequestError struct {
	Status int
	Msg    string
}

func (mr *MalformedRequestError) Error() string {
	return mr.Msg
}
