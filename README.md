This small utility verifies incoming request and parses slack command for you.
## How to install

```
go get github.com/linhlc888/msgutil

```
## How to use
Please take a look example code as below:

```
package main

import (
	"github.com/linhlc888/msgutil"
	"log"
	"net/http"
	"os"
)

func main() {
	http.HandleFunc("/slack/cmd", func(resp http.ResponseWriter, req *http.Request) {

		var slack = &msgutil.Slack{
			LogWriter:       os.Stdout,
			MySigningSecret: os.Getenv("MY_SIGNING_SECRET"),
		}
		err := slack.ParseCmd(req)
		if err != nil {
			panic(err)
			return
		}
		log.Printf("payload = %s\n", slack.Payload)
		resp.Header().Add("Content-Type", "application/json")
		resp.WriteHeader(http.StatusOK)
	})

	panic(http.ListenAndServe(":8080", nil))

}
```
