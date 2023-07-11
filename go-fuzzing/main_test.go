package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/keploy/go-sdk/keploy"
)

func TestKeploy(t *testing.T) {
	keploy.SetTestMode()
	go main()
	keploy.AssertTests(t)
}

func FuzzValidate(f *testing.F) {
	router := gin.Default()
	integrateKeploy(router)
	router.POST("/validateInformation", validateInformation)

	// add seed
	f.Add("Tom", 26)
	f.Add("Lucy", 24)

	// start fuzzing
	f.Fuzz(func(t *testing.T, name string, age int) {
		w := httptest.NewRecorder()
		bin, _ := json.Marshal(&UserInformation{
			Name: name,
			Age:  age,
		})

		req, err := http.NewRequest("POST",
			"/validateInformation",
			bytes.NewBuffer(bin),
		)
		if err != nil {
			t.Errorf("failed to create http request: %s", err)
		}
		router.ServeHTTP(w, req)

		// check api response
		if !IsValidName(name) {
			if w.Code != http.StatusBadRequest {
				t.Fatalf("status should be %d for name=%s", http.StatusBadRequest, name)
			}
		}
		if !IsValidAge(age) {
			if w.Code != http.StatusBadRequest {
				t.Fatalf("status should be %d for age=%d", http.StatusBadRequest, age)
			}
		}

		if w.Code != http.StatusOK {
			if w.Code != http.StatusBadRequest {
				t.Fatalf("status should be %d for name=%s, age=%d", http.StatusOK, name, age)
			}
		}
	})
}
