package config

import "github.com/zalando/go-keyring"

const serviceName = "woffux"

func SetPassword(email, password string) error {
	return keyring.Set(serviceName, email, password)
}

func GetPassword(email string) (string, error) {
	return keyring.Get(serviceName, email)
}
