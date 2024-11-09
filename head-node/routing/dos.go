package routing

import (
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/freakmaxi/kertish-dos/basics/common"
	"github.com/freakmaxi/kertish-dos/head-node/manager"
	"go.uber.org/zap"
)

type dosRouter struct {
	dos    manager.Dfs
	logger *zap.Logger

	definitions []*Definition
}

func NewDfsRouter(dos manager.Dfs, logger *zap.Logger) Router {
	pR := &dosRouter{
		dos:         dos,
		logger:      logger,
		definitions: make([]*Definition, 0),
	}
	pR.setup()

	return pR
}

func (d *dosRouter) setup() {
	d.definitions =
		append(d.definitions,
			&Definition{
				Path:    "/client/dos",
				Handler: d.manipulate,
			},
		)
}

func (d *dosRouter) Get() []*Definition {
	return d.definitions
}

func (d *dosRouter) manipulate(w http.ResponseWriter, r *http.Request) {
	defer func() { _ = r.Body.Close() }()

	switch r.Method {
	case http.MethodGet:
		d.handleGet(w, r)
	case http.MethodPost:
		d.handlePost(w, r)
	case http.MethodPut:
		d.handlePut(w, r)
	case http.MethodDelete:
		d.handleDelete(w, r)
	default:
		w.WriteHeader(406)
	}
}

func (d *dosRouter) validateApplyTo(applyTo string) bool {
	switch applyTo {
	case "folder", "file":
		return true
	}
	return false
}

func (d *dosRouter) describeXPath(xPath string) ([]string, string, error) {
	action := ""
	commaIdx := strings.Index(xPath, ",")
	if commaIdx == 1 {
		action = xPath[:1]
		xPath = xPath[2:]
	}

	switch action {
	case "", "j":
	default:
		return nil, "", os.ErrInvalid
	}

	paths := strings.Split(xPath, ",")
	for i := range paths {
		p, err := url.QueryUnescape(paths[i])
		if err != nil {
			return nil, "", err
		}
		if !common.ValidatePath(p) {
			return nil, "", os.ErrInvalid
		}
		paths[i] = p
	}

	if len(paths) == 0 {
		return nil, "", os.ErrInvalid
	}

	return paths, action, nil
}

var _ Router = &dosRouter{}
