package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/posflag"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
	_ "github.com/lib/pq"
	pflag "github.com/spf13/pflag"
)

const (
	// AppVersion is the current version of the application.
	AppVersion = "2.5.1"

	// AppName is the name of the application.
	AppName = "listmonk"
)

// App is the global application state container.
type App struct {
	DB      *sqlx.DB
	FS      stuffbin.FileSystem
	Log     *log.Logger
	Cfg     *koanf.Koanf
}

var (
	lo = log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lshortfile)
	ko = koanf.New(".")
)

func init() {
	// Register CLI flags.
	f := pflag.NewFlagSet("config", pflag.ContinueOnError)
	f.Usage = func() {
		fmt.Println(f.FlagUsages())
		os.Exit(0)
	}

	// Default config paths include both config.toml and a local override file,
	// making it easy to keep personal settings without modifying the main config.
	f.StringSlice("config", []string{"config.toml", "config.local.toml"},
		"path to one or more config files (will be merged in order)")
	f.Bool("install", false, "setup database schema and exit")
	f.Bool("upgrade", false, "upgrade database schema to the latest version and exit")
	f.Bool("yes", false, "assume 'yes' to prompts during --install/upgrade")
	f.Bool("version", false, "show current version and exit")
	f.Bool("new-config", false, "generate a new sample config.toml and exit")
	f.String("static-dir", "", "(optional) path to directory with static files")
	f.String("i18n-dir", "", "(optional) path to directory with i18n language files")

	if err := f.Parse(os.Args[1:]); err != nil {
		lo.Fatalf("error parsing flags: %v", err)
	}

	// Display version and exit.
	if ok, _ := f.GetBool("version"); ok {
		fmt.Println(AppVersion)
		os.Exit(0)
	}

	// Load config files.
	cfgFiles, _ := f.GetStringSlice("config")
	for _, c := range cfgFiles {
		if err := ko.Load(file.Provider(c), toml.Parser()); err != nil {
			if os.IsNotExist(err) {
				lo.Printf("config file not found, skipping: %s", c)
				continue
			}
			lo.Fatalf("error loading config file %s: %v", c, err)
		}
	}

	// Load environment variables (e.g. LISTMONK_app__address).
	if err := ko.Load(env.Provider("LISTMONK_", ".", func(s string) string {
		return strings.Replace(
			strings.ToLower(strings.TrimPrefix(s, "LISTMONK_")), "__", ".", -1)
	}), nil); err != nil {
		lo.Fatalf("error loading environment variables: %v", err)
	}

	// Override config with CLI flags.
	if err := ko.Load(posflag.Provider(f, ".", ko), nil); err != nil {
		lo.Fatalf("error loading CLI flags into config: %v", err)
	}
}

func main() {
	lo.Printf("starting %s v%s", AppName, AppVersion)

	// Initialize the database connection.
	db, err := initDB()
	if err != nil {
		lo.Fatalf("error initializing database: %v", err)
	}
	defer db.Close()

	// Initialize the app.
	app := &App{
		DB:  db,
		Log: lo,
		Cfg: ko,
	}

	// Start the HTTP server.
	if err := initHTTPServer(app); err != nil {
		lo.Fatalf("error starting HTTP server: %v", err)
	}
}
