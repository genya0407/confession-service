package main

import (
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/julienschmidt/httprouter"
	"log"
	"net/http"
	"regexp"
)

func requireAuthorizationBearer(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Bearer realm=""`)
	w.WriteHeader(http.StatusUnauthorized)
}

func invalidRequest(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Bearer error="invalid_request"`)
	w.WriteHeader(http.StatusBadRequest)
}

func invalidToken(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
	w.WriteHeader(http.StatusUnauthorized)
}

const token68regexp = `[A-Za-z0-9\-._~+/]+`

func fetchBearerToken(s string) (string, error) {
	rep := regexp.MustCompile(fmt.Sprintf(`Bearer (%s)`, token68regexp))
	matched := rep.FindStringSubmatch(s)
	if len(matched) != 2 {
		log.Println(s)
		return "", errors.New("Invalid Request")
	}

	token := matched[1]
	return token, nil
}

type Person struct {
	token string
}

var persons = map[string]Person{
	"aaaaa": Person{"aaaaa"},
}

func findPerson(token string) (Person, bool) {
	p, err := persons[token]
	return p, err
}

func fetchTokenByHeader(r *http.Request) (string, error) {
	authorization := r.Header.Get("Authorization")
	if authorization == "" {
		log.Println(authorization)
		return "", errors.New("Authorization required")
	}

	token, err := fetchBearerToken(authorization)
	if err != nil {
		log.Println(token)
		return "", errors.New("Invalid Request")
	}

	return token, nil
}

func fetchTokenByQueryParams(r *http.Request) (string, error) {
	tokens, ok := r.URL.Query()["access_token"]
	if !ok {
		return "", errors.New("No token specified")
	}

	token := tokens[0]

	re := regexp.MustCompile(token68regexp)
	if !re.MatchString(token) {
		log.Println(token)
		return "", errors.New("Invalid Request")
	}

	return token, nil
}

func fetchToken(r *http.Request) (string, error) {
	token, headerErr := fetchTokenByHeader(r)
	if headerErr == nil {
		log.Println("Token sent by header")
		return token, nil
	}

	token, paramsErr := fetchTokenByQueryParams(r)
	if paramsErr == nil {
		log.Println("Token sent by parameter")
		return token, nil
	}

	if paramsErr.Error() == "No token specified" {
		log.Println("Authorization by header was tried but failed")
		return "", headerErr
	}

	log.Println("Authorization by query parameter was tried but failed")
	return "", paramsErr
}

func AuthorizeBearer(route func(http.ResponseWriter, *http.Request, httprouter.Params, Person)) func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		token, err := fetchToken(r)
		if err != nil {
			switch err.Error() {
			case "Authorization required":
				requireAuthorizationBearer(w)
			case "Invalid Request":
				invalidRequest(w)
			default:
				panic(err.Error())
			}
			return
		}

		person, ok := findPerson(token)
		if !ok {
			log.Printf("Cannot find person by token '%s'", token)
			invalidToken(w)
			return
		}

		route(w, r, ps, person)
	}
}

func Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	fmt.Fprint(w, "Welcome!\n")
}

func Hello(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	fmt.Fprintf(w, "hello, %s!\n", ps.ByName("name"))
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func WebSock(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	log.Println(r)
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	_, p, err := conn.ReadMessage()
	if err != nil {
		log.Println(err)
		return
	}
	log.Println(string(p))
}

func HelloInternal(w http.ResponseWriter, r *http.Request, ps httprouter.Params, p Person) {
	fmt.Fprintf(w, "%s, %s!\n", ps.ByName("greet"), p.token)
}

func main() {
	router := httprouter.New()
	router.GET("/", Index)
	router.GET("/hello/:name", Hello)
	router.GET("/hellointernal/:greet", AuthorizeBearer(HelloInternal))
	router.GET("/connect", WebSock)

	log.Fatal(http.ListenAndServe("localhost:8080", router))
}