// Copyright 2011 Nicolas Paton. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package apn

import (
	"os"
	"sync"
)

type Queue struct {
	Env   ApnEnv             // The dispatch environment (sandbox|production)
	Send  chan *Notification // You send notifications to this channel
	Error chan *Notification // You get back notifications that can't get sent on this channel. Notifications returned contain a non-null Error field. Handle errors quickly.

	service *service     // The connection to Apple
	mu      sync.RWMutex // Only to create unique identifiers for the notifications' identifiers. Maybe ridiculous, no concurency arround the unique id assignment.
	idcount uint32       // Unique identifier increment. Loops at 65535 (uint16 max, also known as the magickMaxUniqueIdentifierCount).
}

func NewQueue(env ApnEnv, certificatePemFilePath, keyPemFilePath string) (queue *Queue, err os.Error) {
	queue = &Queue{
		Env:     env,
		Send:    make(chan *Notification, 1000), // Send can take time and be blocking
		Error:   make(chan *Notification, 10),   // You should handle the errors quickly. Use goroutines if blocking operation at the output of this channel
		idcount: magickUniqueIdentifierNumber,
	}
	queue.service, err = newService(queue, certificatePemFilePath, keyPemFilePath)
	if err != nil {
		return nil, err
	}

	go queue.dispatchLoop()

	return
}

// I'm not good with standard type (uint32 in this case), but 1 + 1 does not always equal 2 under 5000. And other oddities on idcount increment.
const magickUniqueIdentifierNumber = 5001
const magickMaxUniqueIdentifierCount = 65535

func (q *Queue) uniqueIdentifier() uint32 {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.idcount = q.idcount + 1
	if q.idcount == magickMaxUniqueIdentifierCount {
		q.idcount = magickUniqueIdentifierNumber
	}
	return q.idcount
}

func (q *Queue) dispatchLoop() {
	for {
		// Receiving a notification from exported Send queue. Blocking.
		notification := <-q.Send

		// Fill in the blanks
		notification.queue = q
		notification.identifier = q.uniqueIdentifier()

		// Check for obvious errors before sending
		err := notification.validateNotification()

		// If errors found, send back immediatly
		if err != nil {
			q.Error <- notification
		} else {
			// Or try to write to apn connection directly, just like that, hop.
			err = q.service.writeNotification(notification)

			if err != nil {
				q.Error <- notification
			}
		}

	}
}
