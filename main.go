package main

import (
	"context"
	"database/sql"
	"embed"
	"github.com/bbars/assets/commands"
	"github.com/bbars/assets/ctxutil"
	"github.com/bbars/assets/service"
	"github.com/bbars/assets/service/repository"
	"github.com/bbars/assets/service/storage"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
	"log"
	"os"
	"regexp"
	"strings"
)

const AppEnvPrefix = "ASSETS_"

//go:embed migrations/*
var migrations embed.FS

var (
	app *cli.App
)

func init() {
	app = &cli.App{
		Name:  os.Args[0],
		Usage: "Asset storage service",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "dir",
				Usage:    "directory to store asset files (example: './storage')",
				Required: false,
				EnvVars:  []string{AppEnvPrefix + "DIR"},
			},
			&cli.UintFlag{
				Name:    "path-depth",
				Usage:   "max directory tree depth",
				Value:   2,
				EnvVars: []string{AppEnvPrefix + "MAX_DEPTH"},
			},
			&cli.UintFlag{
				Name:    "dir-perm",
				Usage:   "permission flags for new directories within a tree",
				Value:   0755,
				EnvVars: []string{AppEnvPrefix + "DIR_PERM"},
			},
			&cli.UintFlag{
				Name:    "file-perm",
				Usage:   "permission flags for the files within a tree",
				Value:   0655,
				EnvVars: []string{AppEnvPrefix + "FILE_PERM"},
			},
			&cli.Uint64Flag{
				Name:    "max-remote-size",
				Usage:   "size limit for resources fetched by url",
				Value:   1000 * 1024 * 1024, // 1000GiB
				EnvVars: []string{AppEnvPrefix + "MAX_REMOTE_SIZE"},
			},
			&cli.Uint64Flag{
				Name:    "max-remote-wait-size",
				Usage:   "size limit to wait for resources fetched by url",
				Value:   10 * 1024 * 1024, // 10MiB
				EnvVars: []string{AppEnvPrefix + "MAX_REMOTE_WAIT_SIZE"},
				Hidden:  true, // TODO add support
			},
			&cli.Uint64Flag{
				Name:    "max-size",
				Usage:   "size limit for resources pushed directly",
				Value:   2000 * 1024 * 1024, // 2000GiB
				EnvVars: []string{AppEnvPrefix + "MAX_SIZE"},
				Hidden:  true, // TODO add support
			},
			&cli.StringFlag{
				Name:     "original-url-pattern",
				Usage:    "regexp pattern to check urls before fetch (example: '^https?://.')",
				Required: false,
				EnvVars:  []string{AppEnvPrefix + "ORIGINAL_URL_PATTERN"},
			},
			&cli.StringFlag{
				Name:    "http-user-agent",
				Usage:   "user-agent header string used by http client when fetching remote resources",
				Value:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/112.0.0.0 Safari/537.36",
				EnvVars: []string{AppEnvPrefix + "HTTP_USER_AGENT"},
			},
			&cli.StringFlag{
				Name:     "dsn",
				Usage:    "assets data source name (only sqlite3 is supported for now; example: 'sqlite3:./._storage/assets.db?_journal=TRUNCATE')",
				Required: true,
				EnvVars:  []string{AppEnvPrefix + "DSN"},
			},
		},
		Commands: []*cli.Command{
			commands.NewMigrateCommand(initAssetRepo),
			commands.NewHttpCommand(initAssets),
			commands.NewStoreUrlsCommand(initAssets),
			commands.NewStoreFilesCommand(initAssets),
			commands.NewStorePipeCommand(initAssets),
		},
	}
	app.Setup()
}

func initAssetRepo(ctx *cli.Context) (assetRepo repository.Repository, err error) {
	drvDsn := strings.SplitN(ctx.String("dsn"), ":", 2)
	if len(drvDsn) == 1 {
		drvDsn = []string{
			"sqlite3",
			drvDsn[0],
		}
	} else if len(drvDsn) != 2 {
		err = errors.New("invalid value for dsn flag")
		return
	}

	db, err := sql.Open(drvDsn[0], drvDsn[1])
	if err != nil {
		err = errors.New("unable to connect to db")
		return
	}
	assetRepo = repository.NewSqlite(db, migrations)

	return
}

func initAssets(ctx *cli.Context) (assets *service.Assets, err error) {
	originalUrlPattern, err := regexp.Compile(ctx.String("original-url-pattern"))
	if err != nil {
		err = errors.Wrap(err, "invalid regexp passed for original-url-pattern flag")
		return
	}
	assetStorageConf := service.AssetStorageConfig{
		MaxRemoteSize:      ctx.Uint64("max-remote-size"),
		MaxRemoteWaitSize:  ctx.Uint64("max-remote-wait-size"),
		MaxSize:            ctx.Uint64("max-size"),
		OriginalUrlPattern: originalUrlPattern,
		HttpUserAgent:      ctx.String("http-user-agent"),
	}

	dirStorage := &storage.DirStorage{
		Dir:       ctx.String("dir"),
		PathDepth: uint8(ctx.Uint("path-depth")),
		DirPerm:   os.FileMode(ctx.Uint("dir-perm")),
		FilePerm:  os.FileMode(ctx.Uint("file-perm")),
	}

	repo, err := initAssetRepo(ctx)
	if err != nil {
		err = errors.Wrap(err, "unable to init asset repo")
		return
	}

	assets = &service.Assets{
		Storage:    dirStorage,
		Repo:       repo,
		Config:     assetStorageConf,
		HttpClient: nil,
	}

	return
}

func main() {
	ctx := context.Background()
	ctx = ctxutil.SetDebugAuto(ctx)
	ctx = ctxutil.HandleInterruptSignal(ctx)

	err := app.RunContext(ctx, os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
