package apn

import (
	"encoding/hex"
	"encoding/binary"
	"io"
	"os"
	"json"
	"bytes"
	"time"
)


type Notification struct {
	queue       *Queue // Parent queue
	Payload     map[string]interface{}
	DeviceToken string
	identifier  uint32 // The identifier used to find on erronerous notifications
	tries       int    // Number of times tried to be sent
	Sent        bool   // True if sent, amazing
	Invalid     bool
	Error       os.Error
}

const NUMBER_OF_RETRIES = 2

var PayloadTooLargeError = os.NewError("Payload size exceeds limit (255 bytes)")
var BadDeviceToken = os.NewError("The given device token is not valid device token")
var PayloadJSONEncodingError = os.NewError("Notification's payload cannot be encoded to JSON, it must not respect the format")
var RefusedByApple = os.NewError("After several tries, the notification can't find it's way to Apple. (Can be a connection error!)")

func NewNotification(token string, payload map[string]interface{}) *Notification {
	return &Notification{
		Payload:     payload,
		DeviceToken: token,
		Invalid:     false,
	}
}

func (n *Notification) validateNotification() os.Error {
    if len(n.DeviceToken) != 64 {
        return BadDeviceToken
    }
    
    pload, err := n.jsonPayload()
    if err != nil {
        return PayloadJSONEncodingError
    }
    
    if len(pload) > 255 {
        return PayloadTooLargeError
    }

	return nil
}

func (n *Notification) shouldRetry() bool {
	return n.tries < NUMBER_OF_RETRIES && !n.Invalid
}

func (n *Notification) jsonPayload() (string, os.Error) {
	bytes, err := json.Marshal(n.Payload)
	return hex.EncodeToString(bytes), err
}


func (n *Notification) writeTo(writer io.Writer) os.Error {
	// prepare binary payload from JSON structure
	// payload := make(map[string]interface{})
	// payload["aps"] = map[string]string{"alert": "Hello Push"}
	payload, err := n.jsonPayload()
	
	if err != nil {
		return PayloadJSONEncodingError
	}

	if len(payload) > 255 {
	    return PayloadTooLargeError
	}

	// Decode hexadecimal push device token to binary byte array
	token, _ := hex.DecodeString(n.DeviceToken)

	// Buffer in which to stuff the full payload
	buffer := bytes.NewBuffer([]byte{})

	// command
	binary.Write(buffer, binary.BigEndian, uint8(1))

	// Identifier
	binary.Write(buffer, binary.BigEndian, uint32(n.identifier))

	// Expiry
	binary.Write(buffer, binary.BigEndian, uint32(time.Seconds()+60*60))

	// Device token
	binary.Write(buffer, binary.BigEndian, uint16(len(token)))
	binary.Write(buffer, binary.BigEndian, token)

	// Actual payload
	binary.Write(buffer, binary.BigEndian, uint16(len(payload)))
	binary.Write(buffer, binary.BigEndian, payload)

	// Write the bytes
	payloadBytes := buffer.Bytes()
	_, err = writer.Write(payloadBytes)
	return err
}
