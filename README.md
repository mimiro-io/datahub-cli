# MIMIRO Data Hub CLI - `mim`

The MIMIRO Data Hub CLI, known as `mim`, provides command line control over a [MIMIRO data hub](https://github.com/mimiro-io/datahub) instance or any [Universal Data Specification](https://github.com/mimiro-io/universal-data-api-specification) (UDA) compliant endpoint. 

The `mim` client can be used to script and interact with MIMIRO to create datasets, load data, register jobs, test and execute transforms and queries. 

`mim` provides an extensive set of command help that can be accessed by typing:

```
mim help 
```

and for specific commands:

```
mim help COMMAND 

e.g.

mim help datasets
```

More documentation on use can be found in the [MIMIRO data hub](https://github.com/mimiro-io/datahub/blob/master/README.md) docs.

## Build

Requires Go.

```bash
make mim
```

## Installation

Use `make mim` to produce a binary in `bin/mim`. It is recommended to add the `bin` folder to your system's `PATH` so that the `mim` command is globally available. 

## Releases

If you want a pre-built binary then this can be obtained from the [releases](https://github.com/mimiro-io/datahub-cli/releases) page. Download the correct version, make sure it is renamed to `mim` locally, and that it is on the path.

## Connecting

`mim` can be used to connect to the MIMIRO data hub, MIMIRO data layers or any other UDA compliant endpoint. 

It is recommended to create an alias for each distinct service endpoint you want to connect to. To setup and connection to a local unsecured MIMIRO data hub instance create the following login alias:

```
mim login add --alias local --server http://localhost:8080 --type unsecured
```

To see all registered aliases and where they connect to use:

```
mim login ls
```

To connect to a service that has been secured with JWT tokens there are two options, either providing the JWT token as part of the alias definition or configuring OAuth authentication. This will depend on the service and you should check with the service documentation on how to connect.

Configuring to use a token directly:

```
mim login add --alias server1 \ 
              --server="https://my.datahub.server" \
              --token="<valid token>" 
```

Configuring to use an OAuth Token provider:

```
mim login add
       --alias local \
       --server "https://datahubapi.example.io" \
       --clientId "<valid clientId>" \
       --clientSecret "<valid clientSecret>" \
       --authorizer "https://auth.example.io/oauth/token" \
       --audience "https://datahubapi.example.io"
```

An OAuth configured alias will (re)authenticate and retrieve a token when needed. 

## Contributing

The MIMIRO data hub cli project welcomes contributions and constructive engagement, please read our [code of conduct](CODE-OF-CONDUCT.md) and [contributing guidelines](CONTRIBUTING.md) before creating issues or making PRs. 

## Change Log

We try and keep the [change log](CHANGELOG.md) up-to-date.
