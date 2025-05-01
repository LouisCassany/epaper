#!/bin/bash

echo "Building binary..."
GOOS=linux GOARCH=arm GOARM=6 go build -o smartframe
echo "Copying binary..."
scp smartframe epaper:~/smartframe
echo "Stopping service..."
ssh epaper "sudo systemctl stop smartframe.service"
echo "Moving binary..."
ssh epaper "mv ~/smartframe ~/deploy/smartframe"
echo "Starting service..."
ssh epaper "sudo systemctl start smartframe.service"