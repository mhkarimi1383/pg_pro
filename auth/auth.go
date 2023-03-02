package auth

import (
	"github.com/mhkarimi1383/pg_pro/config"
	"github.com/mhkarimi1383/pg_pro/types"
)

var (
	provider types.AuthProvider
)

func init() {
	providerName := config.GetString("auth.provider")
	switch providerName {
	case "yaml":
		provider = new(types.YAMLFileAuthProvider)
		err := provider.(*types.YAMLFileAuthProvider).SetConfig(config.GetString("auth.path"))
		if err != nil {
			panic(err)
		}
	}
}

func GetProvider() types.AuthProvider {
	return provider
}
