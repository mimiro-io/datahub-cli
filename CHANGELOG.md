# Change Log

## 29/09/2021

* `mim jobs ls` or `mim jobs list` now has the ability to filter in existing jobs. by using `mim jobs ls --filter foo,bar` you will only get the result set that includes either foo or bar.
* `mim jobs ls` or `mim jobs list` now returns only the first 30 characters of error messages
* triggers output in `mim jobs ls` or `mim jobs list` is moved to `mim jobs ls --verbose` or `mim jobs list --verbose`
* added new properties to the job struct `Title`, `Tags` and `Description`

##  24/03/2021

* `mim jobs ls` or `mim jobs list` now returns only the first 50 characters of error messages
* `mim jobs ls` or `mim jobs list` prints triggers in a new, shorter format.
  * the prefix symbol encodes the jobType. `>` is incremental and `>>` is fullsync
  * color encodes the triggerType. cyan are cron triggers and lightblue are onchange jobs
* `mim jobs history <jobid>` or `mim jobs history --id <jobid>` is a new subcommand. It returns the history entry for the last job run, with full error text

## 22/03/2021

Initial open source release.
