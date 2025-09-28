# Demo usage of the secret driver

_PS: You'll need Docker Swarm initialized for this to work._

```shell
docker swarm init
```

### Setup 1Password Connect

See [Get started with a 1Password Connect server](https://developer.1password.com/docs/connect/get-started/)


### Manual secret definition example

```shell
docker secret create \
    --driver op-connect-secret-driver:latest \
    --label vault="Test" \
    --label item="Test Secret" \
    --label field="username" \
    username
```

### Docker Stack example

```shell
docker stack deploy -c compose.yml test
```