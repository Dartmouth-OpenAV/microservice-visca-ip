#!/bin/bash

# Set your environment variables before running
MICROSERVICE_URL="your.microservice.url"
DEVICE_FQDN="your.device.fqdn"

echo "Running VISCA Microservice Tests..."

# GET requests
curl -X GET "http://$MICROSERVICE_URL/$DEVICE_FQDN/power"
sleep 1

curl -X GET "http://$MICROSERVICE_URL/$DEVICE_FQDN/focus"
sleep 1

curl -X GET "http://$MICROSERVICE_URL/$DEVICE_FQDN/preset"
sleep 1

curl -X GET "http://$MICROSERVICE_URL/$DEVICE_FQDN/ptzabsolute"
sleep 1

curl -X GET "http://$MICROSERVICE_URL/$DEVICE_FQDN/autotracking"
sleep 1

# PUT requests
curl -X PUT "http://$MICROSERVICE_URL/$DEVICE_FQDN/power" -H "Content-Type: application/json" -d '"on"'
sleep 10

curl -X PUT "http://$MICROSERVICE_URL/$DEVICE_FQDN/focus" -H "Content-Type: application/json" -d '"auto"'
sleep 1

curl -X PUT "http://$MICROSERVICE_URL/$DEVICE_FQDN/preset" -H "Content-Type: application/json" -d '"0"'
sleep 1

curl -X PUT "http://$MICROSERVICE_URL/$DEVICE_FQDN/calibrate"
sleep 1

curl -X PUT "http://$MICROSERVICE_URL/$DEVICE_FQDN/ptzdrive" -H "Content-Type: application/json" -d '{ "action":"in", "zoom_speed": 1, "pan_tilt_speed": 2 }'
sleep 1

curl -X PUT "http://$MICROSERVICE_URL/$DEVICE_FQDN/ptzabsolute" -H "Content-Type: application/json" -d '{ "pan": 150, "tilt": -100, "zoom": 500 }'
sleep 1

curl -X PUT "http://$MICROSERVICE_URL/$DEVICE_FQDN/autotracking" -H "Content-Type: application/json" -d '"off"'
sleep 1

echo "Tests complete."
