package manager

import (
	"context"
	"github.com/datadaodevs/go-service-framework/util"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"
)

type backgroundSvc struct {
	starter backgroundSvcStarter
	stopper backgroundSvcStopper
	ctx     context.Context
	cancel  context.CancelFunc
}
type httpSrv struct {
	server *http.Server
	cancel context.CancelFunc
}
type grpcSrv struct {
	server *grpc.Server
	port   string
	cancel context.CancelFunc
}
type backgroundSvcStarter func(ctx context.Context) error
type backgroundSvcStopper func()
type Manager struct {
	app                 string
	env                 Environment
	useGracefulShutdown bool
	svcContext          context.Context
	svcContextCancel    context.CancelFunc
	shutdownFunc        context.CancelFunc
	wg                  sync.WaitGroup
	backgroundSvc       map[string]*backgroundSvc
	httpSrvs            map[string]*httpSrv
	grpcSrvs            map[string]*grpcSrv
	logger              util.Logger
	metrics             util.Metrics
}

func New(opts ...opt) *Manager {
	parent := context.Background()
	ctx, cancel := context.WithCancel(parent)

	cfg := mustParseConfig()
	metrics := mustInitMetrics(cfg)
	logger := mustInitLogger(cfg)

	m := Manager{
		app:                 cfg.AppName,
		env:                 cfg.Env,
		useGracefulShutdown: true,
		svcContext:          ctx,
		svcContextCancel:    cancel,
		backgroundSvc:       map[string]*backgroundSvc{},
		httpSrvs:            map[string]*httpSrv{},
		grpcSrvs:            map[string]*grpcSrv{},
		logger:              logger,
		metrics:             metrics,
	}

	for _, opt := range opts {
		opt(&m)
	}

	return &m
}

func (m *Manager) RegisterBackgroundSvc(name string, starter backgroundSvcStarter, stopper backgroundSvcStopper) {
	m.backgroundSvc[name] = &backgroundSvc{
		starter: starter,
		stopper: stopper,
	}
}
func (m *Manager) RegisterHttpServer(name string, srv *http.Server) {
	m.httpSrvs[name] = &httpSrv{
		server: srv,
	}
}

func (m *Manager) RegisterGRPCServer(name string, port string, opts ...grpc.ServerOption) *grpc.Server {
	opts = append(opts, grpc.UnaryInterceptor(m.metricsInterceptor))
	baseServer := grpc.NewServer(opts...)
	m.grpcSrvs[name] = &grpcSrv{
		port:   port,
		server: baseServer,
	}
	return baseServer
}

func (m *Manager) Context() context.Context {
	return m.svcContext
}

func (m *Manager) Metrics() util.Metrics {
	return m.metrics
}

func (m *Manager) Logger() util.Logger {
	return m.logger
}

func (m *Manager) Env() Environment {
	return m.env
}

func (m *Manager) App() string {
	return m.app
}

func (m *Manager) ForceKill() {
	m.shutdownFunc()
}

func (m *Manager) WaitForInterrupt() {
	m.startBackgroundServices()
	m.startHTTPServers()
	m.startGRPCServers()

	aliveCtx, cancel := context.WithCancel(m.svcContext)
	m.shutdownFunc = cancel

	//	Attach to OS
	notifyChannel := make(chan os.Signal, 1)
	defer close(notifyChannel)
	signal.Notify(notifyChannel, os.Interrupt)
	signal.Notify(notifyChannel, os.Kill)

	log.Info("Waiting for interrupt")
	select {
	case <-aliveCtx.Done():
		log.Warn("Manual force kill signal received")
	case sig := <-notifyChannel:
		log.Warnf("OS signal received: %s", sig.String())
	}

	if !m.useGracefulShutdown {
		log.Warn("Graceful shutdown disabled; force-killing all services")
		m.svcContextCancel()
		log.Info("Manager exiting")
		return
	}

	timer := time.NewTimer(20 * time.Second)
	select {
	case <-m.attemptGracefulShutdown():
		log.Info("Graceful shutdown succeeded")
	case <-timer.C:
		log.Info("Graceful shutdown deadline exceeded; force-killing all services")
		m.svcContextCancel()
	}
	log.Info("Manager exiting")
}

func (m *Manager) attemptGracefulShutdown() chan struct{} {
	ch := make(chan struct{})
	go func() {
		for name, server := range m.httpSrvs {
			log.WithField("name", name).Info("Attempting graceful shutdown of HTTP server")
			server.cancel()
		}
	}()
	go func() {
		for name, server := range m.grpcSrvs {
			log.WithField("name", name).Info("Attempting graceful shutdown of GRPC server")
			server.cancel()
		}
	}()
	go func() {
		for name, svc := range m.backgroundSvc {
			log.WithField("name", name).Info("Attempting graceful shutdown of background service")
			svc.cancel()
		}
	}()
	go func() {
		m.wg.Wait()
		ch <- struct{}{}
	}()
	return ch
}

func (m *Manager) startGRPCServers() {
	for name, server := range m.grpcSrvs {
		//	Create "alive" context that can be used to trigger graceful shutdown
		aliveCtx, cancel := context.WithCancel(m.svcContext)
		m.grpcSrvs[name].cancel = cancel

		m.wg.Add(1)
		go func(aliveCtx context.Context, server *grpcSrv) {
			defer m.wg.Done()
			grpcListener, err := net.Listen("tcp", server.port)
			if err != nil {
				log.WithError(err).Fatal("Failed to start grpc server")
			}
			defer grpcListener.Close()

			log.WithField("name", name).Info("Starting GRPC server")
			if err := server.server.Serve(grpcListener); err != nil {
				log.WithField("name", name).WithError(err).Info("GRPC server stopped")
			}
		}(aliveCtx, server)
		m.wg.Add(1)
		go func(server *grpc.Server, name string) {
			defer m.wg.Done()
			<-aliveCtx.Done()
			log.WithField("name", name).Info("Shutting down grpc server")
			server.GracefulStop()
		}(server.server, name)
	}
}

func (m *Manager) startHTTPServers() {
	for name, server := range m.httpSrvs {
		//	Create "alive" context that can be used to trigger graceful shutdown
		aliveCtx, cancel := context.WithCancel(m.svcContext)
		m.httpSrvs[name].cancel = cancel

		m.wg.Add(1)
		go func(server *http.Server, name string) {
			defer m.wg.Done()
			log.WithField("name", name).Info("Starting HTTP server")
			if err := server.ListenAndServe(); err != nil {
				log.WithError(err).Info("HTTP exited")
			}
		}(server.server, name)
		m.wg.Add(1)
		go func(aliveCtx context.Context, server *http.Server) {
			defer m.wg.Done()
			<-aliveCtx.Done()
			log.WithField("name", name).Info("Shutting down HTTP server")
			server.Shutdown(m.svcContext)
		}(aliveCtx, server.server)
	}
}

func (m *Manager) startBackgroundServices() {
	for name, svc := range m.backgroundSvc {
		//	Create "alive" context that can be used to trigger graceful shutdown
		aliveCtx, cancel := context.WithCancel(m.svcContext)
		m.backgroundSvc[name].ctx = aliveCtx
		m.backgroundSvc[name].cancel = cancel

		m.wg.Add(1)
		go func(aliveCtx context.Context, svc *backgroundSvc, name string) {
			defer m.wg.Done()
			//	Create operating context to be passed to service
			opCtx, _ := context.WithCancel(m.svcContext)
			log.WithField("name", name).Info("Starting background service")
			if err := svc.starter(opCtx); err != nil {
				log.WithField("name", name).WithError(err).Warn("Failed to start background service")
				return
			}
			log.WithField("name", name).Info("Background service now running")
			<-aliveCtx.Done()
			log.WithField("name", name).Info("Service context cancelled; beginning graceful shutdown")
			svc.stopper()
			log.WithField("name", name).Info("Graceful shutdown complete")
		}(aliveCtx, svc, name)
	}
}
