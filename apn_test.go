package apn

import "testing"

import "log"
import "time"

import "encoding/binary"
import "bytes"

const fakeDeviceToken = "bedb115e0f9afef1bbc49eb03cd789365956aa4bef1f6229f504541f8e2dfdca"

func init() {
	log.Println("")
}

func TestPushQueueWrongInit(t *testing.T) {
	// Without id certificates, will fail
	q, err := NewQueue(Test, "", "")
	if err == nil || q != nil {
		t.Fail()
	}

}

func TestPushQueue(t *testing.T) {
	// Without id certificates, will fail
	q, err := NewQueue(Test, "cert.pem", "key.pem")
	if err != nil || q == nil {
		t.Fatal("Can't create Queue!!")
	}
	
	payload := makeTestPayload()

	notif := NewNotification(fakeDeviceToken, payload)

	q.Send <- notif
	
	time.Sleep(0.1*1E9)
	
	// Should be found in the recent
	if q.service.recent[notif.identifier] == nil {
		t.Fail()
	}
}

func makeTestPayload() map[string]interface{} {
    payload := make(map[string]interface{})
	payload["aps"] = map[string]interface{}{
	    "alert": "You've got emails.",
	    "badge": 9,
	    "sound": "bingbong.aiff",
	}
	payload["foo"] = "bar"
	payload["answer"] = 42
	return payload
}

func TestBinaryReadErrorFromServiceResponse(t *testing.T) {
    
    // Simulated response
	buffer := bytes.NewBuffer([]byte{})
	binary.Write(buffer, binary.BigEndian, uint8(8))
	binary.Write(buffer, binary.BigEndian, uint8(6))
	binary.Write(buffer, binary.BigEndian, uint32(999999))
    
    // Checking
	status, identifier := checkServiceResponse(bytes.NewBuffer(buffer.Bytes()[:]))

	if status != 6 {
		t.Fail()
		t.Log("Failed reading status should be %v was %v", 6, status)
	}

	if identifier != 999999 {
		t.Fail()
		t.Log("Failed reading identifier should be %v was %v", 999999, identifier)
	}
}

func TestServiceErrorChan(t *testing.T) {
	q, _ := NewQueue(Test, "cert.pem", "key.pem")
	
    payload := makeTestPayload()
    
	notif := NewNotification(fakeDeviceToken, payload)
	q.service.recent[notif.identifier] = notif
	q.service.handleServiceError(2, notif.identifier)

	_, ok := <-q.Error

	if !ok {
		t.Fail()
		t.Log("Didn't receive the faultfull notification on the error channel!")
	}

	if notif.Error == nil {
		t.Fail()
		t.Log("The error has not been filled!")
	}

}


