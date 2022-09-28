package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/mrmeku/sqlboiler-bazel/psqltesting"
	"github.com/pelletier/go-toml"
)

var (
	outputDir      = flag.String("output_dir", "", "Path to output directory")
	schema         = flag.String("schema", "", "Name of the schema to generate files for")
	postgresBinDir = flag.String("postgres_bin_dir", "", "Path to a directory containing the standard suite of postgres binaries.")
	sqlFile        = flag.String("sql", "", "Path to write sqlboiler config file")
	sqlboiler      = flag.String("sqlboiler", "", "Path to sqlboiler executable")
	sqlboilerPSQL  = flag.String("sqlboiler_psql", "", "Path to sqlboiler-psql executable")
	createUsers    = flag.String("create_users", "", "Comma delimited list of users to create")
	config         = flag.String("config", "", "Path to additional toml config file")
)

func writeConfigFile(connection *psqltesting.Connection, configPath *string) *os.File {

	tree, err := toml.TreeFromMap(nil)
	if configPath != nil && *configPath != "" {
		tree, err = toml.LoadFile(*configPath)
		if err != nil {
			exitf("toml.LoadFile(%s): %v", *configPath, err)
		}
	}

	tree.Set("psql.user", connection.UserName)
	tree.Set("psql.dbname", connection.DbName)
	tree.Set("psql.sslmode", "disable")
	tree.Set("psql.host", connection.Host)
	tree.Set("psql.port", fmt.Sprintf("%d", connection.Port))
	tree.Set("psql.schema", *schema)
	f, err := ioutil.TempFile("", "*.config.toml")
	if err != nil {
		exitf("ioutil.TempFile() err: %v", err)
	}
	if _, err := tree.WriteTo(f); err != nil {
		exitf("tree.WriteTo(): %v", err)
	}

	return f
}

func readSQL() (string, error) {
	sqlContent, err := ioutil.ReadFile(*sqlFile)
	if err != nil {
		return "", fmt.Errorf("failed to read %s: %v", *sqlFile, err)
	}

	return string(sqlContent), nil
}

func sqlboilerCmd(args ...string) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command(*sqlboiler, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		exitf("cmd.Run(%q, %v) error: %v\nstdout: %s\nstderr: %s", *sqlboiler, args, err, stdout.String(), stderr.String())
	}
}

func main() {
	flag.Parse()

	if *outputDir == "" || *schema == "" || *sqlFile == "" || *sqlboiler == "" || *postgresBinDir == "" {
		flag.Usage()
		exitf(`Need to pass output_dir, schema, sql and sqlboiler as flags, received
  --output_dir=%q
  --postgres_bin_dir=%q
  --schema=%q
  --sql=%q
  --sqlboiler=%q
`,
			*outputDir,
			*postgresBinDir,
			*schema,
			*sqlFile,
			*sqlboiler,
		)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, connection, err := psqltesting.New(ctx,
		psqltesting.WithTCP(),
		psqltesting.WithPostgresqlBinDir(*postgresBinDir))
	if err != nil {
		exitf(fmt.Sprintf("New() error: %v", err))
	}
	if err = db.Ping(); err != nil {
		exitf(fmt.Sprintf("db.Ping() error: %v", err))
	}

	for _, user := range strings.Split(*createUsers, ",") {
		query := fmt.Sprintf("CREATE ROLE %q;", user)
		if _, err := db.Exec(query); err != nil {
			exitf("%s ERROR: %v", query, err)
		}
	}

	schemaContent, err := readSQL()
	if err != nil {
		exitf("Error reading sql: %v", err)
	}
	if _, err := db.ExecContext(ctx, schemaContent); err != nil {
		exitf("unable to migrate database: %v", err)
	}

	configFile := writeConfigFile(connection, config)
	defer os.Remove(configFile.Name())

	args := []string{*sqlboilerPSQL, "--output", *outputDir, "--config", configFile.Name(), "--no-tests", "--no-hooks"}

	sqlboilerCmd(args...)
}

func exitf(format string, args ...interface{}) {
	fmt.Printf(format, args...)
	os.Exit(1)
}
