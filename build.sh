#!bin/bash

GOOS=linux GOARCH=arm GOARM=6 go build -o smartframe

# delete deploy directory if it exists
if [ -d "deploy" ]; then
    rm -rf deploy
fi

# create deploy directory
mkdir deploy

# copy static files
cp -r static deploy

# copy smartframe binary
cp smartframe deploy