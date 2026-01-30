#!/bin/bash

echo "Building binary..."
GOOS=linux GOARCH=arm GOARM=6 go build -o smartframe
echo "Copying binary..."
scp smartframe epaper.local:~/smartframe
echo "Stopping service..."
ssh epaper.local "sudo systemctl stop smartframe.service"
echo "Moving binary..."
ssh epaper.local "mv ~/smartframe ~/deploy/smartframe"
echo "Starting service..."
ssh epaper.local "sudo systemctl start smartframe.service"