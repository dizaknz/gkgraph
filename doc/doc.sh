#! /bin/bash

which goplantuml &> /dev/null || {
    (
        echo "INFO: Installing goplantuml"
        go get github.com/jfeliu007/goplantuml/parser
        go get github.com/jfeliu007/goplantuml/cmd/goplantuml
        cd $GOPATH/src/github.com/jfeliu007/goplantuml
        go install ./...
    )
}


echo "INFO: Generating UML"
goplantuml \
    -show-aggregations \
    -show-compositions \
    -show-connection-labels \
    -show-implementations \
    -show-aliases \
    -output gkgraph.puml \
    -recursive ../ || {
    echo "ERROR: Failed to create UML"
    exit 1
}

plantuml ./gkgraph.puml || echo "ERROR: Failed to generate PNG"
