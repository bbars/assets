package main

import (
	"context"
	"database/sql"
	"embed"
	"github.com/bbars/assets/commands"
	"github.com/bbars/assets/service"
	"github.com/bbars/assets/service/repository"
	"github.com/bbars/assets/service/storage"
	"github.com/bbars/assets/utils"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
	"log"
	"os"
	"regexp"
	"strings"
)

//go:embed migrations/*
var migrations embed.FS

var (
	app *cli.App
)

func init() {
	app = &cli.App{
		Name:        os.Args[0],
		Usage:       "",
		Description: "Asset storage service.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "dir",
				Usage:    "Directory to store asset files. Example: './storage'.",
				Required: false,
				EnvVars:  []string{"ASSETS_DIR"},
			},
			&cli.UintFlag{
				Name:    "path-depth",
				Usage:   "Directory tree depth.",
				Value:   2,
				EnvVars: []string{"ASSETS_PATH_DEPTH"},
			},
			&cli.UintFlag{
				Name:    "dir-perm",
				Usage:   "Permission flags for new directories within a tree.",
				Value:   0755,
				EnvVars: []string{"ASSETS_DIR_PERM"},
			},
			&cli.UintFlag{
				Name:    "file-perm",
				Usage:   "Permission flags for new files within a tree.",
				Value:   0655,
				EnvVars: []string{"ASSETS_FILE_PERM"},
			},
			&cli.Uint64Flag{
				Name:    "max-remote-size",
				Usage:   "Size limit for resources fetched by URL.",
				Value:   1024 * 1024 * 1024, // 1GiB
				EnvVars: []string{"ASSETS_MAX_REMOTE_SIZE"},
			},
			&cli.Uint64Flag{
				Name:    "max-remote-wait-size",
				Usage:   "Size limit to wait for resources fetched by URL.",
				Value:   10 * 1024 * 1024, // 10MiB
				EnvVars: []string{"ASSETS_MAX_REMOTE_WAIT_SIZE"},
				Hidden:  true, // TODO add support
			},
			&cli.Uint64Flag{
				Name:    "max-size",
				Usage:   "Size limit for resources pushed directly.",
				Value:   0, // no limit
				EnvVars: []string{"ASSETS_MAX_SIZE"},
			},
			&cli.StringFlag{
				Name:     "original-url-pattern",
				Usage:    "RegExp pattern to check URLs before fetch. Example: '^https?://.'.",
				Required: false,
				EnvVars:  []string{"ASSETS_ORIGINAL_URL_PATTERN"},
			},
			&cli.StringFlag{
				Name:    "http-user-agent",
				Usage:   "User-Agent header string used by HTTP client when fetching remote resources.",
				Value:   "AssetsClient",
				EnvVars: []string{"ASSETS_HTTP_USER_AGENT"},
			},
			&cli.StringFlag{
				Name:     "dsn",
				Usage:    "Data source name (only sqlite3 is supported for now). Example: 'sqlite3:./storage/assets.db?mode=rwc&_journal=TRUNCATE'.",
				Required: true,
				EnvVars:  []string{"ASSETS_DSN"},
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
	assetsConf := service.AssetsConfig{
		MaxRemoteSize:      ctx.Int64("max-remote-size"),
		MaxRemoteWaitSize:  ctx.Int64("max-remote-wait-size"),
		MaxSize:            ctx.Int64("max-size"),
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
		Config:     assetsConf,
		HttpClient: nil,
	}

	return
}

func main() {
	ctx := context.Background()
	ctx = utils.ContextSetDebugAuto(ctx)
	ctx = utils.ContextHandleInterruptSignal(ctx)

	err := app.RunContext(ctx, os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
