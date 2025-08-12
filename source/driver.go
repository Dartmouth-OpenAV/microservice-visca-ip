package main

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"github.com/Dartmouth-OpenAV/microservice-framework/framework"
)

// GLOBAL VARIABLES
var sequenceNumber int = 0

// HELPER FUNCTIONS
// Direct command to the camera to start tracking the sequence number from 0.
func resetSequenceNumber(socketKey string) (string, error) {
	function := "resetSequenceNumber"

	resetHexStr := "020000010000000101"

	// Not calling the convertAndSend function since that inserts a sequence number in the wrong place.
	decodedHex, err := hex.DecodeString(resetHexStr)

	if err != nil {
		errMsg := fmt.Sprintf(function + " - u6jcdh - Unable to convert hex to byte.")
		framework.AddToErrors(socketKey, errMsg)
		return errMsg, errors.New(errMsg)
	}

	framework.Log(fmt.Sprintf("Decoded Hex Sent: % x", decodedHex))

	sent := framework.WriteLineToSocket(socketKey, string(decodedHex))

	if !sent {
		errMsg := fmt.Sprintf(function + " - gt0nd2 - error sending command")
		framework.AddToErrors(socketKey, errMsg)
		return errMsg, errors.New(errMsg)
	}

	resp, errMsg, err := readAndConvert(socketKey, "SET")

	if err != nil{
		return errMsg, err
	}

	framework.Log("Response to Reset Seq Number: " + fmt.Sprintf("% x", resp[0:10]))

	sequenceNumber = 1

	return "No error", nil
}
// Converts from a hex string to a byte array, maintains the sequence number, and sends the command to the camera.
func convertAndSend(socketKey string, hexStr string) bool {
	function := "convertAndSend"

	//Use this function to insert and maintain the sequence number.
	hexStr = maintainSequenceNumber(hexStr)

	//DecodeString returns the bytes represented by the hexadecimal string (hexStr).
	decodedHex, err := hex.DecodeString(hexStr)

	if err != nil {
		errMsg := fmt.Sprintf(function + " DFK893 - Unable to convert hex to byte.")
		framework.AddToErrors(socketKey, errMsg)
	}

	framework.Log(fmt.Sprintf("Decoded Hex Sent: % x", decodedHex))

	sent := framework.WriteLineToSocket(socketKey, string(decodedHex))

	if !sent {
		errMsg := fmt.Sprintf(function + " - h3okxu3 - error sending command")
		framework.AddToErrors(socketKey, errMsg)
	}

	return sent
}
// Reads a response from the camera, removes extra 0s past the terminator, and checks for VISCA error codes.
func readAndConvert(socketKey string, funcType string) ([]byte, string, error) {
	function := "readAndConvert"
	for attempt := 0; attempt < 3; attempt++ {
		resp := framework.ReadLineFromSocket(socketKey)

		arr := make([]byte, 0)
		
		// Normally, there is an acknowledgement response or error message.
		// AVer Set Power was returning a blank response.
		if resp == "" {
			errMsg := function + " - k3kxlpo - Response was blank."
			framework.AddToErrors(socketKey, errMsg)
			return arr, errMsg, errors.New(errMsg)
		}

		decodedByteArray, err := hex.DecodeString(fmt.Sprintf("%x", resp))

		if err != nil {
			errMsg := function + " - 5kdlkc - Unable to convert hex to byte."
			framework.AddToErrors(socketKey, errMsg)
			return arr, errMsg, errors.New(errMsg)
		}
		// Responses include several zeros after the terminator.
		// This excludes everything after the terminator.
		for index := range decodedByteArray {
			arr = append(arr, decodedByteArray[index])
			if decodedByteArray[index] == 0xFF {
				if index > 3 && index < 8{ 
					continue
				} else {
					break
				}
			}
		}
		// Check for VISCA error codes
		errMsg, err := determineErrorCode(socketKey, arr)
		if err != nil{
			return arr, errMsg, err
		}

		//Converts the response array to a string
		convertedArr := fmt.Sprintf("%x", arr)
		framework.Log(function + " converted array: " + convertedArr)

		//Skip the sequence number check for SETs. Not necessary for ack or completion responses.
		//Only holds up other commands while waiting for a completion response.
		if funcType == "SET"{
			return arr, "No error", nil
		}

		// Find and convert the sequence number in the response
		responseSequenceNumber, err := strconv.ParseInt(convertedArr[8:16],16,32)
		framework.Log("Sequence numbers below")
		framework.Log(fmt.Sprint(responseSequenceNumber))
		framework.Log(fmt.Sprint(sequenceNumber))
		if err != nil {
			errMsg := function + " - 243rjdf Error with ParseInt."
			framework.AddToErrors(socketKey, errMsg)
			return arr, errMsg, errors.New(errMsg)
		}
		// Make sure the sequence number in the response matches the one sent in the command
		if responseSequenceNumber == int64(sequenceNumber){
			return arr, "No error", nil
		}
		// If the sequence numbers don't match, the loop will try to read again.
	}
	errMsg := function + "34dm4j - Read failed after 3 attempts"
	return nil, errMsg, errors.New(errMsg)
}
// Checks if a camera response matches the format for a VISCA error.
func determineErrorCode(socketKey string, response []byte) (string, error) {
	camAddress := fmt.Sprintf("%x", response[len(response)-4])
	errorID := fmt.Sprintf("%x", response[len(response)-3])
	errorType := response[len(response)-2]

	// Error codes defined by the VISCA Protocol
	if camAddress == "90" && errorID == "60" {
		switch errorType {
		case 1:
			errMsg := "VISCA Error - Message Length Error"
			framework.AddToErrors(socketKey, errMsg)
			return errMsg, errors.New(errMsg)
		case 2:
			errMsg := "VISCA Error - Syntax Error"
			framework.AddToErrors(socketKey, errMsg)
			return errMsg, errors.New(errMsg)
		case 3:
			errMsg := "VISCA Error - Command Buffer Full"
			framework.AddToErrors(socketKey, errMsg)
			return errMsg, errors.New(errMsg)
		case 4:
			errMsg := "VISCA Error - Command Canceled"
			framework.AddToErrors(socketKey, errMsg)
			return errMsg, errors.New(errMsg)
		case 5:
			errMsg := "VISCA Error - No Socket (To be canceled)"
			framework.AddToErrors(socketKey, errMsg)
			return errMsg, errors.New(errMsg)
		case 41:
			errMsg := "VISCA Error - Command Not Executable"
			framework.AddToErrors(socketKey, errMsg)
			return errMsg, errors.New(errMsg)
		default:
			errMsg := "VISCA Error - Unknown. Matched error formatting, but no ID"
			framework.AddToErrors(socketKey, errMsg)
			return errMsg, errors.New(errMsg)
		}
	}

	return "Not an error", nil
}
// Increases the sequence number that is used to match a command with the camera's response.
func maintainSequenceNumber(cmdString string) string {
	sequenceNumber += 1

	sequenceHex := fmt.Sprintf("%08x", sequenceNumber)
	hexStr1 := cmdString[:8]
	hexStr2 := cmdString[16:]

	cmdString = hexStr1 + sequenceHex + hexStr2
	framework.Log("cmdString: " + cmdString)
	
	return cmdString
}
// Converts an int to a hex string of length 8 with padded 0s. For example, 100 will be converted to 00000604
func convertIntsToPaddedBytes(cmdInt int) string {

	// Check if the value is negative
	isNegative := cmdInt < 0

	// If the value is negative, calculate its signed 2's complement
	var hexInt uint
	if isNegative {
		// Calculate 2's complement for negative numbers
		hexInt = uint(1<<16) + uint(cmdInt)
	} else {
		// For positive numbers, convert directly to unsigned integer
		hexInt = uint(cmdInt)
	}

	// Convert the unsigned integer to a hexadecimal string
	cmdHex := fmt.Sprintf("%04x", hexInt)

	//add padded 0s
	byteString := ""
	for idx := range(cmdHex){
		byteString = byteString + "0" + string(cmdHex[idx])
	}

	return byteString
}
// Converts a hex string with padded 0s to an int. For example, 00000604 will be converted to 100.
func convertPaddedBytesToInts(respString string) (int64, error) {
	//Ignore the padded 0s
	respHexString := ""
	for index := range(respString){
		if index%2 != 0{
			respHexString = respHexString + string(respString[index])
		}
	}

	// Parse hexadecimal string to an unsigned integer
	hexInt, err := strconv.ParseUint(respHexString, 16, 32)
	if err != nil {
		fmt.Println("jedj4nd - Error parsing hexadecimal string:", err)
		return 0, err
	}

	// Check if MSB is set (indicating a negative number)
	msbSet := hexInt&(1<<15) != 0
	var finalValue int64

	// If the MSB is set (indicating a negative number), convert it to its decimal equivalent
	if msbSet {
		// Calculate the signed 2's complement
		decimalVal := int64(hexInt) - (1 << 16)
		// fmt.Printf("Hexadecimal %s in signed decimal (2's complement): %d\n", respHexString, decimalVal)
		finalValue = decimalVal
	} else {
		// If positive, simply print the decimal value
		// fmt.Printf("Hexadecimal %s in signed decimal: %d\n", respHexString, hexInt)
		finalValue = int64(hexInt)
	}

	return finalValue, nil
}
// Validating that zoom and pan/tilt speeds are within the right range.
func validateSpeedValues(panTiltSpeedValue float64, zoomSpeedValue float64) (string, string, error) {
	zoomSpeedString := ""
	panTiltSpeedString := ""
	
	if (panTiltSpeedValue < 1 || panTiltSpeedValue > 14 ) {
		framework.Log("Pan/Tilt speed out of acceptable range of 1-14. Setting to default of 5")
		panTiltSpeedString = "05"
	}else {
		panTiltSpeedString = strconv.FormatFloat(panTiltSpeedValue, 'f', -1, 32)
		if panTiltSpeedValue < 10 {
			panTiltSpeedString = "0" + panTiltSpeedString
		}
	}

	if (zoomSpeedValue < 0 || zoomSpeedValue > 7 ) {
		framework.Log("Zoom speed out of acceptable range of 0-7. Setting to default of 2")
		zoomSpeedString = "2"
	}else {
		zoomSpeedString = strconv.FormatFloat(zoomSpeedValue, 'f', -1, 32)
	}

	return panTiltSpeedString, zoomSpeedString, nil
}

//MAIN FUNCTIONS

// GET Functions
// Clears the command buffer in the camera.
func clearInterface(socketKey string) (string, error){
	function:= "clearInterface"
	
	clearHexStr := "010000050000000081010001ff"

	sent := convertAndSend(socketKey, clearHexStr)

	if !sent {
		errMsg := fmt.Sprintf(function + " - gt0nd2 - error sending command")
		framework.AddToErrors(socketKey, errMsg)
		return errMsg, errors.New(errMsg)
	}

	resp, errMsg, err := readAndConvert(socketKey, "SET")

	if err != nil{
		return errMsg, err
	}

	framework.Log(function + " - Decoded Response: "+ fmt.Sprintf("% x", resp))

	return fmt.Sprintf("% x", resp), nil
}

func getPower(socketKey string) (string, error) {
	function := "getPower"

	value := `"unknown"`
	err := error(nil)
	maxRetries := 2
	for maxRetries > 0 {
		value, err = getPowerDo(socketKey)
		if value == `"unknown"` { // Something went wrong - perhaps try again
			framework.Log(function + " - fq3sdvc - retrying power operation")
			maxRetries--
			time.Sleep(1 * time.Second)
			if maxRetries == 0 {
				errMsg := fmt.Sprintf(function + "f839dk4 - max retries reached")
				framework.AddToErrors(socketKey, errMsg)
			}
		} else { // Succeeded
			maxRetries = 0
		}
	}

	return value, err
}
// Gets the camera's current power state. Returns "On" or "Off"
func getPowerDo(socketKey string) (string, error) {
	function := "getPowerDo"

	powerHexStr := "011000050000000081090400FF"

	sent := convertAndSend(socketKey, powerHexStr)

	if !sent {
		errMsg := fmt.Sprintf(function + " - fk4kxy7 - error sending command")
		framework.AddToErrors(socketKey, errMsg)
		return errMsg, errors.New(errMsg)
	}

	respArr, errMsg, err := readAndConvert(socketKey, "GET")

	if err != nil{
		return errMsg, err
	}

	value := `"unknown"`
	resp := respArr[len(respArr)-2]
	framework.Log(function + " - Decoded Response: "+ fmt.Sprintf("% x", respArr))

	if len(respArr) > 100 {
		errMsg := "length of respArr is greater than 100. Likely an error with camera. Resetting sequence number"
		framework.AddToErrors(socketKey, errMsg)
		resetSequenceNumber(socketKey)
		clearInterface(socketKey)
		return `"unknown"`, errors.New(errMsg)
	}

	if resp == 2 {
		value = `"on"`
	} else if resp == 3 {
		value = `"off"`
	} else {
		errMsg := fmt.Sprintf(function + socketKey+" - can't interpret response")
		framework.AddToErrors(socketKey, errMsg)
		return errMsg, errors.New(errMsg)
	}
	framework.Log(function + " - value: " + fmt.Sprintf(value))

	// If we got here, the response was good, so successful return with the state indication
	return fmt.Sprintf(value), nil
}

func getPresetNumber(socketKey string) (string, error) {
	function := "getPresetNumber"

	value := `"unknown"`
	err := error(nil)
	maxRetries := 2
	for maxRetries > 0 {
		value, err = getPresetNumberDo(socketKey)
		if value == `"unknown"` { // Something went wrong - perhaps try again
			framework.Log(function + " - sdf09nd - retrying preset operation")
			maxRetries--
			time.Sleep(1 * time.Second)
			if maxRetries == 0 {
				errMsg := fmt.Sprintf(function + "f4fk5n3 - max retries reached")
				framework.AddToErrors(socketKey, errMsg)
			}
		} else { // Succeeded
			maxRetries = 0
		}
	}

	return value, err
}
// Gets the preset number that the camera was last set to. Returns the preset number.
func getPresetNumberDo(socketKey string) (string, error) {
	function := "getPresetNumberDo"

	getPresetHexStr := "01100005000000008109043FFF"

	sent := convertAndSend(socketKey, getPresetHexStr)

	if !sent {
		errMsg := fmt.Sprintf(function + " - io6k3d - error sending command")
		framework.AddToErrors(socketKey, errMsg)
		return errMsg, errors.New(errMsg)
	}

	respArr, errMsg, err := readAndConvert(socketKey, "GET")
	framework.Log("respARR: " + fmt.Sprintf("%x", respArr))

	if err != nil{
		return errMsg, err
	}

	value := `"unknown"`

	if len(respArr) > 100 {
		errMsg := "length of respArr is greater than 100. Likely an error with camera. Resetting sequence number"
		framework.AddToErrors(socketKey, errMsg)
		resetSequenceNumber(socketKey)
		clearInterface(socketKey)
		return `"unknown"`, errors.New(errMsg)
	}

	resp := respArr[len(respArr)-2]

	if resp >= 0 && resp <= 16 {
		value = fmt.Sprint(resp)
	} else {
		framework.AddToErrors(socketKey, socketKey+" - can't interpret response")
	}

	framework.Log(function + " - value: " + value)

	// If we got here, the response was good, so successful return with the state indication
	return `"` + value + `"`, nil
}

func getFocus(socketKey string) (string, error) {
	function := "getFocus"

	value := `"unknown"`
	err := error(nil)
	maxRetries := 2
	for maxRetries > 0 {
		value, err = getFocusDo(socketKey)
		if value == `"unknown"` { // Something went wrong - perhaps try again
			framework.Log(function + " - fq3sdvc retrying focus operation")
			maxRetries--
			time.Sleep(1 * time.Second)
			if maxRetries == 0 {
				errMsg := fmt.Sprintf(function + "fds3df3 - max retries reached")
				framework.AddToErrors(socketKey, errMsg)
			}
		} else { // Succeeded
			maxRetries = 0
		}
	}

	return value, err
}
// Gets the camera's current focus mode. Returns "auto" or "manual".
func getFocusDo(socketKey string) (string, error) {
	function := "getFocusDo"

	focusHexStr := "011000050000000081090438FF"

	sent := convertAndSend(socketKey, focusHexStr)

	if !sent {
		errMsg := fmt.Sprintf(function + " - 44mk5hd - error sending command")
		framework.AddToErrors(socketKey, errMsg)
		return errMsg, errors.New(errMsg)
	}

	respArr, errMsg, err := readAndConvert(socketKey, "GET")
	
	if err != nil{
		return errMsg, err
	}

	framework.Log(function + " - Decoded Response: " + fmt.Sprintf("% x", respArr))

	value := `"unknown"`

	if len(respArr) > 100 {
		errMsg := "length of respArr is greater than 100. Likely an error with camera. Resetting sequence number"
		framework.AddToErrors(socketKey, errMsg)
		resetSequenceNumber(socketKey)
		clearInterface(socketKey)
		return `"unknown"`, errors.New(errMsg)
	}

	resp := respArr[len(respArr)-2]

	if resp == 2 {
		value = `"auto"`
	} else if resp == 3 {
		value = `"manual"`
	} else {
		framework.AddToErrors(socketKey, socketKey+" - can't interpret response")
	}

	framework.Log(function + " - value: " + fmt.Sprintf(value))

	// If we got here, the response was good, so successful return with the state indication
	return fmt.Sprintf(value), nil
}
	
func getPTZAbsolute(socketKey string) (string, error) {
	function := "getPTZAbsolute"

	value := `"unknown"`
	err := error(nil)
	maxRetries := 2
	for maxRetries > 0 {
		value, err = getPTZAbsoluteDo(socketKey)
		if value == `"unknown"` { // Something went wrong - perhaps try again
			framework.Log(function + " - df4gd3 - retrying pantiltzoom operation")
			maxRetries--
			time.Sleep(1 * time.Second)
		} else { // Succeeded
			maxRetries = 0
		}
	}

	return value, err
}
// Gets the camera's current pan/tilt coordinates. Returns the values as a string "{"pan": value, "tilt": value, "zoom": value}"
func getPTZAbsoluteDo(socketKey string) (string, error) {
	function := "getPTZAbsoluteDo"

	// GET Pan/Tilt Coordinates
	getPanTiltHexStr := "011000050000000081090612ff"
	sent := convertAndSend(socketKey, getPanTiltHexStr)

	if !sent {
		errMsg := fmt.Sprintf(function + " - fg4dcs - error sending command")
		framework.AddToErrors(socketKey, errMsg)
		return `"unknown"`, errors.New(errMsg)
	}

	respArr, errMsg, err := readAndConvert(socketKey, "GET")
	
	if err != nil{
		return errMsg, err
	}
	framework.Log("Response for PT: " + fmt.Sprintf("%x", respArr))
	if len(respArr) > 100 {
		errMsg := "length of respArr is greater than 100. Likely an error with camera. Resetting sequence number"
		framework.AddToErrors(socketKey, errMsg)
		resetSequenceNumber(socketKey)
		clearInterface(socketKey)
		return `"unknown"`, errors.New(errMsg)
	}

	panValue := respArr[(len(respArr) - 9):(len(respArr) - 5)]
	tiltValue := respArr[(len(respArr) - 5):(len(respArr) - 1)]

	panValueString := fmt.Sprintf("%x", panValue)
	tiltValueString := fmt.Sprintf("%x", tiltValue)

	panDecimal, err := convertPaddedBytesToInts(panValueString)
	if err != nil{
		return `"unknown"`, err
	}

	tiltDecimal, err := convertPaddedBytesToInts(tiltValueString)
	if err != nil{
		return `"unknown"`, err
	}

	// GET Zoom Value
	getZoomHexStr := "011000050000000081090447ff"
	sent = convertAndSend(socketKey, getZoomHexStr)

	if !sent {
		errMsg := fmt.Sprintf(function + " - k4nd3w - error sending command")
		framework.AddToErrors(socketKey, errMsg)
		return `"unknown"`, errors.New(errMsg)
	}

	respArr, errMsg, err = readAndConvert(socketKey, "GET")
	framework.Log("Response for Zoom:" + fmt.Sprintf("%x",respArr))
	if err != nil{
		return errMsg, err
	}
	if len(respArr) > 100 {
		errMsg := "length of respArr is greater than 100. Likely an error with camera. Resetting sequence number"
		framework.AddToErrors(socketKey, errMsg)
		resetSequenceNumber(socketKey)
		clearInterface(socketKey)
		return `"unknown"`, errors.New(errMsg)
	}
	zoomValue := respArr[(len(respArr) - 5):(len(respArr) - 1)]
	zoomValueString := fmt.Sprintf("%x", zoomValue)

	zoomDecimal, err := convertPaddedBytesToInts(zoomValueString)

	if err != nil {
		return `"unknown"`, err
	}

	ptzCoordinates := "{\"pan\":" + fmt.Sprint(panDecimal) + ",\"tilt\":" + fmt.Sprint(tiltDecimal) + ",\"zoom\":" + fmt.Sprint(zoomDecimal) +"}"

	// If we got here, the response was good, so successful return with the state indication
	return ptzCoordinates, nil
}

func getAutoTracking(socketKey string) (string, error) {
	function := "getAutoTracking"

	value := `"unknown"`
	err := error(nil)
	maxRetries := 2
	for maxRetries > 0 {
		value, err = getAutoTrackingDo(socketKey)
		if value == `"unknown"` { // Something went wrong - perhaps try again
			framework.Log(function + " - fq3sdvc - retrying autotracking operation")
			maxRetries--
			time.Sleep(1 * time.Second)
			if maxRetries == 0 {
				errMsg := fmt.Sprintf(function + "f839dk4 - max retries reached")
				framework.AddToErrors(socketKey, errMsg)
			}
		} else { // Succeeded
			maxRetries = 0
		}
	}

	return value, err
}
// Gets the camera's autotracking state. Returns "On" or "Off"
func getAutoTrackingDo(socketKey string) (string, error) {
	function := "getAutoTrackingDo"

	autoTrackingHexStr := "011000060000000081097E043AFF"

	sent := convertAndSend(socketKey, autoTrackingHexStr)

	if !sent {
		errMsg := fmt.Sprintf(function + " - fk4kxy7 - error sending command")
		framework.AddToErrors(socketKey, errMsg)
		return errMsg, errors.New(errMsg)
	}

	respArr, errMsg, err := readAndConvert(socketKey, "GET")

	if err != nil{
		return errMsg, err
	}

	value := `"unknown"`
	resp := respArr[len(respArr)-2]
	framework.Log(function + " - Decoded Response: "+ fmt.Sprintf("% x", respArr))

	if len(respArr) > 100 {
		errMsg := "length of respArr is greater than 100. Likely an error with camera. Resetting sequence number"
		framework.AddToErrors(socketKey, errMsg)
		resetSequenceNumber(socketKey)
		clearInterface(socketKey)
		return `"unknown"`, errors.New(errMsg)
	}

	if resp == 1 {
		value = `"on"`
	} else if resp == 0 {
		value = `"off"`
	} else {
		errMsg := fmt.Sprintf(function + socketKey+" - can't interpret response")
		framework.AddToErrors(socketKey, errMsg)
		return errMsg, errors.New(errMsg)
	}
	framework.Log(function + " - value: " + fmt.Sprintf(value))

	// If we got here, the response was good, so successful return with the state indication
	return fmt.Sprintf(value), nil
}

//SET Functions

func setPower(socketKey string, state string) (string, error) {
	function := "setPower"

	value := "notok"
	err := error(nil)
	maxRetries := 2
	for maxRetries > 0 {
		value, err = setPowerDo(socketKey, state)
		if value != "ok" { // Something went wrong - perhaps try again
			framework.Log(function + " - fq3sdvc - retrying power operation")
			maxRetries--
			time.Sleep(1 * time.Second)
			if maxRetries == 0 {
				errMsg := fmt.Sprintf(function + " - fds3nf3 - max retries reached")
				framework.AddToErrors(socketKey, errMsg)
			}
		} else { // Succeeded
			maxRetries = 0
		}
	}

	return value, err
}
// Sets power to on, off, or reboots.
func setPowerDo(socketKey string, state string) (string, error) {
	function := "setPowerDo"

	pwrValue := ""

	if state == `"on"` {
		pwrValue = "2"
	} else if state == `"off"` {
		pwrValue = "3"
	} else if state == `"reboot"` {
		pwrValue = "0"
	} else {
		errMsg := fmt.Sprintf(function + " - unrecognized state value: " + state)
		framework.AddToErrors(socketKey, errMsg)
		return state, errors.New(errMsg)
	}

	powerHexString := "0100000600000000810104000" + pwrValue + "FF"

	sent := convertAndSend(socketKey, powerHexString)

	if !sent {
		errMsg := fmt.Sprintf(function + " - dfji4k - error sending command")
		framework.AddToErrors(socketKey, errMsg)
		return errMsg, errors.New(errMsg)
	}

	respArr, errMsg, err := readAndConvert(socketKey, "SET")

	framework.Log(function + " - Decoded response: " + fmt.Sprintf("% x", respArr))

	if err != nil{
		return errMsg, err
	}

	if len(respArr) > 100 {
		errMsg := "length of respArr is greater than 100. Likely an error with camera. Resetting sequence number"
		framework.AddToErrors(socketKey, errMsg)
		resetSequenceNumber(socketKey)
		clearInterface(socketKey)
		return "notok", errors.New(errMsg)
	}

	// If we got here, the response was good, so successful return with the state indication
	return "ok", nil
}

func setPresetRecall(socketKey string, state string) (string, error) {
	function := "setPresetRecall"

	value := "notok"
	err := error(nil)
	maxRetries := 2
	for maxRetries > 0 {
		value, err = setPresetRecallDo(socketKey, state)
		if value != "ok" { // Something went wrong - perhaps try again
			framework.Log(function + " - fq5dhs - retrying preset operation")
			maxRetries--
			time.Sleep(1 * time.Second)
			if maxRetries == 0 {
				errMsg := fmt.Sprintf(function + " - 03kfl4d - max retries reached")
				framework.AddToErrors(socketKey, errMsg)
			}
		} else { // Succeeded
			maxRetries = 0
		}
	}

	return value, err
}
// Moves the camera to a specifc preset number.
func setPresetRecallDo(socketKey string, state string) (string, error) {
	function := "setPresetRecallDo"
	state = strings.Trim(state, "\"")

	presetHexStr := "01000007000000008101043F020" + string(state) + "FF"
	
	sent := convertAndSend(socketKey, presetHexStr)

	if !sent {
		errMsg := fmt.Sprintf(function + " - djj3d2 - error sending command")
		framework.AddToErrors(socketKey, errMsg)
		return errMsg, errors.New(errMsg)
	}

	respArr, errMsg, err := readAndConvert(socketKey, "SET")
	framework.Log(function + " - Decoded response: " + fmt.Sprintf("% x", respArr))
	
	if err != nil{
		return errMsg, err
	}

	if len(respArr) > 100 {
		errMsg := "length of respArr is greater than 100. Likely an error with camera. Resetting sequence number"
		framework.AddToErrors(socketKey, errMsg)
		resetSequenceNumber(socketKey)
		clearInterface(socketKey)
		return "notok", errors.New(errMsg)
	}

	return "ok", nil
}

func setFocus(socketKey string, state string) (string, error) {
	function := "setFocus"

	value := "notok"
	err := error(nil)
	maxRetries := 2
	for maxRetries > 0 {
		value, err = setFocusDo(socketKey, state)
		if value != "ok" { // Something went wrong - perhaps try again
			framework.Log(function + " - fq6d9s retrying focus operation")
			maxRetries--
			time.Sleep(1 * time.Second)
			if maxRetries == 0 {
				errMsg := fmt.Sprintf(function + " - fd4k6f3 - max retries reached")
				framework.AddToErrors(socketKey, errMsg)
			}
		} else { // Succeeded
			maxRetries = 0
		}
	}

	return value, err
}
// Sets the camera's focus to auto, manual, or triggers the one-push autofocus.
func setFocusDo(socketKey string, state string) (string, error) {
	function := "setFocusDo"

	focusHexStr := ""

	if state == `"auto"` {
		focusHexStr = "01000006000000008101043802FF"
	} else if state == `"manual"` {
		focusHexStr = "01000006000000008101043803FF"
	} else if state == `"trigger"` {
		focusHexStr = "01000006000000008101041801FF"
	} else {
		framework.Log(function + "- jknjy5 - Unrecognized focus mode")
	}

	sent := convertAndSend(socketKey, focusHexStr)

	if !sent {
		errMsg := fmt.Sprintf(function + " - j3dnx3 - error sending command")
		framework.AddToErrors(socketKey, errMsg)
		return errMsg, errors.New(errMsg)
	}

	respArr, errMsg, err := readAndConvert(socketKey, "SET")
	framework.Log(function + " - Decoded response: " + fmt.Sprintf("% x", respArr))
	
	if err != nil{
		return errMsg, err
	}

	if len(respArr) > 100 {
		errMsg := "length of respArr is greater than 100. Likely an error with camera. Resetting sequence number"
		framework.AddToErrors(socketKey, errMsg)
		resetSequenceNumber(socketKey)
		clearInterface(socketKey)
		return "notok", errors.New(errMsg)
	}

	return "ok", nil
}

func setCalibrate(socketKey string) (string, error) {
	function := "setCalibrate"

	value := "notok"
	err := error(nil)
	maxRetries := 2
	for maxRetries > 0 {
		value, err = setCalibrateDo(socketKey)
		if value != "ok" { // Something went wrong - perhaps try again
			framework.Log(function + " - fs9d4j - retrying calibrate operation")
			maxRetries--
			time.Sleep(1 * time.Second)
			if maxRetries == 0 {
				errMsg := fmt.Sprintf(function + " - f93mwrj - max retries reached")
				framework.AddToErrors(socketKey, errMsg)
			}
		} else { // Succeeded
			maxRetries = 0
		}
	}

	return value, err
}
// Calibrates the camera.
func setCalibrateDo(socketKey string) (string, error) {
	function := "setCalibrateDo"
	calibrateHexStr := "010000050000000081010605FF"

	sent := convertAndSend(socketKey, calibrateHexStr)

	if !sent {
		errMsg := fmt.Sprintf(function + " - lxl2lx - error sending command")
		framework.AddToErrors(socketKey, errMsg)
		return errMsg, errors.New(errMsg)
	}

	respArr, errMsg, err := readAndConvert(socketKey, "SET")
	framework.Log(function + " - Decoded response: " + fmt.Sprintf("% x", respArr))
	
	if err != nil{
		return errMsg, err
	}

	if len(respArr) > 100 {
		errMsg := "length of respArr is greater than 100. Likely an error with camera. Resetting sequence number"
		framework.AddToErrors(socketKey, errMsg)
		resetSequenceNumber(socketKey)
		clearInterface(socketKey)
		return "notok", errors.New(errMsg)
	}

	return "ok", nil
}

func setPTZDrive(socketKey string, action string) (string, error) {
	function := "setPTZDrive"

	value := "notok"
	err := error(nil)
	maxRetries := 2
	for maxRetries > 0 {
		value, err = setPTZDriveDo(socketKey, action)
		if value != "ok" { // Something went wrong - perhaps try again
			framework.Log(function + " - 2nmdk4 - retrying PTZDrive operation")
			maxRetries--
			time.Sleep(1 * time.Second)
			if maxRetries == 0 {
				errMsg := fmt.Sprintf(function + " - 4kdnmi4 - max retries reached")
				framework.AddToErrors(socketKey, errMsg)
			}
		} else { // Succeeded
			maxRetries = 0
		}
	}

	return value, err
}
// Moves the camera left, right, up, down, or zooms in and out until a stop command is sent.
// Pan_Tilt_Speed must be between 1 and 14
// Zoom_Speed must be between 0 and 7
func setPTZDriveDo(socketKey string, action string) (string, error) {
	function := "setPTZDriveDo"
	var responseMap map[string]interface{}
	var zoomSpeedValue float64
	var panTiltSpeedValue float64
	actionValue := ""

	err := json.Unmarshal([]byte(action), &responseMap)
		if err != nil {
			errMsg := function +  " - k4kl3kx - Error umarshaling action"
			framework.AddToErrors(socketKey, errMsg)
			return errMsg, errors.New(errMsg)
		}
	_, actionPresent := responseMap["action"]
	_, zoomSpeedPresent := responseMap["zoom_speed"]
	_, panTiltSpeedPresent := responseMap["pan_tilt_speed"]
	if actionPresent {
		actionValue = responseMap["action"].(string)
	} else {
		errMsg := function +  " - 6kdm38 - Did not pass an action in the PUT command"
		framework.AddToErrors(socketKey, errMsg)
		return errMsg, errors.New(errMsg)
	}
	// If speeds aren't specified, use a default
	if zoomSpeedPresent {
		zoomSpeedValue = responseMap["zoom_speed"].(float64)
	} else {
		zoomSpeedValue = 2
	}
	if panTiltSpeedPresent {
		panTiltSpeedValue = responseMap["pan_tilt_speed"].(float64)
	} else {
		panTiltSpeedValue = 5
	}
	// Validating that the speeds are in a certain range
	panTiltSpeedString, zoomSpeedString, err := validateSpeedValues(panTiltSpeedValue, zoomSpeedValue)
	if err != nil {
		framework.AddToErrors(socketKey, err.Error())
		return err.Error(), err
	}
	framework.Log(function)
	framework.Log("Action is: " + actionValue)
	framework.Log("PanTilt Speed is: " + panTiltSpeedString)
	framework.Log("Zoom Speed is: " + zoomSpeedString)

	// Duplicating the speed value for pan and tilt speed. Ex: 0505
	ptSpeeds := panTiltSpeedString + panTiltSpeedString

	var pantiltHexStr, zoomHexStr string
	switch actionValue {
		case "up":
			pantiltHexStr = "010000090000000081010601" + ptSpeeds + "0301" + "FF"
		case "down":
			pantiltHexStr = "010000090000000081010601" + ptSpeeds + "0302" + "FF"
		case "left":
			pantiltHexStr = "010000090000000081010601" + ptSpeeds + "0103" + "FF"
		case "right":
			pantiltHexStr = "010000090000000081010601" + ptSpeeds + "0203" + "FF"
		case "pan stop":
			pantiltHexStr = "0100000900000000810106010505" + "0303" + "FF"
		case "in":
			zoomHexStr = "010000060000000081010407" + "2" + zoomSpeedString + "FF"
		case "out":
			zoomHexStr = "010000060000000081010407" + "3" + zoomSpeedString + "FF"
		case "zoom stop":
			zoomHexStr = "010000060000000081010407" + "00" + "FF"
		default:
			errMsg := function +  " - mxb4n- Did not pass a valid action"
			framework.AddToErrors(socketKey, errMsg)
			return errMsg, errors.New(errMsg)
	}

	if pantiltHexStr != "" {
		
		sent := convertAndSend(socketKey, pantiltHexStr)

		if !sent {
			errMsg := fmt.Sprintf(function + " - qx4uxn - error sending pan/tilt drive command")
			framework.AddToErrors(socketKey, errMsg)
			return errMsg, errors.New(errMsg)
		}

		respArr, errMsg, err := readAndConvert(socketKey, "SET")
		framework.Log(function + " - Decoded pan/tilt drive response: " + fmt.Sprintf("% x", respArr))
		
		if err != nil{
			return errMsg, err
		}

		if len(respArr) > 100 {
			errMsg := "length of respArr is greater than 100. Likely an error with camera. Resetting sequence number"
			framework.AddToErrors(socketKey, errMsg)
			resetSequenceNumber(socketKey)
			clearInterface(socketKey)
			return "notok", errors.New(errMsg)
		}
	}
	if zoomHexStr != "" {

		sent := convertAndSend(socketKey, zoomHexStr)

		if !sent {
			errMsg := fmt.Sprintf(function + " - p2kxjh - error sending zoom drive command")
			framework.AddToErrors(socketKey, errMsg)
			return errMsg, errors.New(errMsg)
		}

		respArr, errMsg, err := readAndConvert(socketKey, "SET")
		framework.Log(function + " - Decoded zoom drive response: " + fmt.Sprintf("% x", respArr))
		
		if err != nil{
			return errMsg, err
		}

		if len(respArr) > 100 {
			errMsg := "length of respArr is greater than 100. Likely an error with camera. Resetting sequence number"
			framework.AddToErrors(socketKey, errMsg)
			resetSequenceNumber(socketKey)
			clearInterface(socketKey)
			return "notok", errors.New(errMsg)
		}
	}

	return "ok", nil
}

func setPTZAbsolute(socketKey string,coordinates string) (string, error) {
	function := "setPTZAbsolute"

	value := "notok"
	err := error(nil)
	maxRetries := 2
	for maxRetries > 0 {
		value, err = setPTZAbsoluteDo(socketKey, coordinates)
		if value != "ok" { // Something went wrong - perhaps try again
			framework.Log(function + " - 2nmdk4 - retrying PTZAbsolute operation")
			maxRetries--
			time.Sleep(1 * time.Second)
			if maxRetries == 0 {
				errMsg := fmt.Sprintf(function + " - 4kdnmi4 - max retries reached")
				framework.AddToErrors(socketKey, errMsg)
			}
		} else { // Succeeded
			maxRetries = 0
		}
	}

	return value, err
}
// Moves the camera to specific coordinates for pan, tilt, and zoom. Ex: {"pan": value, "tilt": value, "zoom": value}
func setPTZAbsoluteDo(socketKey string, coordinates string) (string, error) {
	function := "setPTZAbsoluteDo"
	responseMap := map[string]int{}

	if coordinates != "null" {
		err := json.Unmarshal([]byte(coordinates), &responseMap)
		if err != nil {
			errMsg := function +  " - i53jdx - Error umarshaling coordinates"
			framework.AddToErrors(socketKey, errMsg)
			return errMsg, errors.New(errMsg)
		}
		var panCoordinate, tiltCoordinate, zoomCoordinate int
		var panHex, tiltHex, zoomHex string
		_, panPresent := responseMap["pan"]
		_, tiltPresent := responseMap["tilt"]
		_, zoomPresent := responseMap["zoom"]

		if panPresent && tiltPresent {
			panCoordinate = responseMap["pan"]
			panHex = convertIntsToPaddedBytes(panCoordinate)

			tiltCoordinate = responseMap["tilt"]
			tiltHex = convertIntsToPaddedBytes(tiltCoordinate)

			pantiltHexStr := "0100000F00000000810106021814" + panHex + tiltHex + "FF"

			sent := convertAndSend(socketKey, pantiltHexStr)

			if !sent {
				errMsg := fmt.Sprintf(function + " - 3mdxm2 - error sending pan/tilt command")
				framework.AddToErrors(socketKey, errMsg)
				return errMsg, errors.New(errMsg)
			}

			respArr, errMsg, err := readAndConvert(socketKey, "SET")
			framework.Log(function + " - Decoded pan/tilt response: " + fmt.Sprintf("% x", respArr))
			
			if err != nil{
				return errMsg, err
			}
			if len(respArr) > 100 {
				errMsg := "length of respArr is greater than 100. Likely an error with camera. Resetting sequence number"
				framework.AddToErrors(socketKey, errMsg)
				resetSequenceNumber(socketKey)
				clearInterface(socketKey)
				return "notok", errors.New(errMsg)
			}
		}else{
			framework.Log("Pan/Tilt were not included in the PUT command")
		}
		if zoomPresent {
			zoomCoordinate = responseMap["zoom"]
			zoomHex = convertIntsToPaddedBytes(zoomCoordinate)
			zoomHexStr := "010000090000000081010447" + zoomHex + "FF"
			sent := convertAndSend(socketKey, zoomHexStr)

			if !sent {
				errMsg := fmt.Sprintf(function + " - m3mx6h - error sending zoom command")
				framework.AddToErrors(socketKey, errMsg)
				return errMsg, errors.New(errMsg)
			}

			respArr, errMsg, err := readAndConvert(socketKey, "SET")
			framework.Log(function + " - Decoded zoom response: " + fmt.Sprintf("% x", respArr))
			
			if err != nil{
				return errMsg, err
			}
			if len(respArr) > 100 {
				errMsg := "length of respArr is greater than 100. Likely an error with camera. Resetting sequence number"
				framework.AddToErrors(socketKey, errMsg)
				resetSequenceNumber(socketKey)
				clearInterface(socketKey)
				return "notok", errors.New(errMsg)
			}
		}else{
			framework.Log("Zoom was not included in the PUT command")
		}
	} else {
		errMsg := function +  " - Did not pass coordinates in PUT command"
		framework.AddToErrors(socketKey, errMsg)
		return errMsg, errors.New(errMsg)
	}

	return "ok", nil
}

func setAutoTracking(socketKey string, state string) (string, error) {
	function := "setAutoTracking"

	value := "notok"
	err := error(nil)
	maxRetries := 2
	for maxRetries > 0 {
		value, err = setAutoTrackingDo(socketKey, state)
		if value != "ok" { // Something went wrong - perhaps try again
			framework.Log(function + " - fq3sdvc - retrying autotracking operation")
			maxRetries--
			time.Sleep(1 * time.Second)
			if maxRetries == 0 {
				errMsg := fmt.Sprintf(function + " - fds3nf3 - max retries reached")
				framework.AddToErrors(socketKey, errMsg)
			}
		} else { // Succeeded
			maxRetries = 0
		}
	}

	return value, err
}
// Sets autotracking to On or Off.
func setAutoTrackingDo(socketKey string, state string) (string, error) {
	function := "setAutoTrackingDo"

	autoTrackingValue := ""

	if state == `"on"` {
		autoTrackingValue = "1"
	} else if state == `"off"` {
		autoTrackingValue = "0"
	} else {
		errMsg := fmt.Sprintf(function + " - unrecognized state value: " + state)
		framework.AddToErrors(socketKey, errMsg)
		return state, errors.New(errMsg)
	}

	autoTrackingHexString := "010000070000000081017E043A0" + autoTrackingValue + "FF"

	sent := convertAndSend(socketKey, autoTrackingHexString)

	if !sent {
		errMsg := fmt.Sprintf(function + " - dfji4k - error sending command")
		framework.AddToErrors(socketKey, errMsg)
		return errMsg, errors.New(errMsg)
	}

	respArr, errMsg, err := readAndConvert(socketKey, "SET")

	framework.Log(function + " - Decoded response: " + fmt.Sprintf("% x", respArr))

	if err != nil{
		return errMsg, err
	}

	if len(respArr) > 100 {
		errMsg := "length of respArr is greater than 100. Likely an error with camera. Resetting sequence number"
		framework.AddToErrors(socketKey, errMsg)
		resetSequenceNumber(socketKey)
		clearInterface(socketKey)
		return "notok", errors.New(errMsg)
	}

	// If we got here, the response was good, so successful return with the state indication
	return "ok", nil
}

func healthCheck(socketKey string) (string, error) {
	_, err := getSystemDo(socketKey)
	returnStr := "true"
	if err != nil && strings.Contains(err.Error(), "error sending command") {
		returnStr = "false"
	}
	return `"` + returnStr + `"`, nil
}

// Gets the camera's current system state. Returns "System responded" or error message
func getSystemDo(socketKey string) (string, error) {
	function := "getSystemDo"

	powerHexStr := "011000050000000081090002FF"

	sent := convertAndSend(socketKey, powerHexStr)

	if !sent {
		errMsg := fmt.Sprintf(function + " - fk4kxy755 - error sending command")
		framework.AddToErrors(socketKey, errMsg)
		return errMsg, errors.New(errMsg)
	}

	respArr, errMsg, err := readAndConvert(socketKey, "GET")

	if err != nil{
		return errMsg, err
	}

	value := `"System responded"`
	framework.Log(function + " - Decoded Response: "+ fmt.Sprintf("% x", respArr))

	// If we got here, the response was good, so successful return with the state indication
	return fmt.Sprintf(value), nil
}
