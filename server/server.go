package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// gorm.Model definition
type User struct {
	Name  string  `gorm:"<-"`
	Email *string `gorm:"<-"`
}

type UserStore struct {
	db *gorm.DB
}

type UserHandler struct {
	userStore UserStore
}

func healthHandler(w http.ResponseWriter, r *http.Request) {

	go func() {
		randomInt := rand.Intn(3) + 1
		time.Sleep(time.Second * time.Duration(randomInt))
	}()
	jsonResponse := map[string]string{"status": "ok"}
	jsonData, err := json.Marshal(jsonResponse)
	handleErr(err)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)

}

func (user2 *UserStore) createUser(ctx context.Context, u *User) User {
	result := gorm.WithResult()
	user := User{Name: u.Name, Email: u.Email}
	err := gorm.G[User](user2.db, result).Create(ctx, &user)
	handleErr(err)
	fmt.Println("No. of Rows : ", result.RowsAffected)
	return user
}

func (uh UserHandler) userHandler(w http.ResponseWriter, r *http.Request) {
	var user User
	err := json.NewDecoder(r.Body).Decode(&user)
	handleErr(err)
	newUser := uh.userStore.createUser(r.Context(), &user)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(newUser)
}

func getEnv(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func databaseDSN() string {
	host := getEnv("POSTGRES_HOST", "localhost")
	port := getEnv("POSTGRES_PORT", "5435")
	user := getEnv("POSTGRES_USER", "postgres")
	password := getEnv("POSTGRES_PASSWORD", "postgres")
	name := getEnv("POSTGRES_DB", "postgres")
	sslMode := getEnv("POSTGRES_SSLMODE", "disable")
	timeZone := getEnv("POSTGRES_TIMEZONE", "Asia/Shanghai")

	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=%s",
		host,
		user,
		password,
		name,
		port,
		sslMode,
		timeZone,
	)
}

func main() {
	db, err := gorm.Open(postgres.Open(databaseDSN()), &gorm.Config{})
	handleErr(err)
	db.AutoMigrate(&User{})
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)
	userStore := UserStore{db: db}
	userHandlerInstance := UserHandler{userStore: userStore}
	mux.HandleFunc("POST /createuser", userHandlerInstance.userHandler)
	srv := http.Server{Addr: ":3030", Handler: mux}
	srv.ListenAndServe()
}

func handleErr(err error) {
	if err != nil {
		fmt.Println("Error: ", err)
		os.Exit(1)
	}
}
