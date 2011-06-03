// Copyright 2011 Nicolas Paton. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

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
		t.Fatal("Can't create Queue!! (need valid cert.pem and key.pem in root dir for tests to pass)")
	}

}

func TestPushQueue(t *testing.T) {
	// Without id certificates, will fail
	q, err := NewQueue(Test, "cert.pem", "key.pem")
	if err != nil || q == nil {
		t.Fatal("Can't create Queue!! (need valid cert.pem and key.pem in root dir for tests to pass)")
	}

	payload := makeTestPayload()

	notif := NewNotification(fakeDeviceToken, payload, 0)

	q.Send <- notif

	time.Sleep(1 * 1E9)

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

	// Simulated Apple response
	buffer := bytes.NewBuffer([]byte{})
	binary.Write(buffer, binary.BigEndian, uint8(8))
	binary.Write(buffer, binary.BigEndian, uint8(6))
	binary.Write(buffer, binary.BigEndian, uint32(999999))

	// Checking
	status, identifier := checkServiceResponse(buffer)

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
	q, err := NewQueue(Test, "cert.pem", "key.pem")

	if err != nil {
		t.Fatal("Can't create Queue!! (need valid cert.pem and key.pem in root dir for tests to pass)")
	}

	payload := makeTestPayload()

	notif := NewNotification(fakeDeviceToken, payload, 0)

	// Fake a successfull send and a receiving an error back from Apple
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


// In Test environment, a fake 2ms lag is added to each notification dispath. This does not have much sense... but it's fun!
func BenchmarkAPNSend(b *testing.B) {
	q, _ := NewQueue(Test, "cert.pem", "key.pem")

	go func() {
		for {
			notification := <-q.Error
			if notification.Error != nil {
				log.Println("notification.Error %v", notification.Error)
			}
		}
	}()

	payload := makeTestPayload()
	for i := 0; i < b.N; i++ {
		notif := NewNotification(fakeDeviceToken, payload, 0)
		q.Send <- notif
	}
}
