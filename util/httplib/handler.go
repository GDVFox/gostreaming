package httplib

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/GDVFox/gostreaming/util"
)

type ctxKey string

const (
	// RequestLogger ключ контекста для логгера.
	RequestLogger ctxKey = "logger"
)

// Handler обработчик запроса.
// err в данном случае означает системную ошибку.
type Handler func(r *http.Request) (*Response, error)

// CreateHandler создает обертку, которая преобразует переданный обработчик
// в стандартный http.Handler
func CreateHandler(h Handler, l *util.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := uuid.New()
		logger := &util.Logger{SugaredLogger: l.With(zap.String("request_id", token.String()))}
		logger.Infof("got request %s, %s", r.Method, r.URL)

		ctx := context.WithValue(r.Context(), RequestLogger, logger)
		reply, err := h(r.WithContext(ctx))
		if err != nil {
			logger.Errorf("hander error: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if reply.IsJSON {
			w.Header().Set("Content-Type", "application/json")
		} else {
			w.Header().Set("Content-Type", "application/octet-stream")
		}

		w.WriteHeader(reply.StatusCode)
		if _, err := w.Write(reply.Body); err != nil {
			logger.Errorf("resp body write error: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		logger.Infof("request done with code: %d %s", reply.StatusCode, http.StatusText(reply.StatusCode))
	}
}
