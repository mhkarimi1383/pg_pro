package types

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

var salt [4]byte = [4]byte{}

type YAMLFileAuthProviderConfigUser struct {
	Superuser bool   `yaml:"superuser"`
	Password  string `yaml:"password"`
	Tables    []struct {
		Name       string `yaml:"name"`
		Schema     string `yaml:"schema"`
		AccessMode string `yaml:"access_mode"`
	} `yaml:"tables"`
}

type YAMLFileAuthProviderConfig map[string]YAMLFileAuthProviderConfigUser

type YAMLFileAuthProvider struct {
	config YAMLFileAuthProviderConfig
}

func (p *YAMLFileAuthProvider) SetConfig(configFilePath string) error {
	data, err := os.ReadFile(configFilePath)
	if err != nil {
		return err
	}
	cfg := new(YAMLFileAuthProviderConfig)
	err = yaml.Unmarshal(data, cfg)
	if err != nil {
		return err
	}
	p.config = *cfg
	return nil
}

func hexMD5(s string) string {
	hash := md5.New()
	io.WriteString(hash, s)
	return hex.EncodeToString(hash.Sum(nil))
}

func (p *YAMLFileAuthProvider) CheckAccess(accessInfo TableAccessInfo, username string) bool {
	fmt.Printf("p.config[username]: %+v\n", p.config[username])
	println(p.config[username].Password + username)
	log.Println("md5" + hexMD5(hexMD5(p.config[username].Password+username)) + string(salt[:]))
	if p.config[username].Superuser {
		return true
	}
	return false
}
