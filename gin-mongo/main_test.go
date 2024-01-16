package main

import (
	"github.com/keploy/go-sdk/keploy"
	"testing"
)

var messi1 int

func TestKeploy(t *testing.T) {
	keploy.SetTestMode()
	go main()
	keploy.AssertTests(t)
}
