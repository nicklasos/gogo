package testkit

import (
	"encoding/json"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"app/internal/errs"
)

// Response is what a request returned. Its Assert methods stop the test on failure and
// return the response, so they chain:
//
//	app.GET("/api/v1/examples").AssertOK().AssertJSONCount("data", 3)
type Response struct {
	StatusCode int
	Body       []byte
	Header     http.Header

	t       *testing.T
	request string
}

func (r *Response) AssertStatus(status int) *Response {
	r.t.Helper()
	if r.StatusCode != status {
		r.t.Fatalf("%s: expected status %d, got %d\n%s", r.request, status, r.StatusCode, r.Body)
	}
	return r
}

func (r *Response) AssertOK() *Response {
	r.t.Helper()
	return r.AssertStatus(http.StatusOK)
}

func (r *Response) AssertBadRequest() *Response {
	r.t.Helper()
	return r.AssertStatus(http.StatusBadRequest)
}

func (r *Response) AssertUnauthorized() *Response {
	r.t.Helper()
	return r.AssertStatus(http.StatusUnauthorized)
}

func (r *Response) AssertForbidden() *Response {
	r.t.Helper()
	return r.AssertStatus(http.StatusForbidden)
}

func (r *Response) AssertNotFound() *Response {
	r.t.Helper()
	return r.AssertStatus(http.StatusNotFound)
}

func (r *Response) AssertTooManyRequests() *Response {
	r.t.Helper()
	return r.AssertStatus(http.StatusTooManyRequests)
}

// AssertError checks the status and the error_key of an error response.
func (r *Response) AssertError(status int, key string) *Response {
	r.t.Helper()
	return r.AssertStatus(status).AssertJSONPath("error_key", key)
}

// AssertInvalid checks that validation failed, and that each given field has an error.
func (r *Response) AssertInvalid(fields ...string) *Response {
	r.t.Helper()
	r.AssertError(http.StatusBadRequest, errs.ErrKeyValidationFailed)
	for _, field := range fields {
		if _, ok := r.lookup("errors." + field); !ok {
			r.t.Fatalf("%s: expected a validation error for %q\n%s", r.request, field, r.Body)
		}
	}
	return r
}

// AssertJSONPath checks the value at a dot-separated path, where a number is an array index:
//
//	resp.AssertJSONPath("data.0.title", "Hello")
func (r *Response) AssertJSONPath(path string, want any) *Response {
	r.t.Helper()
	got, ok := r.lookup(path)
	if !ok {
		r.t.Fatalf("%s: no %q in the response\n%s", r.request, path, r.Body)
	}
	if !reflect.DeepEqual(got, asJSON(r.t, want)) {
		r.t.Fatalf("%s: expected %q to be %v, got %v\n%s", r.request, path, want, got, r.Body)
	}
	return r
}

// AssertJSONCount checks the length of the array at the path.
func (r *Response) AssertJSONCount(path string, count int) *Response {
	r.t.Helper()
	got, ok := r.lookup(path)
	items, isArray := got.([]any)
	if !ok || !isArray {
		r.t.Fatalf("%s: expected an array at %q\n%s", r.request, path, r.Body)
	}
	if len(items) != count {
		r.t.Fatalf("%s: expected %d items at %q, got %d\n%s", r.request, count, path, len(items), r.Body)
	}
	return r
}

// AssertJSONMissing checks that the response has nothing at the path.
func (r *Response) AssertJSONMissing(path string) *Response {
	r.t.Helper()
	if _, ok := r.lookup(path); ok {
		r.t.Fatalf("%s: expected no %q in the response\n%s", r.request, path, r.Body)
	}
	return r
}

// JSON decodes the body into v, which is a pointer:
//
//	var page example.PaginatedExamplesResponse
//	app.GET("/api/v1/examples").AssertOK().JSON(&page)
func (r *Response) JSON(v any) *Response {
	r.t.Helper()
	if err := json.Unmarshal(r.Body, v); err != nil {
		r.t.Fatalf("%s: decode response: %v\n%s", r.request, err, r.Body)
	}
	return r
}

func (r *Response) String() string {
	return string(r.Body)
}

// Decode is JSON for when a variable declaration is in the way:
//
//	page := testkit.Decode[example.PaginatedExamplesResponse](resp)
func Decode[T any](r *Response) T {
	r.t.Helper()
	var v T
	r.JSON(&v)
	return v
}

func (r *Response) lookup(path string) (any, bool) {
	r.t.Helper()

	var current any
	if err := json.Unmarshal(r.Body, &current); err != nil {
		r.t.Fatalf("%s: response is not JSON: %v\n%s", r.request, err, r.Body)
	}

	for _, key := range strings.Split(path, ".") {
		switch node := current.(type) {
		case map[string]any:
			value, ok := node[key]
			if !ok {
				return nil, false
			}
			current = value
		case []any:
			index, err := strconv.Atoi(key)
			if err != nil || index < 0 || index >= len(node) {
				return nil, false
			}
			current = node[index]
		default:
			return nil, false
		}
	}
	return current, true
}

// asJSON turns a Go value into what decoding its JSON gives, so that int32(5) equals the 5 in a body.
func asJSON(t *testing.T, value any) any {
	t.Helper()

	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("testkit: encode expected value: %v", err)
	}
	var decoded any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("testkit: decode expected value: %v", err)
	}
	return decoded
}
