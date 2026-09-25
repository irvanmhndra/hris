package apperror

type Error struct {
	Status  int
	Message string
}

func (e *Error) Error() string     { return e.Message }
func Invalid(message string) error { return &Error{422, message} }

var ErrOverlap = &Error{409, "Tanggal cuti bertumpang tindih dengan pengajuan sebelumnya"}
