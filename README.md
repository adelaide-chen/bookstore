How to build bridge between MongoDB container and bookstore container


1. Establish bridge network:
docker network create --driver bridge <name of network>
docker network create --driver bridge mongo-net

1a. Check new bridge network:
docker network inspect mongo-net

2. Connect mongo server container to bridge network
docker run -dit --name <uri mongo server name> --network <name of network> <image name>
docker run -dit --name mongodb --network mongo-net mongo:latest

3. Create new container (called bookstore) and connect network to project, and set env var DB to mongodb
docker run --name <unique ID> -dit -p 8080:8080 --network <unique ID1>_network1 <username>/<project name> --env DB=<env var>
docker run -dit -p 8080:8080 --name bookstore --network mongo-net achen141/bookstore --env DB=mongodb