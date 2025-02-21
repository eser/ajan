package httpfx

import (
	"net/http"

	"github.com/eser/ajan/httpfx/uris"
)

type RouteParameterType int

const (
	RouteParameterTypeHeader RouteParameterType = iota
	RouteParameterTypeQuery
	RouteParameterTypePath
	RouteParameterTypeBody
)

type RouterParameterValidator func(inputString string) (string, error)

type RouterParameter struct {
	Name        string
	Description string
	Validators  []RouterParameterValidator
	Type        RouteParameterType
	IsRequired  bool
}

type RouteOpenApiSpecRequest struct {
	Model any
}

type RouteOpenApiSpecResponse struct {
	Model      any
	StatusCode int
	HasModel   bool
}

type RouteOpenApiSpec struct {
	OperationId string
	Summary     string
	Description string
	Tags        []string

	Requests   []RouteOpenApiSpecRequest
	Responses  []RouteOpenApiSpecResponse
	Deprecated bool
}

type Route struct {
	Pattern        *uris.Pattern
	Parameters     []RouterParameter
	Handlers       []Handler
	MuxHandlerFunc func(http.ResponseWriter, *http.Request)

	Spec RouteOpenApiSpec
}

func (r *Route) HasOperationId(operationId string) *Route {
	r.Spec.OperationId = operationId

	return r
}

func (r *Route) HasSummary(summary string) *Route {
	r.Spec.Summary = summary

	return r
}

func (r *Route) HasDescription(description string) *Route {
	r.Spec.Description = description

	return r
}

func (r *Route) HasTags(tags ...string) *Route {
	r.Spec.Tags = tags

	return r
}

func (r *Route) IsDeprecated() *Route {
	r.Spec.Deprecated = true

	return r
}

func (r *Route) HasPathParameter(name string, description string) *Route {
	r.Parameters = append(r.Parameters, RouterParameter{
		Type:        RouteParameterTypePath,
		Name:        name,
		Description: description,
		IsRequired:  true,

		Validators: []RouterParameterValidator{
			// func(inputString string) (string, error) {
			// 	return inputString, nil
			// },
		},
	})

	return r
}

func (r *Route) HasQueryParameter(name string, description string) *Route {
	r.Parameters = append(r.Parameters, RouterParameter{
		Type:        RouteParameterTypeQuery,
		Name:        name,
		Description: description,
		IsRequired:  true,

		Validators: []RouterParameterValidator{
			// func(inputString string) (string, error) {
			// 	return inputString, nil
			// },
		},
	})

	return r
}

func (r *Route) HasRequestModel(model any) *Route {
	r.Spec.Requests = append(r.Spec.Requests, RouteOpenApiSpecRequest{
		Model: model,
	})

	return r
}

func (r *Route) HasResponse(statusCode int) *Route {
	r.Spec.Responses = append(r.Spec.Responses, RouteOpenApiSpecResponse{
		StatusCode: statusCode,
		HasModel:   false,
		Model:      nil,
	})

	return r
}

func (r *Route) HasResponseModel(statusCode int, model any) *Route {
	r.Spec.Responses = append(r.Spec.Responses, RouteOpenApiSpecResponse{
		StatusCode: statusCode,
		HasModel:   true,
		Model:      model,
	})

	return r
}
