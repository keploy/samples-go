package main

import (
	"es4gophers/logic"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func putURL(c *gin.Context) {
	var m map[string]string

	err := c.ShouldBindJSON(&m)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to decode req"})
		return
	}
	u := m["indexName"]
	if u == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing indexName param"})
		return
	}
	ctx = logic.LoadMoviesFromFile(ctx)
	logic.IndexMoviesAsDocuments(c.Request.Context(), ctx, u)
	t := time.Now()
	c.JSON(http.StatusOK, gin.H{
		"ts":    t.UnixNano(),
		"index": u + " indexed !",
	})
}

func getURL(c *gin.Context) {
	var m map[string]string
	err := c.ShouldBindJSON(&m)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to decode req"})
		return
	}
	indexName := m["indexName"]
	docId := m["docId"]
	fmt.Println(indexName, " ", docId)
	if indexName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing indexName param"})
		return
	}
	if docId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing docId param"})
		return
	}
	movieName := logic.QueryMovieByDocumentID(c.Request.Context(), ctx, indexName, docId)
	t := time.Now()
	c.JSON(http.StatusOK, gin.H{
		"ts":         t.UnixNano(),
		"Movie Name": movieName,
	})
}

func deleteURL(c *gin.Context) {
	var m map[string]string
	err := c.ShouldBindJSON(&m)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to decode req"})
		return
	}
	indexName := m["indexName"]
	docId := m["docId"]
	if indexName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing indexName param"})
		return
	}
	if docId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing docId param"})
		return
	}
	logic.DeleteMovieByDocumentID(c.Request.Context(), ctx, indexName, docId)
	t := time.Now()
	c.JSON(http.StatusOK, gin.H{
		"ts": t.UnixNano(),
	})
}
