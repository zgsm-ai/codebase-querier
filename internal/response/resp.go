package response

import (
	"context"
	"github.com/zeromicro/go-zero/rest/httpx"
	"net/http"
)

const (
	CodeOK = 0

	MessageOk = "ok"

	CodeError = -1
)

type Response[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data,omitempty"`
}

func Ok(w http.ResponseWriter) {
	httpx.Ok(w)
}

func Error(w http.ResponseWriter, e error) {
	httpx.OkJson(w, wrapResponse(e))
}

func RawText(w http.ResponseWriter, v any) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(v.([]byte))
	w.Header().Set("Content-Type", "plain/text")
}

func JsonCtx(ctx context.Context, w http.ResponseWriter, v any) {
	httpx.OkJsonCtx(ctx, w, wrapResponse(v))
}

func Json(w http.ResponseWriter, v any) {
	httpx.OkJson(w, wrapResponse(v))
}

func wrapResponse(v any) Response[any] {
	var resp Response[any]
	switch data := v.(type) {
	case *codeMsg:
		resp.Code = data.Code
		resp.Message = data.Message
	case codeMsg:
		resp.Code = data.Code
		resp.Message = data.Message
	case error:
		resp.Code = CodeError
		resp.Message = data.Error()
	default:
		resp.Code = CodeOK
		resp.Message = MessageOk
		resp.Data = v
	}

	return resp
}
