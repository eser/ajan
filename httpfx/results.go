package httpfx

import (
	"encoding/json"
	"net/http"
)

// Result Options.
type ResultOption func(*Result)

func WithBody(body []byte) ResultOption {
	return func(result *Result) {
		result.InnerBody = body
	}
}

func WithPlainText(body string) ResultOption {
	return func(result *Result) {
		result.InnerBody = []byte(body)
	}
}

func WithJson(body any) ResultOption {
	return func(result *Result) {
		encoded, err := json.Marshal(body)
		if err != nil {
			result.InnerBody = []byte("Failed to encode JSON")
			result.InnerStatusCode = http.StatusInternalServerError

			return
		}

		result.InnerBody = encoded
	}
}

// Results With Options.
type Results struct{}

func (r *Results) Ok(options ...ResultOption) Result {
	result := Result{ //nolint:exhaustruct
		InnerStatusCode: http.StatusNoContent,
		InnerBody:       make([]byte, 0),
	}

	for _, option := range options {
		option(&result)
	}

	return result
}

func (r *Results) Accepted(options ...ResultOption) Result {
	result := Result{ //nolint:exhaustruct
		InnerStatusCode: http.StatusAccepted,
		InnerBody:       make([]byte, 0),
	}

	for _, option := range options {
		option(&result)
	}

	return result
}

func (r *Results) NotFound(options ...ResultOption) Result {
	result := Result{ //nolint:exhaustruct
		InnerStatusCode: http.StatusNotFound,
		InnerBody:       []byte("Not Found"),
	}

	for _, option := range options {
		option(&result)
	}

	return result
}

func (r *Results) Unauthorized(options ...ResultOption) Result {
	result := Result{ //nolint:exhaustruct
		InnerStatusCode: http.StatusUnauthorized,
		InnerBody:       make([]byte, 0),
	}

	for _, option := range options {
		option(&result)
	}

	return result
}

func (r *Results) BadRequest(options ...ResultOption) Result {
	result := Result{ //nolint:exhaustruct
		InnerStatusCode: http.StatusBadRequest,
		InnerBody:       []byte("Bad Request"),
	}

	for _, option := range options {
		option(&result)
	}

	return result
}

func (r *Results) Error(statusCode int, options ...ResultOption) Result {
	result := Result{ //nolint:exhaustruct
		InnerStatusCode: statusCode,
		InnerBody:       make([]byte, 0),
	}

	for _, option := range options {
		option(&result)
	}

	return result
}

// Results Without Options.
func (r *Results) Bytes(body []byte) Result {
	return Result{ //nolint:exhaustruct
		InnerStatusCode: http.StatusOK,
		InnerBody:       body,
	}
}

func (r *Results) PlainText(body []byte) Result {
	return Result{ //nolint:exhaustruct
		InnerStatusCode: http.StatusOK,
		InnerBody:       body,
	}
}

func (r *Results) Json(body any) Result {
	encoded, err := json.Marshal(body)
	if err != nil {
		// TODO(@eser) Log error
		return Result{ //nolint:exhaustruct
			InnerStatusCode: http.StatusInternalServerError,
			InnerBody:       []byte("Failed to encode JSON"),
		}
	}

	return Result{ //nolint:exhaustruct
		InnerStatusCode: http.StatusOK,
		InnerBody:       encoded,
	}
}

func (r *Results) Redirect(uri string) Result {
	return Result{ //nolint:exhaustruct
		InnerStatusCode:    http.StatusTemporaryRedirect,
		InnerBody:          make([]byte, 0),
		InnerRedirectToUri: uri,
	}
}

func (r *Results) Abort() Result {
	// TODO(@eser) implement this
	return Result{ //nolint:exhaustruct
		InnerStatusCode: http.StatusNotImplemented,
		InnerBody:       []byte("Not Implemented"),
	}
}
