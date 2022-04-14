

package main

import (
	"github.com/keploy/go-sdk/keploy"
	"os"
	"testing"
)

func TestKeploy(t *testing.T) {
	// change port so that test server can run concurrently
	os.Setenv("PORT", "8090")

	keploy.SetTestMode()
	go main()
	keploy.AssertTests(t)
}