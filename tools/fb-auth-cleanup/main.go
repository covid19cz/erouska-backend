package main

import (
	"context"
	"log"

	firebase "firebase.google.com/go"
	"google.golang.org/api/iterator"
)

func main() {

	ctx := context.Background()

	conf := &firebase.Config{
		ProjectID: "covid19cz",
		//ProjectID: "daring-leaf-272223",
	}

	app, err := firebase.NewApp(ctx, conf)
	if err != nil {
		log.Fatalf("error initializing app: %v\n", err)
	}

	client, err := app.Auth(ctx)
	if err != nil {
		log.Fatalf("error getting Auth client: %v\n", err)
	}

	// Note, behind the scenes, the Users() iterator will retrive 1000 Users at a time through the API
	iter := client.Users(ctx, "")
	for {
		user, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatalf("error listing users: %s\n", err)
		}
		if user.PhoneNumber != "" {
			log.Printf("read user: %+v\n", user.PhoneNumber)
		}
	}
}
