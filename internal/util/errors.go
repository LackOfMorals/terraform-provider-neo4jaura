package util

type DiagnosticsError struct {
	Message string
	Details string
}

func NoDiagnosticsError() DiagnosticsError {
	return DiagnosticsError{}
}

func NewDiagnosticsError(message, detail string) DiagnosticsError {
	return DiagnosticsError{Message: message, Details: detail}
}

func (de DiagnosticsError) IsNotEmpty() bool {
	return de != NoDiagnosticsError()
}
