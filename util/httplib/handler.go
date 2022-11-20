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

// WSHandler обработчик на открытие вебсокета.
// err в данном случае означает системную ошибку.
type WSHandler func(w http.ResponseWriter, r *http.Request) error

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

		if err := reply.WriteTo(w); err != nil {
			logger.Errorf("resp body write error: %v", err)
		}
		logger.Infof("request done with code: %d %s", reply.StatusCode, http.StatusText(reply.StatusCode))
	}
}

// CreateWSHandler создает обертку, которая преобразует переданный обработчик
// в стандартный http.Handler
func CreateWSHandler(h WSHandler, l *util.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := uuid.New()
		logger := &util.Logger{SugaredLogger: l.With(zap.String("request_id", token.String()))}
		logger.Infof("got request %s, %s", r.Method, r.URL)

		ctx := context.WithValue(r.Context(), RequestLogger, logger)

		if err := h(w, r.WithContext(ctx)); err != nil {
			logger.Errorf("hander error: %v", err)
			return
		}

		logger.Infof("open ws request done")
	}
}
