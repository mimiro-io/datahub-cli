# Mimiro Datahub CLI - content

## Usage

```bash
Usage:
 mim [flags]
 mim [command]

Available Commands:
 mim content list
 mim content add
 mim content show
 mim content delete

Flags:
     --disable-banner   Set to true to disable the banner
 -h, --help             Shows this help message
```

## About

Content entries can be any kind of content as long as it has an ID.
One typical use case is to store service configuration.

Content can be used as a central location to store configurations for
external microservices extending the features of the datahub.
The configurations are provided to the remotes by them
connecting to the datahub, and asking for content of a specific id.

Because of the dynamic nature of service configurations, it is hard to
display them in a common format, so json is chosen where applicable.

## Commands

The following commands are available:

 * list     - lists all contents
 * add      - adds or updates a content
 * show     - displays detailed json for the content
 * delete   - deletes a single content

## content list

 The list command lists all content entries on the server.
 Because the the flexible nature of the configurations, only the
 id column is shown.

 ```bash
Usage:
  mim content list [flags]

Flags:
  -h, --help   help for list

Global Flags:
      --disable-banner   Set to true to disable the banner

```
## content add

 Adds a content

 ```bash
Adds a new content or updates an existing. For example:
mim content add file=myfile.json

or

cat myfile.json | mim content add

Usage:
  mim content add [flags]

Flags:
  -f, --file string   The input file. Must be json.
  -h, --help          help for add
  -i, --id string     The id of the content to add. This overrides the file id.

Global Flags:
      --disable-banner   Set to true to disable the banner

```

 The command is able to either read from a file (--file) or from stdin. If a file flag
 is present, then that will be preferred instead of the stdin.

 You can override the id from the file or from stdin by setting --id=<some_id>. Ids must
 be unique, and should be url-friendly (ie "my-layer-config" instead of "My Layer Config").

 If you provide an existing id, the old configuration will be overwritten.


## content show

 Shows the details for a single content

 ```bash
Show a single content. For example:
mim content show --id="my-id"

or

mim content show my-id

Usage:
  mim content show [flags]

Flags:
  -h, --help        help for show
  -i, --id string   The id of the content to look for

Global Flags:
      --disable-banner   Set to true to disable the banner

```

 Because of the flexible nature of the contents, prettified json is printed to the console.
 If the content is a configuration, and your configuration contains secrets, for example passwords,
 then these will not be masked.

## content delete

Deletes a single content by its id.

```bash
Deletes a content. For example:
mim content delete --id="my-id"

or

mim content delete my-id

Usage:
  mim content delete [flags]

Flags:
  -h, --help        help for delete
  -i, --id string   The id of the content to delete.

Global Flags:
      --disable-banner   Set to true to disable the banner

```

 There is currently no protection against you deleting the wrong content, so please
 be careful when deleting.
