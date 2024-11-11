package indexer

import (
	"github.com/CosmWasm/wasmd/indexer/server/config"
	cmtcfg "github.com/cometbft/cometbft/config"
	"github.com/spf13/viper"
)

func ReadServiceConfig(configPath string, configFileName string, v *viper.Viper) (*config.Config, error) {
	if err := readConfig(configPath, configFileName, v); err != nil {
		return nil, err
	}

	conf := config.DefaultConfig()
	if err := v.Unmarshal(conf); err != nil {
		return nil, err
	}

	return conf, nil
}

func ReadCometBFTConfig(configPath string, configFileName string, v *viper.Viper) (*cmtcfg.Config, error) {
	if err := readConfig(configPath, configFileName, v); err != nil {
		return nil, err
	}

	conf := cmtcfg.DefaultConfig()
	if err := v.Unmarshal(conf); err != nil {
		return nil, err
	}

	return conf, nil
}

func readConfig(configPath string, configFileName string, v *viper.Viper) error {
	v.AddConfigPath(configPath)
	v.SetConfigName(configFileName)
	v.SetConfigType("toml")

	if err := v.ReadInConfig(); err != nil {
		return err
	}
	return nil
}
