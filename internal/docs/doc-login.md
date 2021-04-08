# Mimiro Datahub CLI - login

## Usage

Log in to the datahub, or work with login profiles
```
%s
```

## Logging in

```
mim login use my-server
```
or just
```
mim login my-server
```

## About

Login enables the user to login to a datahub server.

To log in to a server, you first need to add a login profile.

### Adding profile

You can create a profile by running the "mim login add" command.

```
mim login add --server="https://my.datahub.server" --token="<valid token>"
```

If you don't provide an alias, the server name is used as an alias, so you should
add an alias.

```
mim login add --server="https://my.datahub.server" --token="<valid token>" --alias="server1"
```

You cannot change the alias on an existing profile, you need to delete it first.

Since a token has a limited validity, we suggest you use a setup with a clientId and a clientSecret
instead.

```
mim login add 
    --alias local \
    --server "https://api.example.io" \
    --clientId "<valid clientId>" \
    --clientSecret "<valid clientSecret>" \
    --authorizer "https://auth.example.io/oauth/token" \
    --audience "https://api.example.io"
```

The clientId and clientSecret must be valid for the authorizer server. If you ommit the audience, the audience will be
the same as the server, this is not the case if you are running against a local server, so you should set it appropriately.

Scope has a default value of "app_credentials", and that is correct against the mimiro auth server, however to be compatible
against auth0, you need to change it to "client_credentials".

There is (currently) no caching of tokens, so if you use Auth0, be advised that this can incur additional costs.


### Listing profiles

You can list your registered profiles.

```
mim login ls
```

### Updating profiles

You can update a profile by calling add again with the same alias.

### Deleting profile

Delete a profile by calling delete on the alias

```
mim login delete --alias="server1"
```
or
```
mim login delete server1
```

### Logging out

You can remove the current active login by calling "mim logout".
This will only set the activelogin key in the config to "" (blank).

To fully remove a login, you must delete the profile.

When you are logged out, you can use exported env variables for SERVER and TOKEN
to directly access a datahub.

Example:

```
export SERVER="http://localhost:8080"
export TOKEN="..."

mim jobs ls
```