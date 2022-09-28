// Package psqltesting provides a testing utility for creating a connection
// to a testing postres server.
package psqltesting

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/bazelbuild/rules_go/go/tools/bazel"
	_ "github.com/jackc/pgx/v4/stdlib" // Import pgx so that the caller doesn't need to remember to install it.
)

const (
	timeout     = 20 * time.Second
	runfilePath = "../psql"
)

var (
	driverRegister    sync.Once
	driverName        string
	driverRegisterErr error
)

func pgCmd(postgresBinDir, binaryName string, args ...string) error {
	bin := filepath.Join(postgresBinDir, binaryName)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command(bin, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cmd.Run(%q, %#v) error: %v\nstdout: %s\nstderr: %s", binaryName, args, err, stdout.String(), stderr.String())
	}

	return nil
}

// Add the ability to start the server with hardcoded binary.
// Control if listening over unix socket or TCP socket.
// Return both an active connection and connection parameters in a useable way.

type config struct {
	withTCP          bool
	postgresqlBinDir string
}

// Option describes functions which update the postgresql config.
type Option func(c config) config

// WithTCP tells the postgres server to listen on a TCP port.
func WithTCP() Option {
	return func(c config) config {
		c.withTCP = true
		return c
	}
}

// WithPostgresqlBinDir explicitly sets the directory where postgres binaries are located.
func WithPostgresqlBinDir(dir string) Option {
	return func(c config) config {
		c.postgresqlBinDir = dir
		return c
	}
}

// Connection describes how to connect to the psql database.
type Connection struct {
	UserName string
	Password string
	DbName   string
	Port     int
	Host     string
}

// String generates a postgresql connection string describing this connection.
func (c *Connection) String() string {
	connStr := fmt.Sprintf("user=%s dbname=%s sslmode=disable host=%s", c.UserName, c.DbName, c.Host)

	if c.Port != 0 {
		connStr = fmt.Sprintf("%s port=%d", connStr, c.Port)
	}

	if c.Password != "" {
		connStr = fmt.Sprintf("%s password=%s", connStr, c.Password)
	}
	return connStr
}

// New creates a testing postgres database and return a Unix Socket connection to it.
// Postgres is torn down when the context is done.
func New(ctx context.Context, options ...Option) (_ *sql.DB, _ *Connection, err error) {

	c := config{}
	for _, o := range options {
		if o == nil {
			return nil, nil, fmt.Errorf("received nil option when starting postgres")
		}
		c = o(c)
	}

	if c.postgresqlBinDir == "" {
		var err error
		c.postgresqlBinDir, err = bazel.Runfile(filepath.Join(runfilePath, "psql_compile", "bin"))
		if err != nil {
			return nil, nil, fmt.Errorf("unable to find postgres binary dir using Bazel runfiles. You might consider using psqltesting.WithPostgresDir to set it, or maybe you're missing a data depedency on the actual files. %v", err)
		}
	}

	os.Setenv("LD_LIBRARY_PATH", filepath.Join(c.postgresqlBinDir, "..", "lib"))

	postgresDir, err := ioutil.TempDir("/tmp", "postgres")
	if err != nil {
		return nil, nil, fmt.Errorf("ioutil.TempDir(postgres) error: %v", err)
	}
	defer func() {
		if err != nil {
			os.RemoveAll(postgresDir)
		}
	}()

	conn := &Connection{
		UserName: "bazel",
		DbName:   "postgres",
	}

	if err := pgCmd(c.postgresqlBinDir, "initdb",
		"--pgdata", postgresDir,
		"--username", conn.UserName); err != nil {
		return nil, nil, err
	}

	postgresArgs := []string{"-D", postgresDir, "-c", "unix_socket_directories=" + postgresDir}

	var pgIsReadyArgs []string
	if c.withTCP {
		l, listenErr := net.Listen("tcp", ":0")
		if listenErr != nil {
			return nil, nil, listenErr
		}
		listenErr = l.Close()
		if listenErr != nil {
			return nil, nil, listenErr
		}

		conn.Host = "localhost"
		conn.Port = l.Addr().(*net.TCPAddr).Port

		postgresArgs = append(
			postgresArgs,
			"-c", fmt.Sprintf("listen_addresses=%s", conn.Host),
			"-p", fmt.Sprintf("%d", conn.Port))

		pgIsReadyArgs = append(pgIsReadyArgs, "-p", fmt.Sprintf("%d", conn.Port))
	} else {
		conn.Host = postgresDir
		postgresArgs = append(postgresArgs, "-c", "listen_addresses=")
	}
	pgIsReadyArgs = append(pgIsReadyArgs, "-h", conn.Host)

	cmd := exec.CommandContext(ctx, filepath.Join(c.postgresqlBinDir, "postgres"), postgresArgs...)
	var (
		stdout buffer
		stderr buffer
	)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err = cmd.Start(); err != nil {
		return nil, nil, fmt.Errorf("cmd.Run() error: %v\nStdout: %v\nStderr: %v", err, stdout.String(), stderr.String())
	}

	go func(ctx context.Context) {
		select {
		case <-ctx.Done():
			// Kill postgres' entire process group.
			syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)

			// Kill postgres directly for good measure. It can't hurt and it is possible the syscall above failed.
			cmd.Process.Signal(syscall.SIGTERM)

			// Release resources.
			cmd.Wait()

			// Clean up files.
			os.RemoveAll(postgresDir)
		}
	}(ctx)

	db, err := sql.Open("pgx", conn.String())
	if err != nil {
		return nil, nil, fmt.Errorf("sql.Open(pgx, %q) error: %v\nStdout: %v\n%v", conn.String(), err, stdout.String(), stderr.String())
	}

	healthCheckCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
pg_isready:
	for {
		select {
		case <-healthCheckCtx.Done():
			return nil, nil, fmt.Errorf("tried to pg_isready but did not succeed after %v\nStdout: %v\n%v", timeout, stdout.String(), stderr.String())
		default:
			pgIsReadyArgs = append(pgIsReadyArgs,
				"--dbname", conn.DbName,
				"--username", conn.UserName)
			if err = pgCmd(c.postgresqlBinDir, "pg_isready", pgIsReadyArgs...); err == nil {
				break pg_isready
			} else {
				time.Sleep(100 * time.Millisecond)
			}
		}
	}

db_ping:
	for {
		select {
		case <-healthCheckCtx.Done():
			return nil, nil, fmt.Errorf("tried to db.Ping() but did not succeed after %v\nStdout: %v\n%v", timeout, stdout.String(), stderr.String())
		default:
			if err = db.Ping(); err == nil {
				break db_ping
			} else {
				time.Sleep(100 * time.Millisecond)
			}
		}
	}

	// Emulate Cloud SQL environment.
	if _, err := db.ExecContext(ctx, `CREATE USER "cloudsqlsuperuser";`); err != nil {
		return nil, nil, fmt.Errorf("creating user cloudsqlsuperuser: %w", err)
	}

	return db, conn, err
}
