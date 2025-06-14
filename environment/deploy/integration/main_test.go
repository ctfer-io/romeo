package integration

import (
	"fmt"
	"os"
	"testing"
)

var (
	Server string
)

func TestMain(m *testing.M) {
	server, ok := os.LookupEnv("SERVER")
	if !ok {
		//nolint:lll // the error message is not maintained
		fmt.Println("Environment variable SERVER is not set, please indicate the domain name/IP address to reach out the cluster.")
	}
	Server = server

	os.Exit(m.Run())
}
