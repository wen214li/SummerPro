package main
import(
	elastic "gopkg.in/olivere/elastic.v3"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"time"
	"github.com/dgrijalva/jwt-go"
)

const (
	TYPE_USER = "user"
)

var (
	usernamePattern = regexp.MustCompile(`^[a-z0-9_]+$`).MatchString
)
type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Age int `json:"age"`
	Gender string `json:"gender"`
}
func checkUser(username, password string) bool {
	es_client, err := elastic.NewClient(elastic.SetURL(ES_URL), elastic.SetSniff(false))
	if(err !=nil){
		fmt.Printf("ES is not setup %v\n", err)
		return false
	}

	termQuery:= elastic.NewTermQuery("username", username)
	queryResult, err:= es_client.Search().Index(INDEX).Query(termQuery).Pretty(true).Do()
	if err!= nil{
		fmt.Printf("ES query failed %v\n", err)
		return false
	}
	var tyu User
	for _,item := range queryResult.Each(reflect.TypeOf(tyu)){
		u:= item.(User)
		return u.Password == password && u.Username == username
	}
	return false

}

func addUser(user User) bool {
	es_client, err := elastic.NewClient(elastic.SetURL(ES_URL), elastic.SetSniff(false))
	if err!=nil{
		fmt.Printf("ES is not setup %v\n", err)
		return false
	}

	termQuery := elastic.NewTermQuery("username", user.Username)
	queryResult, err:= es_client.Search().Index(INDEX).Query(termQuery).Pretty(true).Do()

	if err!=nil{
		fmt.Printf("ES query failed %v\n", err)
		return false
	}

	if queryResult.TotalHits() > 0{
		fmt.Printf("User has already exists\n", user.Username)
		return false
	}

	_,err = es_client.Index().Index(INDEX).Type(TYPE_USER).Id(user.Username).BodyJson(user).Refresh(true).Do()
	if err!= nil{
		fmt.Printf("ES save user failed %v\n", err)
		return false
	}

	return true
}

func signupHandler(w http.ResponseWriter, r *http.Request){
	fmt.Println("Received an signup request")

	decoder := json.NewDecoder(r.Body)
	var u User
	if err:= decoder.Decode(&u); err != nil{
		panic(err)
	}

	if u.Username != "" && u.Password != "" && usernamePattern(u.Username){
		if addUser(u){
			fmt.Println("User added succesfully.")
			w.Write([]byte("User added succesfully."))

		}else{
			fmt.Println("Failed to add a new user")
			http.Error(w, "Failed to add a new user", http.StatusInternalServerError)

		}
	}else{
		fmt.Println("Empty password or username")
		http.Error(w, "Empty password or username", http.StatusInternalServerError)
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Access-Control-Allow-Origin", "*")

}

func loginHandler(w http.ResponseWriter, r *http.Request){
	fmt.Println("Received one login request")

	decoder:= json.NewDecoder(r.Body)
	var u User
	if err:= decoder.Decode(&u); err != nil{
		panic(err)
		return
	}

	if checkUser(u.Username, u.Password){
		token:= jwt.New(jwt.SigningMethodHS256)
		claims:= token.Claims.(jwt.MapClaims)
		claims["username"] = u.Username
		claims["exp"] = time.Now().Add(time.Hour * 24).Unix()

		tokenString,_ := token.SignedString(mySigningKey)
		w.Write([]byte(tokenString))

	}else{
		fmt.Println("Invalid password or username")
		http.Error(w,"Invalid password or username", http.StatusForbidden)
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Access-Control-Allow-Origin", "*")
}