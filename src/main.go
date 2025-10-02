package main
import(
    "fmt"
    "net/http"
    // "database/sql"
    "encoding/json"
)

type User struct {
    Username string `json:"username"`
    Password string `json:"password"`
}

func homeHandler(w http.ResponseWriter, r *http.Request)  {
    w.Header().Set("Content-type","text/html")
    fmt.Fprintln(w,"<h2>Welcome to the home page</h2>")
}

func signupHandler(w http.ResponseWriter, r *http.Request)  {
    if r.Method != http.MethodPost {
        http.Error(w,"Only POST method allowed",http.StatusMethodNotAllowed)
        return
    }
    var user User
    if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }
    var exists int
    err := DB.QueryRow("SELECT COUNT(*) FROM users WHERE username = ?", user.Username).Scan(&exists)
    if err != nil {
        http.Error(w,"Database error(query)",http.StatusInternalServerError)
        return
    }
    if exists > 0 {
        http.Error(w,"User already exists",http.StatusConflict)
        return
    }
    _, err = DB.Exec("INSERT INTO users(username, password) VALUES(?, ?)", user.Username, user.Password)
    if err != nil {
        http.Error(w, "Database error(execute)", http.StatusInternalServerError)
        return
    }
    w.WriteHeader(http.StatusCreated)
    fmt.Fprintln(w, "Signup successful")
}

func signinHandler(w http.ResponseWriter, r *http.Request)  {
    var pass string
    var user User
    if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }
    err := DB.QueryRow("SELECT password FROM users WHERE username = ?",user.Username).Scan(&pass)
    if pass != user.Password {
        http.Error(w,"Invalid user",http.StatusUnauthorized)
        return
    } else if err != nil {
        http.Error(w,"Database error",http.StatusInternalServerError)
        return
    }
    fmt.Fprintln(w,"Signin successful")
}

func main()  {
    if err := InitDB(); err != nil {
        panic(err)
    }
    http.HandleFunc("/",homeHandler)
    http.HandleFunc("/signup",signupHandler)
    http.HandleFunc("/signin",signinHandler)
    fmt.Println("Server is running on http://localhost:8080")
    http.ListenAndServe(":8080",nil)
}