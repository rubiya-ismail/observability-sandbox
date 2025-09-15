## Build Docker Image

```
$ cd <root of project, folder containing services, and shared folder>
$ docker build --no-cache --progress=plain  -f services/user-service/Dockerfile -t user-service:latest .
```

## Create Shared Docker Network

```
$ docker network create observablity-net
```

## Run the Service

- Running service with a "service name" in "shared docker network"

```
$ docker run --name user-service --network observablity-net -p 8082:8082 user-service:latest
```

- Running service in detached mode

```
$ docker run -d --name user-service --network observablity-net -p 8082:8082 user-service:latest
```

- To re-run the above commands, remove (or rename) the 'registered' service name as follows:

```
$ docker rm user-service
```

## Stop the Service

```
$ docker stop user-service
```
