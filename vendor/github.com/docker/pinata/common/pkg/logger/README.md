## Common logging functions for the Go code

To use:

```
import "github.com/docker/pinata/common/pkg/logger"
var log = logger.Default
...
log.Infof("hello %s", "there")
```

Note this is included from almost everywhere in common/ mac/ and windows/ so
we must make this package stand-alone and not create cyclic dependencies.

## Mac

On Mac we send the logs to 2 places:
1. ASL viewable in Console.app
2. files, one per service, in ~/Library/Containers/com.docker.docker/Data/logs/{vm, host}

## Windows

On Windows the C# manages the logs for the Go processes, but the VM logs are streamed into
`%PROGRAMDATA%\DockerDesktop\log\vm`
