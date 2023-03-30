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
mim login add --type=client --server="https://my.datahub.server" --token="<valid token>" --alias="server1"
```

You cannot change the alias on an existing profile, you need to delete it first.

There are 3 types of logins that are supported: token, client, user.

 * token - you must provide a valid token yourself
 * client - this is a combination if a client id and secret
 * user - log in as yourself

### User login profile

To use this option, you must provide a (DataHub) server and a login Authorizer.
For a user login type, the Authorizer should point to root, a "/login"-path will be automatically added.

Example:
```
mim login add 
    --type user \
    --server "https://my.datahub.server" \
    --authorizer "https://auth.example.io" \
    --audience "https://my.datahub.server"
    --clientId cc6aeb0f-fdc5-4040-973d-f27313e1c9e6

```

When you try to login using this type, a webserver with the port 31337 will be started on your machine.
The cli will then attempt to open up your browser with the url to the authorizer login, with a callback 
url to the previously started web server added. The link will also be clickable in the console, in case it fails
to open.

Once you have logged in, the auth server must callback to the local server with the callback url provided. 
The callback url must include a login code that can be further exchanged for a refresh token.

This is also known as an "OAuth authorization code flow".

### Client login profile

A client login should be used if you are using a machine to represent you.

```
mim login add 
    --type client \
    --alias local \
    --server "https://api.example.io" \
    --clientId "<valid clientId>" \
    --clientSecret "<valid clientSecret>" \
    --authorizer "https://auth.example.io/oauth/token" \
    --audience "https://api.example.io"
```

The clientId and clientSecret must be valid for the authorizer server. If you ommit the audience, the audience will be
the same as the server, this is not the case if you are running against a local server, so you should set it appropriately.

Tokens are cached locally, so the client will only refresh the token once it is expired.

### Token login profile

You just need a valid token for the operation you are attempting to run.

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