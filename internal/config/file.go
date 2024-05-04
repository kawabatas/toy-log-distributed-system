package config

import (
	"os"
	"path/filepath"
)

var (
	CAFile             = configFile("ca.pem")
	ServerCertFile     = configFile("server.pem")
	ServerKeyFile      = configFile("server-key.pem")
	RootClientCertFile = configFile("root-client.pem")
	RootClientKeyFile  = configFile("root-client-key.pem")
)

func configFile(filename string) string {
	dir := os.Getenv("CONFIG_DIR")
	if dir == "" {
		panic("set CONFIG_DIR env")
	}
	return filepath.Join(dir, filename)
}
