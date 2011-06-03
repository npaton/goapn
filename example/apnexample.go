package main

import (
    "github.com/nicolaspaton/goapn"
    "log"
    "time"
)

func main() {
    // Add cert.pem and key.pem to this folder
    q, err := apn.NewQueue(apn.Sandbox, "cert.pem", "key.pem")
    
    if err != nil {
        log.Fatalln("Error loading queue", err)
    }

    payload := make(map[string]interface{})
    payload["aps"] = map[string]interface{}{
        "alert": "You've got emails.",
        "badge": 9,
        "sound": "default",
    }
    
    deviceToken := "Valid Device Token Here"
    
    notification := apn.NewNotification(deviceToken, payload, 0)
    // log.Println("Notifcation", notification)
    
    q.Send <- notification

    go func(q *apn.Queue) {
        for {
            notification := <- q.Error
            log.Fatalln(notification.Error)
        }
    }(q)
    
    time.Sleep(5*1E9) // Wait for an error response from the service
}
