package util

import (
	"proxy-api-server/log"
)

func CommonError(err error) error {
	log.Error(err)
	return err
}
func Error(message string, err error) error {
	log.Error("%s. Error: %s ", message, err)
	return err
}
func DashboardError(err error, UID string) error {
	log.Error("Error: UID:\n%s\n%s", UID, err)
	return err
}
