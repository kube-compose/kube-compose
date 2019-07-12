package service

import (
	"encoding/base64"
	"encoding/json"

	dockerTypes "github.com/docker/docker/api/types"
)

func EncodeRegistryAuth(username, password string) string {
	authConfig := dockerTypes.AuthConfig{
		Username: username,
		Password: password,
	}
	authConfigBytes, _ := json.Marshal(&authConfig)
	return base64.StdEncoding.EncodeToString(authConfigBytes)
}
