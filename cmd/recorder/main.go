package main

import (
	"context"
	"dispatchr"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
)

func main() {
	ctx := context.Background()
	setupClient(ctx, os.Getenv("DATABASE_URL"))

	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})
	r.POST("/task", func(c *gin.Context) {
		var req AddTaskRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Convert the string URL to a URL struct
		urlTarget, err := url.ParseRequestURI(req.UrlTarget)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// json-ify the task arguments
		jsonArgs, err := json.Marshal(req.TaskArgs)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		job, err := dispatchr.AddTask(ctx, dispatchrClient, urlTarget, req.TaskName, jsonArgs)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "task added", "job": job.Job.ID})
	})
	r.Run("localhost:9090")
}

var dispatchrClient *river.Client[pgx.Tx]

func setupClient(ctx context.Context, dbURL string) {
	dbPool, err := dispatchr.CreateDatabasePool(ctx, dbURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}

	client, err := dispatchr.CreateClient(ctx, dbPool)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create client: %v\n", err)
		os.Exit(1)
	}
	dispatchrClient = client
}

type AddTaskRequest struct {
	UrlTarget string                 `json:"target"`
	TaskName  string                 `json:"taskName"`
	TaskArgs  map[string]interface{} `json:"taskArgs"`
}
