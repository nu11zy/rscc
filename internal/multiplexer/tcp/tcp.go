package tcp

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"rscc/internal/common/logger"
	"rscc/internal/database"
	"rscc/internal/database/ent"
	"strings"
	"time"

	"github.com/go-faster/errors"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type TcpConfig struct {
	Listener net.Listener
	Timeout  time.Duration
}

type Tcp struct {
	lg       *zap.SugaredLogger
	db       *database.Database
	config   *TcpConfig
	listener net.Listener
}

// NewServer prepares environment for new TCP server
func NewServer(ctx context.Context, db *database.Database, config *TcpConfig) (*Tcp, error) {
	return &Tcp{
		lg:       logger.FromContext(ctx).Named("tcp"),
		db:       db,
		config:   config,
		listener: config.Listener,
	}, nil
}

func (t *Tcp) Start(ctx context.Context) error {
	t.lg.Info("Start TCP server")

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		for {
			// get connection
			conn, err := t.listener.Accept()
			if err != nil {
				if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
					return nil
				}
				return errors.Wrap(err, "failed accept TCP downlod connection")
			}
			go t.handle(conn)
		}
	})

	g.Go(func() error {
		<-ctx.Done()
		if err := t.Stop(); err != nil {
			t.lg.Warn("Stop server: %v", err)
		}
		t.lg.Info("Stop server")
		return nil
	})

	return g.Wait()
}

// Stop stops TCP server
func (t *Tcp) Stop() error {
	if t.listener != nil {
		return t.listener.Close()
	}
	return nil
}

// handle processes TCP connection
func (t *Tcp) handle(conn net.Conn) {
	defer conn.Close()

	lg := t.lg.Named(fmt.Sprintf("[%s]", conn.RemoteAddr().String()))

	// deadline to read first bytes from request
	conn.SetDeadline(time.Now().Add(3 * time.Second))

	// RSCC header (4 bytes) + 255 bytes
	b := make([]byte, 4+255)

	// read url as file identificator
	n, err := conn.Read(b)
	if err != nil {
		lg.Warnf("failed to read file's URL: %v", err)
		return
	}

	// check if read data not malformed
	if n < 4 {
		lg.Warnf("malformed TCP packet")
		return
	}

	// reset deadline
	conn.SetDeadline(time.Time{})

	// get url from TCP stream
	url := strings.TrimSpace(string(b[4:n]))

	// get agent by URL
	agent, err := t.db.GetAgentByUrl(context.Background(), url)
	if err != nil {
		if ent.IsNotFound(err) {
			lg.Warnf("Unknown agent by URL %s", url)
		} else {
			lg.Errorf("Get agent by URL from DB: %v", err)
		}
		return
	}

	// open FD to agent's file
	file, err := os.Open(agent.Path)
	if err != nil {
		lg.Errorf("unable open agent's binary by path %s: %v", agent.Path, err)
		return
	}
	defer file.Close()

	// update hits number
	if err = t.db.UpdateAgentHits(context.Background(), agent.ID); err != nil {
		lg.Errorf("unable update agent %s hits number for binary: %v", agent.ID, err)
		return
	}

	// return file output to client
	if _, err := io.Copy(conn, file); err != nil {
		lg.Errorf("transfer file: %s", err.Error())
		return
	}

	lg.Infof("downloaded %s", agent.URL)
}
