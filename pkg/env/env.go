package env 

import(
	"log"
	"os"

	"github.com/joho/godotenv"
)

func LoadEnv(){
	err := godotenv.Load()

	if err != nil{
		log.Println("⚠️  No .env file found, using system envs")
	}
}

func GetEnv(key string, fallback string) string{
	if value, exist := os.LookupEnv(key); exist{
		return value
	}
	return fallback
}

