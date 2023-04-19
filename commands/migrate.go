package commands

import (
	"github.com/bbars/assets/service/repository"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

func NewMigrateCommand(initAssetRepo InitAssetRepo) *cli.Command {
	m := migrate{
		assetRepo: nil,
	}
	return &cli.Command{
		Name:   "migrate",
		Usage:  "Apply migrations on current database",
		Action: m.Action,
		Before: func(ctx *cli.Context) (err error) {
			m.assetRepo, err = initAssetRepo(ctx)
			return
		},
	}
}

type migrate struct {
	assetRepo repository.Repository
	cliCtx    *cli.Context
}

func (m *migrate) Action(_ *cli.Context) (err error) {
	err = m.assetRepo.Migrate()
	if err != nil {
		err = errors.Wrap(err, "unable to migrate db")
		return
	}
	return
}
