// Copyright 2011 Nicolas Paton. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package apn

import (
	"crypto/tls"
	"os"
	"io"
	"net"
	"fmt"
	"log"
	"time"
	"bytes"
	"encoding/binary"
)

type service struct {
	queue      *Queue
	connection *tls.Conn
	tlscert    tls.Certificate
	connected  bool
	recent     [magickMaxUniqueIdentifierCount]*Notification
}

func newService(queue *Queue, certificatePemFilePath, keyPemFilePath string) (*service, os.Error) {

	cert, err := tls.LoadX509KeyPair(certificatePemFilePath, keyPemFilePath)
	if err != nil {
		return nil, err
	}

	s := &service{tlscert: cert, queue: queue}
	s.serviceResponseCheckLoop()
	return s, nil
}


// Wait for an potential erronerous response
// This sould be in a loop checking now and then for new inbound data
func (s *service) serviceResponseCheckLoop() {
	go func() {
		for {

			for s.connection == nil || !s.connected {
				time.Sleep(5 * 1E9) // We have time for the feedback loop, let's just wait
			}

			// Check service response for returned errors
			var status uint8
			var identifier uint32
			if s.queue.Env != Test {
				status, identifier = checkServiceResponse(s.connection)
			} else {
				// For testing purposes
				status, identifier = checkServiceResponse(bytes.NewBuffer([]byte{}))
			}

			s.handleServiceError(status, identifier)
		}
	}()
}

func checkServiceResponse(reader io.Reader) (status uint8, identifier uint32) {
	var unusedCommand uint8
	binary.Read(reader, binary.BigEndian, &unusedCommand)
	binary.Read(reader, binary.BigEndian, &status)
	binary.Read(reader, binary.BigEndian, &identifier)
	if unusedCommand != 8 {
		log.Println("Wow, new kind of response command from Apple:", unusedCommand)
		log.Println("  -", "status", status)
		log.Println("  -", "identifier", identifier)
	}
	return
}

func (s *service) handleServiceError(status uint8, identifier uint32) {
	if status == 0 { // No error! Shouldn't happen...
		return
	}

	var statusMessage string
	switch status {
	case 1:
		statusMessage = "Processing error"
	case 2:
		statusMessage = "Missing device token"
	case 3:
		statusMessage = "Missing topic"
	case 4:
		statusMessage = "Missing payload"
	case 5:
		statusMessage = "Invalid token size"
	case 6:
		statusMessage = "Invalid topic size"
	case 7:
		statusMessage = "Invalid payload size"
	case 8:
		statusMessage = "Invalid token"
	case 255:
		statusMessage = "None (unknown)"
	default:
		statusMessage = "None (unknown)"
	}

	notif := s.recent[identifier]

	if notif == nil {
		return
	}

	notif.Error = os.NewError(fmt.Sprint("Apple Push Notification Service Error: ", statusMessage, "(", status, ")"))

	s.queue.Error <- notif
}


func (s *service) close() {
	s.connection.Close()
	s.connected = false
}

func (s *service) dialIfNeeded() os.Error {
	if !s.connected {
		return s.dial()
	}
	return nil
}


func (s *service) dial() (err os.Error) {
	if s.queue.Env == Test {
		s.connected = true
		return nil
	}
	log.Println("apn.dial start")

	if s.connection != nil {
		s.close()
	}

	conf := &tls.Config{Certificates: []tls.Certificate{s.tlscert}}

	// connect to the APNS and wrap socket to tls client
	var host string
	if s.queue.Env == Sandbox {
		host = "gateway.sandbox.push.apple.com:2195"
	} else if s.queue.Env == Production {
		host = "gateway.push.apple.com:2195"
	}

	log.Println("Wanting to connect!")
	sockconn, err := net.Dial("tcp", host)
	log.Println("Doing it!")

	if err != nil {
		log.Println("Cannot connect to Apple's service!", err)

		return err
	}
	s.connection = tls.Client(sockconn, conf)

	err = s.connection.Handshake()
	if err != nil {
		fmt.Printf("Handshake error ; your certificates might not be good: %s\n", err)
		s.connection.Close()
		return
	}

	s.connected = true

	log.Println("apn.dial end")
	return
}


func (s *service) writeNotification(notification *Notification) os.Error {

	// Connect to s if needed
	err := s.dialIfNeeded()
	if err != nil {
		return err
	}

	// Write notification to connection
	if s.queue.Env != Test {
		_, err = notification.writeTo(s.connection)
	} else {
		// For testing purposes
		_, err = notification.writeTo(bytes.NewBuffer([]byte{}))
		time.Sleep(2 * 1E6) // Why not sleep a little to fake some lengthy io write
	}

	// Possible errors:
	//   - Invalid payload detected before send
	//   - push notification isn't valid, was refused by Apple and we must not send it again
	//   - the connection wen't down
	// NB: The connection goes down when the notification is not valid...
	// Also, even if it's the network, we must send back the unsent notifications back so the app can handle the error

	if err != nil {
		if err == PayloadJSONEncodingError || err == PayloadTooLargeError || err == BadDeviceToken {
			notification.Error = err
			return notification.Error
		} else {
			if notification.shouldRetry() {
				// Retry write in a couple minutes
				s.close()
				go func(n *Notification, serv *service) {
					time.Sleep(2 * 1E9)
					n.tries = n.tries + 1
					serv.writeNotification(n)
				}(notification, s)
			} else {
				notification.Error = RefusedByApple
				return notification.Error
			}
		}

	} else {
		notification.Sent = true
		s.recent[notification.identifier] = notification
	}
	return nil
}
