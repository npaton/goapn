// Copyright 2011 Nicolas Paton. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package apn

import (
	"os"
	"sync"
    // "log"
)

type Queue struct {
	Env     ApnEnv   // The dispatch environment (sandbox|production)
	service *service // The connection to Apple
	Send    chan *Notification
	Error   chan *Notification
	mu      sync.RWMutex
	idCount uint32
}

func NewQueue(env ApnEnv, certificatePemFilePath, keyPemFilePath string) (queue *Queue, err os.Error) {
	// log.Println("New queue in env:", envString(env))
	queue = &Queue{
		Env:   env,
		Send:  make(chan *Notification, 1000),
		Error: make(chan *Notification, 100),
	}
	queue.service, err = newService(queue, certificatePemFilePath, keyPemFilePath)
	if err != nil {
		return nil, err
	}
	
	go queue.dispatchLoop()

	return
}

func (q *Queue) uniqueIdentifier() uint32 {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.idCount = q.idCount + 1
	if q.idCount > 999 {
		q.idCount = 1
	}
	return q.idCount
}

func (q *Queue) dispatchLoop() {
	for {
	    
		notification := <-q.Send
		
		notification.queue = q
		notification.identifier =  q.uniqueIdentifier()
		
		err := notification.validateNotification()
		
    	if err != nil {
    	    q.Error <- notification
    	} else {
    		err := q.service.writeNotification(notification)

    		if err != nil {
    		    q.Error <- notification
    		}
    	}
		
	}
}
