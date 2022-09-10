# Mimiro Datahub CLI - datasets

```bash
Manage datahub datasets from cli such as create, delete, describe and so on. See available Commands.
Examples:
  mim dataset changes --name=cow.Animal --limit=5

or:

  mim dataset get cow.Animal

Usage:
  mim dataset list [flags]
  mim dataset create [flags]
  mim dataset delete [flags]
  mim dataset get [flags]
  mim dataset entities [flags]
  mim dataset changes [flags]
  mim dataset rename [flags]
  mim dataset store [flags]

Flags:
  -n, --name        The dataset to list entities from
  -f, --format      The output format. Valid options are: term|pretty|raw
  -s, --since       Send a since token to the server
      --limit       Limits the number of entities to list
  -h, --help        Help for dataset
  -f, --filename    Used to indicate the file containing entities to load

Global Flags:
      --disable-banner   Set to true to disable the banner

```
