# Dispatchr

Dispatchr is a system for scheduling and executing background tasks.

It was born out of the desire to have a simpler, more reliable, and more observable
background task system than what already exists within the Python ecosystem. However,
since this project aims to develop a specification first and implementation later,
there's nothing preventing an implementation in any other language or framework.

## Components

Dispatchr has the following components:

### Task Recorder

Recorder is an HTTP application for receiving tasks and persisting them to
a database. A Recorder implementation CAN be hosted within your existing web
application, or run as a standalone service.

### Task Scheduler

Scheduler is a CLI application that is responsible for pushing tasks to your application
to be executed. It pulls work items from the database and calls the HTTP endpoint associated
with that task. HTTP status codes returned from your application control whether a task is
marked as completed, failure, or if it needs to be retried in the future.


## Motivation

[Celery](https://docs.celeryq.dev/) is perhaps the best known and most used background
task system in the Python world. Unfortunately it suffers from a range of issues that
make it difficult to trust in production:

- Many bugs that go unfixed for years
- Highly complex software, spread across multiple projects, that make it very
  difficult to debug or fix issues, or otherwise contribute to the project
- It supports a number of different brokers/queues such as RabbitMQ, Redis, and
  SQS with varying levels of support and features
- [At least once delivery](https://www.cloudcomputingpatterns.org/at_least_once_delivery/)
  is exceedingly difficult to achieve
- It's easy to lose tasks transparently, since the only persistence is within the
  task queues themselves. You have to have expertise in whichever queue backend
  you select
- Monitoring and observability aren't built in
- It requires both the application server and the task server to share code
- Storing of results is awkward and can lead to overwhelming your queueing system with
  large amounts of data that should be stored within the application

After building a system using [google cloud tasks](https://cloud.google.com/tasks) a lot
of these problems went away. It relies on persistent queues that automatically fires tasks
at your HTTP handlers. Success, failure, and retries are communicated back via HTTP status
codes. Task execution no longer requires a separate fleet of application servers that pull
work - your existing set of HTTP application servers can be delivered work.

HTTP is a useful protocol for scheduling and executing tasks, since all of your existing
tooling continues to work:

- Rate limiting
- Routing
- Proxying
- Logging / Metrics
- Observability or APM tooling
- Other middleware

Despite the increase in reliability and simplicity, cloudtasks still required some non-trivial
setup, and you weren't able to inspect the tasks currently waiting within the queue to be
executed.

[How do you cut a monolith in half](https://programmingisterrible.com/post/162346490883/how-do-you-cut-a-monolith-in-half) talks about some of the problems of using message brokers
as a means for scheduling work. Though it is mostly in the context of breaking apart services
into a set of distributed systems, many of the lessons translate directly to background task
systems. Some of the highlights:

- Tasks should be persisted to a database
- The scheduler should be responsible for handling failures, retries, and backpressure

[Dropbox ATF](https://dropbox.tech/infrastructure/asynchronous-task-scheduling-at-dropbox) follows
similar principles in their design, such as a database that stores tasks and their statuses. However,
they still use a queue (SQS) between their scheduler and worker subsystems.
