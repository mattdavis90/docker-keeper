# docker-keeper
docker-keeper is designed to keep you docker-swarm updates with the latest images from you CI pipeline.

It can be deployed as simply as

	version: "3.7"

	services:
      keeper:
      image: registry.abitof.space/docker-keeper:latest
      environment:
        - KEEPER_SECRET_KEY=YourSecretKeyGoesHere
      placement:
        constraints:
          - "node.role == manager"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock

This will start a docker-keeper server listening on port 8000 by default

You should then label other services with "keeper.id". This ID is then used to target this service from the API. For example;

    services:
      nginx:
        image: my-registry/nginx:0.1.0
        deploy:
          labels:
            keeper.id: "my-nginx"

The following environmental variables can be used to override functionality

| Variable          | Required | Default | Description                       |
|-------------------|----------|---------|-----------------------------------|
| KEEPER_ROOT       | No       | "/ "    | The path at which to host the API |
| KEEPER_BIND       | No       | ":8000" | The address to bind the server to |
| KEEPER_SECRET_KEY | Yes      |         | Secret key for your deployment    |

GET to *KEEPER_ROOT* will list all services managed by docker-keeper.
POST to *KEEPER_ROOT* with the following payload will update the given service.

    {
      "secretkey": "YourSecretKeyGoesHere",
      "id: "my-nginx",
      "image": "my-registry/nginx:0.2.0"
    }
