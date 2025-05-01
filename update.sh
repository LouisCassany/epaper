#!bin/bash

GOOS=linux GOARCH=arm GOARM=6 go build -o smartframe

# ssh into epaper, stop the smartframe service, copy the binary to the device, start the service

scp smartframe epaper:~/smartframe
ssh epaper "sudo systemctl stop smartframe.service"
ssh epaper "mv ~/smartframe ~/deploy/smartframe"
ssh epaper "sudo systemctl start smartframe.service"