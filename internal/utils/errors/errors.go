package errors

import rpccode "google.golang.org/genproto/googleapis/rpc/code"

//ErouskaError Error with code.
type ErouskaError interface {
	Code() rpccode.Code
	Error() string
}

//CustomError Custom error (who would guess)
type CustomError struct {
	Msg string
}

func (e *CustomError) Error() string {
	return e.Msg
}

//UnknownError Unknown error
type UnknownError struct {
	Msg string
}

func (e *UnknownError) Error() string {
	return e.Msg
}

//Code Code of the error.
func (e *UnknownError) Code() rpccode.Code {
	return rpccode.Code_INTERNAL
}

//MalformedRequestError Error for malformed request
type MalformedRequestError struct {
	Status rpccode.Code
	Msg    string
}

func (mr *MalformedRequestError) Error() string {
	return mr.Msg
}

//Code Code of the error.
func (mr *MalformedRequestError) Code() rpccode.Code {
	return rpccode.Code_INVALID_ARGUMENT
}

//NotFoundError Error for malformed request
type NotFoundError struct {
	Msg string
}

func (mr *NotFoundError) Error() string {
	return mr.Msg
}

//Code Code of the error.
func (mr *NotFoundError) Code() rpccode.Code {
	return rpccode.Code_NOT_FOUND
}
