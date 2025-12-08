package main

type exitError struct {
	code   int
	msg    string
	silent bool
}

func (e *exitError) Error() string {
	if e == nil || e.msg == "" {
		return "exit"
	}
	return e.msg
}
