# Mimiro Datahub CLI - jobs

Manage datahub jobs from cli such as add, delete, describe and so on. 
```
%s
```

## How to use

Most, if not all commands can be called with the job id as a parameter.

For example, instead of doing `mim jobs operate -o run --id=<some-job-id>`, 
you can instead write `mim jobs operate -o run <some-job-id>`

## Jobs

A Datahub server Job is the main mechanism to move and transform data. This specifies where the
dataset comes from (external or internal), where it goes (again external or internal), and it also
applies any transformation to the dataset.

Any Job is internally made up from 3 pieces in what is known as a Pipeline.
The pieces are: Source -> Transform -> Sink, and the dataset also moves in this direction.

Source and Sink are always required, but a Transform is only added when needed, and should also mostly
occur on OnEvent jobs.

### Job types

There are 4 different job types:
 * RunOnce
 * Full Sync
 * Incremental Sync
 * OnChange

### RunOnce

If the RunOnce flag = true, then this job is run once when added, and then deleted when finished.
This job type takes precedence to any other, so you cannot combine it with the other types.

Instead of using this to run a job, it can be smarter to just add a job with its enabled flag set
to false, and use operate to run it.

### Full Sync

FullSync has to have the fullSyncSchedule set to a valid [Cron](https://godoc.org/github.com/robfig/cron) value.
This can be combined with an incremental schedule to have the job run incrementally for regular schedules, but
able to run a full sync maybe once a day.

Full sync is intended to fetch all entities in a remote dataset, and only when it doesn't support changes and/or
 deletes. 

### Incremental Sync

IncrementalSync is a variation on the FullSync, but is intended to run on a more frequent schedule to fetch 
changes, if the dataset source supports it.

There is protection in the Job Scheduler to prevent jobs from running in parallel, so a Full Sync will be 
rescheduled if an incremental is running, but an Incremental will be skipped if a Full is running.

Incremental jobs should finish before the next is scheduled, so make sure your schedule is correct. 

### OnChange

An OnChange Job ( onChange = a dataset) is a special event that triggers on changes to the specified dataset
instead of a schedule.

You use this job to run a transformation on an existing dataset.

## List

```
mim jobs list
```

This command is the simplest command, and it is used to show a list of all server jobs.

| Id   | Source   | Transform   | Sink   | Paused   | Schedule full/incr   | Event   | Last Run   | Last Duration   | Error   |
| --- | ------ | --------- | ---- | ------ | ------------------ | ----- | -------- | ------------ | ----- |
| test-import | DatasetSource |   | DatasetSink | false | /@every 1m |   | 2020-11-19T14:56:17+01:00 | 30ms | |

 * Id - This is the job id
 * Source - Dataset source
 * Transform - Dataset transform if any
 * Sink - Dataset sink
 * Paused - Is the job paused or not
 * Schedule full/incremental - The full and/or the incremental schedule for the job
 * Event - If the job is an event, then this is the dataset it reacts on changes on
 * Last Run - When the job last was run
 * Last Duration - How long did the last job run
 * Error - If any errors occurred, they are shown here

## Add

Adds a new, or updates an existing job.

```
mim jobs add --file=path/to/job.json
```
Add can also take input from StdIn, so you can do this instead:
```
cat path/to/job.json | mim jobs add
```

A Job has this format:

```
{
    "id" : "event-copy-vet",
    "fullSyncSchedule"    : "",
    "incrementalSchedule" : "",
    "onChange": "db.People",
    "runOnce": false,
    "paused": true,
    "source" : {
        "Type" : "DatasetSource",
        "Name": "db.People"
    },
    "transform": {
        "Type": "JavascriptTransform"
        "Code": "<base64 encoded javascript>"
    },
    "sink" : {
        "Type" : "DatasetSink",
        "Name" : "test.People"
    }
}

```

Note that both source and sink are required, but transform is not. You can however add a Transform
from the Jobs cli, although it is much better to do this from the Transform cli.

### Source

There are (currently) 2 supported Source types:
 * DatasetSource
 * HttpDatasetSource
 
More datasources are planned.

### DatasetSource

This is used when the source is an internal dataset that already exists in the Datahub. 
 
Example:
```
"source" : {
    "Type" : "DatasetSource",
    "Name": "db.People"
},
```
Both Type and Name are required fields.

### HttpDatasetSource

This is used when the source is an external dataset with endpoints that support the Datahub integration
protocol.

Example:
```
"source" : {
    "Type" : "HttpDatasetSource",
    "Url" : "https://localhost:4343/datasets/db.people/changes",
    "TokenProvider": "Auth0TokenProvider"
},
```

Type and Url are required fields.

| Field  | Value |
| --- | --- |
| Type | HttpDatasetSource |
| Url | The dataset endpoint |
| TokenProvider | Auth0TokenProvider is currently the only supported provider |
| User | The user in a basic login |
| Password | The password for a basic login |

Note that User and Password combination is not currently supported





 
