package main

import (
	"os"
	"testing"

	"github.com/keploy/go-sdk/keploy"
)

func TestKeploy(t *testing.T) {
	os.Setenv("PORT", "8090")
	keploy.SetTestMode()
	go main()
	keploy.AssertTests(t)
}
