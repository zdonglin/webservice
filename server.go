package webservice

import (
	"errors"
	"net/http"
	"reflect"

	"github.com/labstack/echo"
)

var (
	// ErrMethodNotSupported are returned when a http has a non-supported method
	ErrMethodNotSupported = errors.New("the method is not supported")
)

// Callback is called given the pointer of request info struct
// and expecting a http status code, a response body and an error
// body has the same type of BodyTemplate in RESTConfig
type Callback func(*RESTRequest) (int, interface{}, error)

// RESTConfig is a configuration pack for one RESTful API
type RESTConfig struct {
	Path         string
	Method       string
	BodyTemplate interface{}
	Callback     Callback
}

// RESTRequest contains path params, query params and request body of a request
type RESTRequest struct {
	PathParams  map[string]string
	QueryParams map[string][]string
	Body        interface{}

	// in case for some special situation, usually just ignore it
	Ctx echo.Context
}

// errorMessage is the response body carrying the error message
type errorMessage struct {
	Message string `json:"message"`
}

func (config *RESTConfig) httpHandlerFn() echo.HandlerFunc {
	var handleFn echo.HandlerFunc
	if config.Method == http.MethodGet || config.Method == http.MethodDelete {
		// GET and DELETE requests has no body
		handleFn = func(ctx echo.Context) error {
			restRequest := packPathParamsAndQueryParamsAndContextIntoRESTRequest(ctx)
			statusCode, respBody, err := config.Callback(restRequest)
			if err != nil {
				return ctx.JSON(statusCode, &errorMessage{Message: err.Error()})
			}
			if respBody == nil {
				return ctx.NoContent(statusCode)
			}
			return ctx.JSON(statusCode, respBody)
		}
	} else {
		bodyType := reflect.TypeOf(config.BodyTemplate).Elem()
		// handler function begins here
		handleFn = func(ctx echo.Context) error {
			//create a new restRequest and fullfil it
			restRequest := packPathParamsAndQueryParamsAndContextIntoRESTRequest(ctx)
			restRequest.Body = reflect.New(bodyType).Interface()
			if err := ctx.Bind(restRequest.Body); err != nil {
				return err
			}
			statusCode, respBody, err := config.Callback(restRequest)
			if err != nil {
				return ctx.JSON(statusCode, &errorMessage{Message: err.Error()})
			}
			if respBody == nil {
				return ctx.NoContent(statusCode)
			}
			return ctx.JSON(statusCode, respBody)
		} // handler function ends here
	}
	return handleFn
}

func packPathParamsAndQueryParamsAndContextIntoRESTRequest(ctx echo.Context) *RESTRequest {
	restRequest := &RESTRequest{}
	restRequest.PathParams = iteratePathParamsAndStoreIntoMap(ctx)
	restRequest.QueryParams = map[string][]string(ctx.QueryParams())
	restRequest.Ctx = ctx
	return restRequest
}

func iteratePathParamsAndStoreIntoMap(ctx echo.Context) map[string]string {
	kv := make(map[string]string)
	ps := ctx.ParamNames()
	for _, paramName := range ps {
		kv[paramName] = ctx.Param(paramName)
	}
	return kv
}

// RESTService is the main component which holds the actual HTTP server internally
// and register all entries
type RESTService struct {
	ListenAddr string

	server *echo.Echo
}

// NewRESTService creates a new *RESTService
func NewRESTService(configs []*RESTConfig) (*RESTService, error) {
	e := echo.New()
	for _, config := range configs {
		switch config.Method {
		case http.MethodGet:
			e.GET(config.Path, config.httpHandlerFn())
		case http.MethodPost:
			e.POST(config.Path, config.httpHandlerFn())
		case http.MethodPut:
			e.PUT(config.Path, config.httpHandlerFn())
		case http.MethodPatch:
			e.PATCH(config.Path, config.httpHandlerFn())
		case http.MethodDelete:
			e.DELETE(config.Path, config.httpHandlerFn())
		default:
			return nil, ErrMethodNotSupported
		}
	}
	return &RESTService{server: e}, nil
}

// Serve starts the server and blocks
func (server *RESTService) Serve() error {
	return server.server.Start(server.ListenAddr)
}
