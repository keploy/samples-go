package main

import (
	// "bytes"
	// "encoding/json"
	"fmt"
	// "net/http"
	// "net/http/httptest"
	"testing"
	"time"
	"log"
	"os"

	// "github.com/gin-gonic/gin"
	"github.com/keploy/go-sdk/v2/keploy"
)

// main_test.go file
func TestMain(m *testing.M) {
	// Start the keploy GraphQL server
	err := keploy.RunKeployServer(int64(os.Getpid()), 10, "./", 6789)
	if err != nil {
		log.Fatal("failed to start the keploy server", err)
	}

	code := m.Run()  // Run all tests

        // Stop the keploy server
	keploy.StopKeployServer()
        os.Exit(code)
}

func TestKeploy(t *testing.T) {
        // Fetch the keploy recorded test-sets
	testSets, err := keploy.FetchTestSets()
	if err != nil {
		t.Log(err)
	}

	fmt.Println("TestSets:", testSets)
	fmt.Println("starting user application")

	result := true
	for _, v := range testSets {
		// Run the test-set sequentially
		testRunId, err := keploy.RunTestSet(v)
		go main()
		if err != nil {
			t.Log(err)
		}
		var testRunStatus keploy.TestRunStatus
		for {

			//check status every 2 sec
			time.Sleep(2 * time.Second);
			testRunStatus, err = keploy.FetchTestSetStatus(testRunId)
			if err != nil {
				t.Log(err)
			}
			if (testRunStatus == keploy.Running) {
				fmt.Println("testRun still in progress");
				continue;
			}
			break;
		}
		if (testRunStatus == keploy.Failed) {
			fmt.Println("testrun failed for", v);
			result = false;
		} else if (testRunStatus == keploy.Passed) {
			fmt.Println("testrun passed for", v);
		}
		// trigger shutdown event
		keploy.LaunchShutdown()
	}

	if !result {
		t.Error("the testrun failed")
	}
	t.Log("the overall result:", result)
}