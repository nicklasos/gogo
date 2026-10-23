package testkit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"
)

// Client makes requests to the test server. A path can take arguments, as in fmt.Sprintf:
//
//	app.GET("/api/v1/examples/%d", example.ID)
//
// A body can be a struct or map (sent as JSON), or raw JSON as a string, []byte or io.Reader.
type Client struct {
	app     *App
	headers map[string]string
}

// WithHeader returns a client that also sends the header. The original is not changed.
func (c *Client) WithHeader(key, value string) *Client {
	headers := map[string]string{key: value}
	for k, v := range c.headers {
		if k != key {
			headers[k] = v
		}
	}
	return &Client{app: c.app, headers: headers}
}

func (c *Client) GET(path string, args ...any) *Response {
	c.app.t.Helper()
	return c.send(http.MethodGet, format(path, args), nil, "")
}

func (c *Client) POST(path string, body any, args ...any) *Response {
	c.app.t.Helper()
	return c.sendJSON(http.MethodPost, format(path, args), body)
}

func (c *Client) PUT(path string, body any, args ...any) *Response {
	c.app.t.Helper()
	return c.sendJSON(http.MethodPut, format(path, args), body)
}

func (c *Client) PATCH(path string, body any, args ...any) *Response {
	c.app.t.Helper()
	return c.sendJSON(http.MethodPatch, format(path, args), body)
}

func (c *Client) DELETE(path string, args ...any) *Response {
	c.app.t.Helper()
	return c.send(http.MethodDelete, format(path, args), nil, "")
}

// format takes the arguments as a slice so that vet does not treat the verbs as printf wrappers:
// a path held in a variable, or one with a literal % in its query, is then fine to pass.
func format(path string, args []any) string {
	if len(args) == 0 {
		return path
	}
	return fmt.Sprintf(path, args...)
}

// File is one uploaded file of a multipart request.
type File struct {
	Name    string
	Content []byte
	// ContentType is what the client claims the file is. Empty means application/octet-stream.
	ContentType string
}

// POSTFile uploads a file as multipart/form-data under the given form field.
func (c *Client) POSTFile(path, field string, file File) *Response {
	c.app.t.Helper()

	contentType := file.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	header := textproto.MIMEHeader{}
	header.Set("Content-Disposition", fmt.Sprintf(`form-data; name=%q; filename=%q`, field, file.Name))
	header.Set("Content-Type", contentType)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreatePart(header)
	if err == nil {
		_, err = part.Write(file.Content)
	}
	if err == nil {
		err = writer.Close()
	}
	if err != nil {
		c.app.t.Fatalf("testkit: build multipart body: %v", err)
	}
	return c.send(http.MethodPost, path, body, writer.FormDataContentType())
}

// client returns a redirect as the response instead of following it, so a test can assert on it.
var client = &http.Client{
	CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
}

func (c *Client) sendJSON(method, path string, body any) *Response {
	c.app.t.Helper()

	switch b := body.(type) {
	case nil:
		return c.send(method, path, nil, "")
	case string:
		return c.send(method, path, strings.NewReader(b), "application/json")
	case []byte:
		return c.send(method, path, bytes.NewReader(b), "application/json")
	case io.Reader:
		return c.send(method, path, b, "application/json")
	default:
		encoded, err := json.Marshal(body)
		if err != nil {
			c.app.t.Fatalf("testkit: encode request body: %v", err)
		}
		return c.send(method, path, bytes.NewReader(encoded), "application/json")
	}
}

func (c *Client) send(method, path string, body io.Reader, contentType string) *Response {
	c.app.t.Helper()

	req, err := http.NewRequest(method, c.app.baseURL()+path, body)
	if err != nil {
		c.app.t.Fatalf("testkit: %s %s: %v", method, path, err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	for key, value := range c.headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		c.app.t.Fatalf("testkit: %s %s: %v", method, path, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.app.t.Fatalf("testkit: %s %s: read response: %v", method, path, err)
	}

	return &Response{
		StatusCode: resp.StatusCode,
		Body:       respBody,
		Header:     resp.Header,
		t:          c.app.t,
		request:    method + " " + path,
	}
}
