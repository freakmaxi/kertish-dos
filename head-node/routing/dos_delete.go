package routing

import (
	"net/http"
	"os"
	"strings"

	"github.com/freakmaxi/kertish-dos/basics/errors"
	"go.uber.org/zap"
)

func (d *dosRouter) handleDelete(w http.ResponseWriter, r *http.Request) {
	requestedPaths, _, err := d.describeXPath(r.Header.Get("X-Path"))
	if err != nil || len(requestedPaths) > 1 {
		w.WriteHeader(422)
		return
	}

	killZombiesHeader := strings.ToLower(r.Header.Get("X-Kill-Zombies"))
	killZombies := len(killZombiesHeader) > 0 && (strings.Compare(killZombiesHeader, "1") == 0 || strings.Compare(killZombiesHeader, "true") == 0)

	if err := d.dos.Delete(requestedPaths[0], killZombies); err != nil {
		if err == os.ErrNotExist {
			w.WriteHeader(404)
			return
		} else if err == errors.ErrNoAvailableActionNode {
			w.WriteHeader(503)
			return
		} else if err == errors.ErrLock {
			w.WriteHeader(523)
			return
		} else if err == errors.ErrZombie {
			w.WriteHeader(524)
			return
		} else if err == errors.ErrZombieAlive {
			w.WriteHeader(525)
			return
		} else if err == errors.ErrRepair {
			w.WriteHeader(526)
			return
		} else {
			w.WriteHeader(500)
		}
		d.logger.Error("Delete request is failed", zap.String("path", requestedPaths[0]), zap.Error(err))
	}
}
