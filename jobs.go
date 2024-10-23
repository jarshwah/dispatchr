package dispatchr

import (
	"context"
	"fmt"
	"net/url"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivertype"
)

// DispatchJob is a job that dispatches a task to a URL target, using
// returned HTTP status codes to indicate success or failure.
type DispatchJob struct {
	UrlTarget *url.URL `json:"urlTarget"`
	TaskName  string   `json:"taskName"`
	TaskArgs  []byte   `json:"taskArgs"`
}

func (DispatchJob) Kind() string { return "dispatchr_dispatchJob" }

type DispatchJobWorker struct {
	river.WorkerDefaults[DispatchJob]
}

func (w *DispatchJobWorker) Work(ctx context.Context, job *river.Job[DispatchJob]) error {
	fmt.Println("Dispatching job", job.ID, "to", job.Args.UrlTarget)
	return nil
}

func AddTaskExample(ctx context.Context, client *river.Client[pgx.Tx]) {
	job := &DispatchJob{
		UrlTarget: &url.URL{Scheme: "http", Host: "example.com", Path: "/"},
		TaskName:  "example",
		TaskArgs:  []byte(`{"example": "example"}`),
	}
	if _, err := client.Insert(ctx, job, nil); err != nil {
		fmt.Fprintf(os.Stderr, "Unable to enqueue job: %v\n", err)
		os.Exit(1)
	}

}

func AddTask(ctx context.Context, client *river.Client[pgx.Tx], urlTarget *url.URL, taskName string, taskArgs []byte) (*rivertype.JobInsertResult, error) {
	job := &DispatchJob{
		UrlTarget: urlTarget,
		TaskName:  taskName,
		TaskArgs:  taskArgs,
	}
	jobResult, err := client.Insert(ctx, job, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to enqueue job: %v", err)
	}
	return jobResult, nil

}

func CreateClient(ctx context.Context, dbPool *pgxpool.Pool) (*river.Client[pgx.Tx], error) {
	workers := river.NewWorkers()
	river.AddWorker(workers, &DispatchJobWorker{})

	client, err := river.NewClient(riverpgxv5.New(dbPool), &river.Config{
		// TODO: Queue configuration to be argument
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 100},
		},
		Workers: workers,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to create client: %v", err)
	}

	return client, nil
}

func CreateDatabasePool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	dbPool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %v", err)
	}

	return dbPool, nil
}
