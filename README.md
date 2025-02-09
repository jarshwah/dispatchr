# dispatchr

A background task service built on [River Queue](https://riverqueue.com/).

This software is under development, is proof-of-concept *only*, and has not yet been productionised (or, indeed, fully implemented).

## What is this?

A database-backed background task service.

Clients submit tasks over HTTP to dispatchr-recorder, which stores the task in a database, and signals success to the client - letting it know the task has been successfully confirmed.

A separate process, dispatchr-worker, reads tasks from the database, executes them, and records the results.

A task is a JSON object with the following fields:

```json
{
  "target": "https://client.domain/endpoint",
  "taskName": "client-task-name",
  "taskArgs": {"client": "args", "go": "here", "nested": {"data": "too"}},
}
```

The dispatchr worker will pick up this task, POST `taskName` and `taskArgs` to `urlTarget`, and record the result.

If the client responds with a 2xx status code, the task is marked as successful. If the client responds with a 4xx or 5xx status code, the task is marked as failed.

## Why would I use this?

Simply, because most background task systems suck. They:

- are tightly coupled to the language or framework of the main application
- do not come with good defaults
- are typically backed by a message queue, which does not allow an operator to inspect the *state* of the system
- prioritise throughput over correctness
- are difficult to orchestrate

The core design principles of dispatchr is that HTTP is useful to a background task service because:

- Your application is already built to handle HTTP requests
- You do not need to operate a separate instance of your application that pulls work
- Your existing HTTP middleware can be used to observe background work and apply rate limits
- Retry logic can be handled simply with HTTP status codes

Your application submits tasks over HTTP to dispatchr. Publish confirms are enabled *by default*, as choosing to ignore the HTTP response is atypical.

Tasks are executed over HTTP by your application. Dispatchr takes care of the scheduling, prioritisation, backpressure, and retries.
Have a separate lambda service you want to execute asynchronously? Submit a task to dispatchr with a URL target of your lambda service.

# Running dispatchr

Build the binaries:

```sh
make build
```

Create a database for dispatchr to use, and export the `DATABASE_URL` environment variable:

```sh
createdb dispatchr
export DATABASE_URL=postgres://localhost/dispatchr

river migrate-up --database-url "$DATABASE_URL"
```

Start the recorder service, which listens for incoming tasks:

```sh
DATABASE_URL=... go run ./cmd/recorder
```

Start the worker service, which will dispatch tasks to your application:

```sh
DATABASE_URL=... go run ./cmd/worker
```

(Optional): Start up [River UI](https://github.com/riverqueue/riverui) to inspect the state of the system:

```sh
curl -L https://github.com/riverqueue/riverui/releases/latest/download/riverui_darwin_arm64.gz | gzip -d > riverui
chmod +x riverui
DATABASE_URL=... ./riverui
```

Submit your tasks!

```
curl --request POST \
  --url http://localhost:9090/task \
  --data '{
  "target": "https://client.domain/endpoint",
  "taskName": "client-task-name",
  "taskArgs": {
    "data": "here"
  }
}'
```
