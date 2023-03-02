package types

import (
	"crypto/md5"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

var MD5AuthSalt [4]byte = [4]byte{'1', '2', '3', '4'}

type YAMLFileAuthProviderConfigUser struct {
	Superuser bool   `yaml:"superuser"`
	Password  string `yaml:"password"`
	Tables    []struct {
		Name        string   `yaml:"name"`
		Schema      string   `yaml:"schema"`
		AccessModes []string `yaml:"access_modes"`
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

func md5s(s string) string {
	h := md5.New()
	h.Write([]byte(s))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func encodeMD5Password(username, password, salt string) string {
	hashedCreds := md5s(username + password)
	return "md5" + md5s(hashedCreds+salt)
}

func (p *YAMLFileAuthProvider) CheckAuth(username, password string) bool {
	md5Pass := encodeMD5Password(p.config[username].Password, username, string(MD5AuthSalt[:]))
	return md5Pass == password
}

func (p *YAMLFileAuthProvider) CheckAccess(accessInfo TableAccessInfo, username string) bool {
	if p.config[username].Superuser {
		return true
	}
	for _, table := range p.config[username].Tables {
		if table.Name == accessInfo.Name && table.Schema == accessInfo.Schema {
			for _, mode := range table.AccessModes {
				if mode == accessInfo.AccessMode.ToString() {
					return true
				}
			}
		}
	}
	return false
}
