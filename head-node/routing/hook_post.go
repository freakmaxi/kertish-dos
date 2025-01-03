package routing

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/freakmaxi/kertish-dos/basics/hooks"
	"go.uber.org/zap"
)

func (h *hookRouter) handlePost(w http.ResponseWriter, r *http.Request) {
	requestedPaths, err := h.describeXPath(r.Header.Get("X-Path"))
	if err != nil {
		w.WriteHeader(422)
		return
	}

	hook := hooks.Hook{}
	if err := json.NewDecoder(r.Body).Decode(&hook); err != nil {
		w.WriteHeader(422)
		return
	}
	hook.Prepare()

	if err := h.hook.Add(requestedPaths, &hook); err != nil {
		if err == os.ErrExist {
			w.WriteHeader(409)
			return
		}
		w.WriteHeader(500)
		h.logger.Error(
			"Add hook request is failed",
			zap.String("paths", strings.Join(requestedPaths, ",")),
			zap.Error(err),
		)
		return
	}

	if err := json.NewEncoder(w).Encode([]string{*hook.Id}); err != nil {
		w.WriteHeader(500)
		h.logger.Error(
			"Response of add hook request is failed",
			zap.Error(err),
		)
		return
	}

	w.WriteHeader(202)
}
