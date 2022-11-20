package httplib

import (
	"encoding/json"
	"net/http"
)

// ContentType тип возвращаемого значения
type ContentType string

// Возможные типы ContentType
const (
	ContentTypeRaw  ContentType = "application/octet-stream"
	ContentTypeJSON ContentType = "application/json"
	ContentTypeHTML ContentType = "text/html"
)

// Response представляет ответ обработчика.
type Response struct {
	StatusCode  int
	Body        []byte
	ContentType ContentType
}

// WriteTo записывает response в w
func (r *Response) WriteTo(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", string(r.ContentType))
	w.WriteHeader(r.StatusCode)
	if _, err := w.Write(r.Body); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}
	return nil
}

// NewOKResponse возвращает 200 OK, если body ненулевое и 204 No Content, если body нулевое.
func NewOKResponse(body []byte, t ContentType) *Response {
	if body == nil {
		return &Response{StatusCode: http.StatusNoContent}
	}
	return &Response{
		StatusCode:  http.StatusOK,
		Body:        body,
		ContentType: t,
	}
}

// NewBadRequestResponse возвращает Response с кодом 400 Bad Request
func NewBadRequestResponse(body []byte) *Response {
	return &Response{
		StatusCode:  http.StatusBadRequest,
		Body:        body,
		ContentType: ContentTypeJSON,
	}
}

// NewNotFoundResponse возвращает Response с кодом 404 Not Found
func NewNotFoundResponse(body []byte) *Response {
	return &Response{
		StatusCode:  http.StatusNotFound,
		Body:        body,
		ContentType: ContentTypeJSON,
	}
}

// NewConflictResponse возвращает Response с кодом 409 Conflict
func NewConflictResponse(body []byte) *Response {
	return &Response{
		StatusCode:  http.StatusConflict,
		Body:        body,
		ContentType: ContentTypeJSON,
	}
}

// NewInternalErrorResponse возвращает Response с кодом 500 Internal Server Error
func NewInternalErrorResponse(body []byte) *Response {
	return &Response{
		StatusCode:  http.StatusInternalServerError,
		Body:        body,
		ContentType: ContentTypeJSON,
	}
}

// ErrorBody тело ответа, при возникновении ошибки.
type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// NewErrorBody создает новое тело ответа, содержащего ошибку
func NewErrorBody(c string, msg string) []byte {
	errBody := &ErrorBody{
		Code:    c,
		Message: msg,
	}
	data, err := json.Marshal(errBody)
	if err != nil {
		return nil
	}
	return data
}
