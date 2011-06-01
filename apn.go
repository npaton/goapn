// Copyright 2011 Nicolas Paton. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*

apn is a Go Apple Push Notification package. Heavily uses queues and goroutines. 
It has a few tests and **seems** to work properly but it hasn't had thourough testing and hasn't gone into production yet. 

NB: This is my first Go code ever.


Installation

        goinstall github.com/nicolaspaton/goapn


Usage
    
        import "github.com/nicolaspaton/goapn"

        // You can create multiple queues for different environments and apps
        q, err := apn.NewQueue(apn.Sandbox, "cert.pem", "key.pem")
        
        if err != nil {
            log.Fatalln("Error loading queue", err)
        }
        
        // The payload is a nested string to interface{} map that respects Apple doc [1]. You should too.
        payload := make(map[string]interface{})
        payload["aps"] = map[string]interface{}{
            "alert": "You've got emails.",
            "badge": 9,
            "sound": "bingbong.aiff",
        }
        payload["foo"] = "bar"
        payload["answer"] = 42
        
        // Et hop, send!
        q.Send <- apn.NewNotification("3f6e...device token here", payload)

        // Start a loop in a goroutine somewhere to handle errors
        // The erronerous notification is returned with an non null Error (os.Error) attribute
        // This interface handles internal validation errors and errors sent back by apple [2]
        // It will eventually also return, in the same way, the feedback service errors soon [3]
        go func(q *apn.Queue) {
            for {
                notification := <- q.Error
                if notification.Error != nil {
                    // Do something with that notification
                }
            }
        }(q)


Certificates

You need a certification and an unprotected key pem file. See http://blog.boxedice.com/2010/06/05/how-to-renew-your-apple-push-notification-push-ssl-certificate/

Reminder, after you've got you .p12 files

        openssl pkcs12 -clcerts -nokeys -out dev-cert.pem -in dev-cert.p12
        openssl pkcs12 -nocerts -out dev-key.pem -in dev-key.p12
        openssl rsa -in dev-key.pem -out dev-key-noenc.pem



License

BSD-style. See LICENSE file.


Apple and Apple Push Notifications are trademarks owned by Apple inc. and have nothing to do with the creator of this software.


[1] http://developer.apple.com/library/ios/#documentation/NetworkingInternet/Conceptual/RemoteNotificationsPG/ApplePushService/ApplePushService.html%23//apple_ref/doc/uid/TP40008194-CH100-SW9

[2] http://developer.apple.com/library/ios/#documentation/NetworkingInternet/Conceptual/RemoteNotificationsPG/CommunicatingWIthAPS/CommunicatingWIthAPS.html%23//apple_ref/doc/uid/TP40008194-CH101-SW4

[3] http://developer.apple.com/library/ios/#documentation/NetworkingInternet/Conceptual/RemoteNotificationsPG/CommunicatingWIthAPS/CommunicatingWIthAPS.html%23//apple_ref/doc/uid/TP40008194-CH101-SW4

*/
package apn

import (
// "os"
// "log"
// "time"
// "os/signal"

// "fmt"
// "sync"
// "path"
// "crypto/tls"
// "net"
// "bytes"
)


type ApnEnv int

var (
	Test       = ApnEnv(0)
	Sandbox    = ApnEnv(1)
	Production = ApnEnv(2)
)

func envString(env ApnEnv) (envString string) {
	if env == Test {
		envString = "test"
	} else if env == Sandbox {
		envString = "sandbox"
	} else if env == Production {
		envString = "production"
	}
	return envString
}

func envObject(envString string) (env ApnEnv) {
	if envString == "sandbox" {
		env = Sandbox
	} else if envString == "production" {
		env = Production
	}
	return
}


var queues []Queue = make([]Queue, 10)



