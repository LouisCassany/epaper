#!bin/bash

GOOS=linux GOARCH=arm GOARM=6 go build -o smartframe

# ssh into epaper, stop the smartframe service, copy the binary to the device, start the service

echo "Copying binary..."
scp smartframe epaper:~/smartframe
echo "Stopping service..."
ssh epaper "sudo systemctl stop smartframe.service"
echo "Moving binary..."
ssh epaper "mv ~/smartframe ~/deploy/smartframe"
echo "Starting service..."
ssh epaper "sudo systemctl start smartframe.service"