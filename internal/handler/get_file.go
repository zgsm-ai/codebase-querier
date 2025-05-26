package handler

import (
	"github.com/zgsm-ai/codebase-indexer/pkg/response"
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"github.com/zgsm-ai/codebase-indexer/internal/logic"
	"github.com/zgsm-ai/codebase-indexer/internal/svc"
	"github.com/zgsm-ai/codebase-indexer/internal/types"
)

func getFileContentHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.FileContentRequest
		if err := httpx.Parse(r, &req); err != nil {
			response.Error(w, err)
			return
		}

		l := logic.NewGetFileContentLogic(r.Context(), svcCtx)
		err := l.GetFileContent(&req)
		if err != nil {
			response.Error(w, err)
		} else {
			response.Ok(w)
		}
	}
}
