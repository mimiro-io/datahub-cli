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
occur on `onchange` jobs.

### Job triggers

A Job can have one or more trigger configurations. A trigger  consists of a `triggerType` and a `jobType`, and depending
on `triggerType` either `schedule` or `monitoredDataset` as extra property.

There are 2 different job types:
 * `fullsync`
 * `incremental`

A `fullsync` job is intended to fetch all entities in a remote dataset, and only when it doesn't support changes and/or
deletes.

`incremental` is a variation on the FullSync, but is intended to run on a more frequent schedule to fetch
changes, if the dataset source supports it.

There is protection in the Job Scheduler to prevent jobs from running in parallel, so a Full Sync will be
rescheduled if an incremental is running, but an Incremental will be skipped if a Full is running.

Incremental jobs should finish before the next is scheduled, so make sure your schedule is correct.


There are 2 different trigger types:
 * `cron`
 * `onchange`

A `cron` trigger has to have a `schedule` set to a valid [Cron](https://godoc.org/github.com/robfig/cron) value.

An `onchange` Job ( onChange = a dataset) is a special event that triggers on changes to the specified dataset
instead of a schedule. The specified dataset to monitor is set with the `monitoredDataset` property of the trigger.
You use this job to run a transformation on an existing dataset.

## List

```
mim jobs list
```

This command is the simplest command, and it is used to show a list of all server jobs.

| Title          | Paused | Tags         |Source  | Transform | Sink    | Last Run                  | Last Duration | Error |
|----------------|--------|--------------|--------|-----------|---------|---------------------------|---------------|-------|
| test-import    | false  | import,tests |http    |           | Dataset | 2020-11-19T14:56:17+01:00 | 2m            |       |
| import-person  | false  | import       |Http    |           | Dataset | 2020-11-19T14:56:17+01:00 | 30ms          |       |
| import-order   | false  | import       |Http    |           | Dataset | 2020-11-19T14:56:17+01:00 | 30ms          |       |
| person-crm     | false  | crm          |Dataset |           | Http    | 2020-11-19T14:56:17+01:00 | 379ms         |       |
| order-crm      | false  | crm          |Dataset |           | Http    | 2020-11-19T14:56:17+01:00 | 30ms          |       |

 * Title - This is the job title
 * Paused - Is the job paused or not
 * Source - Dataset source
 * Transform - Dataset transform if any
 * Sink - Dataset sink
 * Last Run - When the job last was run
 * Last Duration - How long did the last job run
 * Error - If any errors occurred, they are shown here

```
mim jobs list --verbose
```

By using --verbose more columns can be displayed. the extra columns are:

* Id
* Triggers
```
>         = incremental
>>        = fullsync
cyan      = schedule
lightblue = onchange
```

### Filter
The list command support filtering the result by pattern matching.
```
mim jobs list --filter "title=person"
```
This command will return a subset of the jobs that exist based on filtering criteria.

| Title          | Paused | Tags         |Source  | Transform | Sink    | Last Run                  | Last Duration | Error |
|----------------|--------|--------------|--------|-----------|---------|---------------------------|---------------|-------|
| import-person  | false  | import       |Http    |           | Dataset | 2020-11-19T14:56:17+01:00 | 30ms          |       |
| person-crm     | false  | crm          |Dataset |           | Http    | 2020-11-19T14:56:17+01:00 | 379ms         |       |

The filter is exclusive by default, meaning all filters must match for a row to be returned. To make it inclusive (return all results matching one or more filters), add `--filterMode inclusive`.


Filters can be combined to narrow down the result even more. *Remember to use quotes when combining filters*
```
mim jobs list --filter "tags=tests;source=http"
```
This will return the following:

| Title          | Paused | Tags         |Source  | Transform | Sink    | Last Run                  | Last Duration | Error |
|----------------|--------|--------------|--------|-----------|---------|---------------------------|---------------|-------|
| test-import    | false  | import,tests |http    |           | Dataset | 2020-11-19T14:56:17+01:00 | 2m            |       |



It is also possible to add multiple values for a single filter:
```
mim jobs ls --filter "source=http,dataset;sink=http"
```
This will return rows matching source=http and sink=http as well as source=dataset and sink=http:

| Title          | Paused | Tags         |Source  | Transform | Sink    | Last Run                  | Last Duration | Error |
|----------------|--------|--------------|--------|-----------|---------|---------------------------|---------------|-------|
| test-import    | false  | import,tests |http    |           | Dataset | 2020-11-19T14:56:17+01:00 | 2m            |       |
| import-person  | false  | import       |Http    |           | Dataset | 2020-11-19T14:56:17+01:00 | 30ms          |       |
| import-order   | false  | import       |Http    |           | Dataset | 2020-11-19T14:56:17+01:00 | 30ms          |       |
| person-crm     | false  | crm          |Dataset |           | Http    | 2020-11-19T14:56:17+01:00 | 379ms         |       |
| order-crm      | false  | crm          |Dataset |           | Http    | 2020-11-19T14:56:17+01:00 | 30ms          |       |


All columns can be used for filtering. See below list for allowed syntax for each column:
```
title=mystringhere
tags=mytag
id=myidstring
paused=true
source=dataset
sink=http
transform=javascript
error=my error message
duration>10s or duration<30ms
lastrun<2020-11-19T14:56:17+01:00 or lastrun>2020-11-19T14:56:17+01:00
triggers=@every 60 or triggers=fullsync or triggers=person.Crm
```

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
    "triggers": [
        {
            "triggerType": "onchange",
            "monitoredDataset": "db.People"
        }
    ],
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

## Delete

```
mim job delete --id <id>
```

## Status

```
mim jobs status
```

This command lists the currently running jobs, with job-id and starting time as table columns.

## History

```
mim jobs history <job-id>
```

This command shows information about the last run of the given job
