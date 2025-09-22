package main

import (
	"flag"
	"log"
	"net/http"
)

// We will access the POST /v1/tokens/authentication endpoint
const html = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
</head>
<body>
    <h1>Appletree Preflight CORS</h1>
    <div id="output"></div>
    <script>
         document.addEventListener('DOMContentLoaded', function() {
         fetch("http://localhost:4000/v1/tokens/authentication", {
           method: "POST",
           headers: {
                    'Content-Type': 'application/json'
                    },
           body: JSON.stringify({
                    email: 'john@example.com',
                    password: 'mangotree'
                 })
           }).then( function(response) {
      response.text().then(function (text) {
                  document.getElementById("output").innerHTML = text;
               });
             },
             function(err) {
               document.getElementById("output").innerHTML = err;
             }
          );
       });
  </script>
</body>
</html>`

// A very simple HTTP server
func main() {
	addr := flag.String("addr", ":9000", "Server address")
	flag.Parse()

	log.Printf("starting server on %s", *addr)

	err := http.ListenAndServe(*addr,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(html))
		}))
	log.Fatal(err)
}
