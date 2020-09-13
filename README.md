# gkgraph

Demo event ingestion service using

* [go-kit](https://github.com/go-kit/kit)
* [kit](https://github.com/kujtimiihoxha/kit)
* [neo4j](https://github.com/neo4j) 

## service

```
cd event
go build
```

```
Usage of event:
      --debug_url string   Debug and metrics URL (default ":8080")
      --grpc_url string    gRPC URL (default ":8082")
      --http_url string    HTTP URL (default ":8081")
      --neo4j string       Neo4j bolt (default "bolt://username:password@localhost:7687")
```

or use env

```
GKEVENT_NEO4J=bolt:/admin:admin@moo:7687 GKEVENT_DEBUG_URL=:8888 ./event
```

## client

### http

Create a parent event

```
curl -X POST -H "Content-Type: application/json" -d '{
   "event":{
      "id":"4afe6251-9aeb-4d36-a709-5e0fbfb8b3d6",
      "type":"PARENT",
      "timestamp":"2019-10-07T08:19:20+00:00"
   }
}' http://localhost:8081/add
```

Create a child event

```
curl -X POST -H "Content-Type: application/json" -d '{ 
    "event": { 
        "id": "4afe6251-9aeb-4d36-a709-5e0fbfb8b7",
        "type": "CHILD",
        "timestamp": "2019-10-07T08:19:20+00:00",
        "attributes": [
            { 
                "name": "distance", 
                "value": "100"
            }
        ], 
        "links": [ 
            { 
                "event_id": "4afe6251-9aeb-4d36-a709-5e0fbfb8b3d6",
                "event_typ": "PARENT",
                "link_type": "CHILD_OF"
            }
         ]
    }
}' http://localhost:8081/add
```

### grpc

Build and run the grpc client to generate demo events

```
cd event/cmd/demo
go build
```

`./demo --help`

eg. 
```
neo4j start
cd cmd/event/
./event -neo4j bolt://neo4j:neo4j@localhost:7687 -grpc-addr :8082
./demo -service :8082 -messages 100
```

![demo](event/demo.png)

## Schema

See the event [proto](event/pkg/grpc/pb/event.proto) for the schema definition.

## UML

![](./doc/gkgraph.png)

Genarated with `./doc/doc.sh`

* [goplantuml](https://github.com/jfeliu007/goplantuml)
* [plantuml](https://github.com/plantuml)


