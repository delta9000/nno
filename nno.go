package main

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"math/rand"
	"net/http"
)

func main() {
	db := initDB()
	defer db.Close()

	http.HandleFunc("/shorten", handleShortenWithDB(db))
	http.HandleFunc("/r/", handleExpandWithBD(db))
	http.HandleFunc("/", renderIndex())

	//curl commands to test
	//curl -X POST -d "url=http://www.google.com" http://localhost:8080/shorten
	//curl  http://localhost:8080/r/xxxxxx

	http.ListenAndServe(":8080", nil)
}

// renderIndex is a simple handler that renders an inline template html page.
func renderIndex() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Link Shortener</title>
    <link href="https://cdn.jsdelivr.net/npm/tailwindcss@2.2.19/dist/tailwind.min.css" rel="stylesheet">
    <script src="https://unpkg.com/htmx.org"></script>
</head>
<body class="bg-gray-100">
    <div class="container mx-auto p-8">
        <div class="max-w-md mx-auto bg-white rounded-lg overflow-hidden md:max-w-lg">
            <div class="md:flex">
                <div class="w-full p-4">
                    <div class="mb-4">
                        <h1 class="text-center font-bold text-xl">URL Shortener</h1>
                    </div>
                    <form hx-post="/shorten" hx-target="#shortenedUrl" hx-swap="innerHTML">
                        <div class="mb-4">
                            <input type="url" name="url" class="w-full h-12 px-3 rounded text-sm focus:outline-none" placeholder="Enter URL to shorten" required>
                        </div>
                        <div class="mb-4">
                            <button type="submit" class="w-full h-12 text-lg text-white bg-blue-500 rounded hover:bg-blue-600 focus:outline-none">Shorten</button>
                        </div>
                    </form>
                    <!-- This is where the shortened URL will be displayed, it should be stylized and clickable -->
                    <div id="shortenedUrl" class="mb-4"></div>
                </div>
            </div>
        </div>
    </div>
</body>
</html>
`))
	}
}

func handleExpandWithBD(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		shortened := r.URL.Path[3:]
		if shortened == "" {
			http.Error(w, "Shortened URL is required", http.StatusBadRequest)
			return
		}

		stmt, err := db.Prepare("SELECT url FROM links WHERE shortPath = ?")
		if err != nil {
			panic(err)
		}
		defer stmt.Close()

		var url string
		err = stmt.QueryRow(shortened).Scan(&url)
		if err != nil {
			if err.Error() == "sql: no rows in result set" {
				http.Error(w, "Shortened URL not found", http.StatusNotFound)
				return
			}
			panic(err)
		}

		http.Redirect(w, r, url, http.StatusFound)
	}

}

func initDB() *sql.DB {
	db, err := sql.Open("sqlite3", "./nno.db")
	if err != nil {
		panic(err)
	}
	query := `CREATE TABLE IF NOT EXISTS links (id INTEGER PRIMARY KEY AUTOINCREMENT, url TEXT NOT NULL, shortPath TEXT UNIQUE NOT NULL)`
	_, err = db.Exec(query)
	if err != nil {
		panic(err)
	}

	return db
}

func handleShortenWithDB(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		url := r.FormValue("url")
		if url == "" {
			http.Error(w, "URL is required", http.StatusBadRequest)
			return
		}

		stmt, err := db.Prepare("INSERT INTO links (url,shortPath) VALUES (?,?)")
		if err != nil {
			panic(err)
		}
		defer stmt.Close()
		var shortened []byte
		for {
			shortened = encode()
			_, err := stmt.Exec(url, string(shortened))
			if err != nil {
				if err.Error() == "UNIQUE constraint failed: links.shortPath" {
					continue
				}
				panic(err)
			}
			break
		}
		//add the host server to the shortened url and wrap in anchor tag
		shortened = []byte("http://localhost:8080/r/" + string(shortened))

		wrapped := []byte("<a href=\"" + string(shortened) + "\">" + string(shortened) + "</a>")

		w.Write(wrapped)
	}
}

// encode creates a unique 6 character string based on the ID of the link in the database.
// It uses a base62 encoding scheme, which is a modified base64 encoding scheme that uses
// the characters [a-zA-Z0-9] instead of [a-zA-Z0-9+/].
func encode() []byte {
	allowedrunes := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	allowedruneslen := len(allowedrunes)
	encoded := make([]rune, 6)
	for i := 0; i < 6; i++ {
		encoded[i] = allowedrunes[rand.Intn(allowedruneslen)]
	}
	return []byte(string(encoded))
}
