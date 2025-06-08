package results

type ResultKind string

const (
	ResultKindSuccess ResultKind = "success"
	ResultKindError   ResultKind = "error"
)

var Ok = Define( //nolint:gochecknoglobals
	ResultKindSuccess,
	"OK",
	"OK",
)
