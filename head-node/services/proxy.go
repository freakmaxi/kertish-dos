package services

import (
	"fmt"
	"net/http"

	"github.com/freakmaxi/kertish-dos/head-node/routing"
	"go.uber.org/zap"
)

type Proxy struct {
	bindAddr string
	manager  *routing.Manager
	logger   *zap.Logger
}

func NewProxy(bindAddr string, manager *routing.Manager, logger *zap.Logger) *Proxy {
	return &Proxy{
		bindAddr: bindAddr,
		manager:  manager,
		logger:   logger,
	}
}

func (p *Proxy) Start() {
	p.logger.Info(fmt.Sprintf("Head Service is running on %s", p.bindAddr))
	if err := http.ListenAndServe(p.bindAddr, p.manager.Get()); err != nil {
		p.logger.Error("Head service is failed", zap.Error(err))
	}
}
