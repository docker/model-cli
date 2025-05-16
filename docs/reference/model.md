# docker model

<!---MARKER_GEN_START-->
Docker Model Runner (EXPERIMENTAL)

### Subcommands

| Name                                            | Description                                                            |
|:------------------------------------------------|:-----------------------------------------------------------------------|
| [`inspect`](model_inspect.md)                   | Display detailed information on one model                              |
| [`install-runner`](model_install-runner.md)     | Install Docker Model Runner                                            |
| [`list`](model_list.md)                         | List the available models that can be run with the Docker Model Runner |
| [`logs`](model_logs.md)                         | Fetch the Docker Model Runner logs                                     |
| [`package`](model_package.md)                   | package a model                                                        |
| [`pull`](model_pull.md)                         | Download a model                                                       |
| [`push`](model_push.md)                         | Upload a model                                                         |
| [`rm`](model_rm.md)                             | Remove models downloaded from Docker Hub                               |
| [`run`](model_run.md)                           | Run a model with the Docker Model Runner                               |
| [`status`](model_status.md)                     | Check if the Docker Model Runner is running                            |
| [`tag`](model_tag.md)                           | Tag a model                                                            |
| [`uninstall-runner`](model_uninstall-runner.md) | Uninstall Docker Model Runner                                          |
| [`version`](model_version.md)                   | Show the Docker Model Runner version                                   |


### Options

| Name                | Type     | Default                  | Description                                                                                                                           |
|:--------------------|:---------|:-------------------------|:--------------------------------------------------------------------------------------------------------------------------------------|
| `--config`          | `string` | `/root/.docker`          | Location of client config files                                                                                                       |
| `-c`, `--context`   | `string` |                          | Name of the context to use to connect to the daemon (overrides DOCKER_HOST env var and default context set with "docker context use") |
| `-D`, `--debug`     | `bool`   |                          | Enable debug mode                                                                                                                     |
| `-H`, `--host`      | `list`   |                          | Daemon socket to connect to                                                                                                           |
| `-l`, `--log-level` | `string` | `info`                   | Set the logging level ("debug", "info", "warn", "error", "fatal")                                                                     |
| `--tls`             | `bool`   |                          | Use TLS; implied by --tlsverify                                                                                                       |
| `--tlscacert`       | `string` | `/root/.docker/ca.pem`   | Trust certs signed only by this CA                                                                                                    |
| `--tlscert`         | `string` | `/root/.docker/cert.pem` | Path to TLS certificate file                                                                                                          |
| `--tlskey`          | `string` | `/root/.docker/key.pem`  | Path to TLS key file                                                                                                                  |
| `--tlsverify`       | `bool`   |                          | Use TLS and verify the remote                                                                                                         |


<!---MARKER_GEN_END-->

