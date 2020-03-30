package langserver

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/channel"
	"github.com/creachadair/jrpc2/server"
	"github.com/hashicorp/terraform-ls/langserver/svcctl"
)

type langServer struct {
	srvCtx     context.Context
	logger     *log.Logger
	srvOptions *jrpc2.ServerOptions
	sf         svcctl.ServiceFactory
}

func NewLangServer(srvCtx context.Context, sf svcctl.ServiceFactory) *langServer {
	opts := &jrpc2.ServerOptions{
		AllowPush: true,
	}

	return &langServer{
		srvCtx:     srvCtx,
		logger:     log.New(ioutil.Discard, "", 0),
		srvOptions: opts,
		sf:         sf,
	}
}

func (ls *langServer) SetLogger(logger *log.Logger) {
	ls.srvOptions.Logger = logger
	ls.srvOptions.RPCLog = &rpcLogger{logger}
	ls.logger = logger
}

func (ls *langServer) newService() server.Service {
	svc := ls.sf(ls.srvCtx)
	svc.SetLogger(ls.logger)
	return svc
}

func (ls *langServer) newServer() *server.Simple {
	return server.NewSimple(ls.newService(), ls.srvOptions)
}

func (ls *langServer) StartAndWait(reader io.Reader, writer io.WriteCloser) error {
	srv := ls.newServer()

	ls.logger.Printf("Starting server (pid %d) ...", os.Getpid())

	// Wrap waiter with a context so that we can cancel it here
	// after the service is cancelled (and srv.Wait returns)
	ctx, cancelFunc := context.WithCancel(ls.srvCtx)
	go func() {
		ch := channel.LSP(reader, writer)
		err := srv.Run(ch)
		if err != nil {
			ls.logger.Printf("Error: %s", err)
		}
		cancelFunc()
	}()

	select {
	case <-ctx.Done():
		ls.logger.Printf("Stopping server (pid %d) ...", os.Getpid())
		srv.Stop()
	}

	ls.logger.Printf("Server (pid %d) stopped.", os.Getpid())
	return nil
}

func (ls *langServer) StartTCP(address string) error {
	ls.logger.Printf("Starting TCP server (pid %d) at %q ...",
		os.Getpid(), address)
	lst, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("TCP Server failed to start: %s", err)
	}
	ls.logger.Printf("TCP server running at %q", lst.Addr())

	go func() {
		ls.logger.Println("Starting loop server ...")
		err = server.Loop(lst, ls.newService, &server.LoopOptions{
			Framing:       channel.LSP,
			ServerOptions: ls.srvOptions,
		})
		if err != nil {
			ls.logger.Printf("Loop server failed to start: %s", err)
		}
	}()

	select {
	case <-ls.srvCtx.Done():
		ls.logger.Printf("Stopping TCP server (pid %d) ...", os.Getpid())
		err = lst.Close()
		if err != nil {
			ls.logger.Printf("TCP server (pid %d) failed to stop: %s", os.Getpid(), err)
			return err
		}
	}

	ls.logger.Printf("TCP server (pid %d) stopped.", os.Getpid())
	return nil
}
