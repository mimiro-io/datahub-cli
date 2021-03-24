# Change Log

##  24/03/2021

* `mim jobs ls` or `mim jobs list` now returns only the first 50 characters of error messages
* `mim jobs ls` or `mim jobs list` prints triggers in a new, shorter format.
  * the prefix symbol encodes the jobType. `>` is incremental and `>>` is fullsync
  * color encodes the triggerType. cyan are cron triggers and lightblue are onchange jobs
* `mim jobs history <jobid>` or `mim jobs history --id <jobid>` is a new subcommand. It returns the history entry for the last job run, with full error text

## 22/03/2021

Initial open source release.
