package test

import (
	"github.com/linkly-id/auth/internal/conf"
	"github.com/linkly-id/auth/internal/storage"
)

func SetupDBConnection(globalConfig *conf.GlobalConfiguration) (*storage.Connection, error) {
	return storage.Dial(globalConfig)
}
