package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
)

// Error is the api's error answer, {"errors": {"<field>": [...], "_": [...]},
// "data": ...}: messages per form field, and general ones under "_".
type Error struct {
	Status int
	Fields map[string][]string
	Data   json.RawMessage
}

func parseError(status int, body []byte) *Error {
	e := &Error{Status: status}

	var shaped struct {
		Errors map[string][]string `json:"errors"`
		Data   json.RawMessage     `json:"data"`
	}
	if json.Unmarshal(body, &shaped) == nil && len(shaped.Errors) > 0 {
		e.Fields = shaped.Errors
		e.Data = shaped.Data
		return e
	}

	// Not the api's shape: a rate limit or timeout page, a bare status, or plain text like the
	// "Not Found" of a resource that isn't the account's
	message := strings.TrimSpace(string(body))
	if message == "" || strings.HasPrefix(message, "<") {
		message = fmt.Sprintf("%d %s", status, http.StatusText(status))
	}
	e.Fields = map[string][]string{"_": {message}}
	return e
}

func (e *Error) Error() string {
	var lines []string
	lines = append(lines, e.Fields["_"]...)

	fields := make([]string, 0, len(e.Fields))
	for field := range e.Fields {
		if field != "_" {
			fields = append(fields, field)
		}
	}
	slices.Sort(fields)
	for _, field := range fields {
		for _, message := range e.Fields[field] {
			lines = append(lines, field+": "+message)
		}
	}

	if len(lines) == 0 {
		return fmt.Sprintf("%d %s", e.Status, http.StatusText(e.Status))
	}
	return strings.Join(lines, "\n")
}

func StatusOf(err error) int {
	var e *Error
	if errors.As(err, &e) {
		return e.Status
	}
	return 0
}

func IsNotFound(err error) bool { return StatusOf(err) == http.StatusNotFound }

// MayHaveGoneThrough is a write that timed out before the api answered, which
// the api may still have carried out.
func MayHaveGoneThrough(err error) bool {
	status := StatusOf(err)
	return status == http.StatusBadGateway || status == http.StatusGatewayTimeout
}
