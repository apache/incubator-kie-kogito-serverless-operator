#!/bin/sh

# cleanup temporary files
cleanup () {
    rm -rf target
    rm -rf src/main/resources/postgresql
    rm -rf tmp
    rm -f src/main/cekit/modules/kogito-postgres-db-migration-deps/sonataflow-db-migrator-1.0-SNAPSHOT-runner.jar
}

# Start with cleanup
cleanup

# Get Data Index/ Jobs Service DDL Files
mkdir -p tmp
# Change the variables below, as needed
DDL_VERSION=10.0.999-SNAPSHOT
DDL_FILE=kogito-ddl-10.0.999-20240806.011718-23-db-scripts.zip
DDL_URL=https://repository.apache.org/content/groups/snapshots/org/kie/kogito/kogito-ddl/$DDL_VERSION/$DDL_FILE
wget $DDL_URL
mv $DDL_FILE tmp
cd tmp
unzip $DDL_FILE
mv ./postgresql ../src/main/resources
cd ..

# Create an Uber jar
mvn package -Dquarkus.package.jar.type=uber-jar
cp target/sonataflow-db-migrator-1.0-SNAPSHOT-runner.jar src/main/cekit/modules/kogito-postgres-db-migration-deps

# Build the container image
cd src/main/cekit
cekit -v build podman

# Cleanup
cd ../../..
cleanup