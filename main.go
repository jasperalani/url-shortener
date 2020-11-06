package main

import (
	"log"
	"math/rand"
	"net/http"
	"regexp"
	"time"
	"unsafe"

	sq "github.com/Masterminds/squirrel"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

var dB *sqlx.DB

func main() {

	dB = initDB()

	r := gin.Default()

	r.POST("/shorten", shorten)
	r.GET("/:forward", forward)

	r.Run(":8080")
}

func initDB() *sqlx.DB {
	db, err := sqlx.Connect("mysql", "root:123456@(localhost:3306)/url-shortener")
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Successfully connected to the database.")
	return db
}

func shorten(c *gin.Context) {
	url := c.Query("url")
	if "" == url {
		respond(c, 400, gin.H{
			"error":   "query param not found",
			"message": "Query param 'url' must be present.",
		})
		return
	}

	generatedIdentifier := randStringBytesMaskImprSrcUnsafe(7)

	insert := sq.Insert("`forwarding-table`").Columns("short", "`long`").Values(generatedIdentifier, url)

	_, err := insert.RunWith(dB).Query()
	logFatal(err)

	respond(c, 400, gin.H{
		"id": generatedIdentifier,
	})

}

func logFatal(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

var src = rand.NewSource(time.Now().UnixNano())

func randStringBytesMaskImprSrcUnsafe(n int) string {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return *(*string)(unsafe.Pointer(&b))
}

func forward(c *gin.Context) {

	forwardID := c.Param("forward")

	valid, err := regexp.MatchString(`^[a-zA-Z0-9]{7}$`, forwardID)
	if err != nil {
		log.Fatal("Fatal error when matching regexp.")
	}

	if !valid {
		respond(c, http.StatusOK, gin.H{
			"error":   "invalid identifier",
			"message": "A forwarding policy matching the supplied identifier was not found.",
		})
		return
	}

	realPolicy := sq.Select("`long`").From("`forwarding-table`").Where(sq.Eq{"short": forwardID})

	possibleLong, err := realPolicy.RunWith(dB).Query()
	logFatal(err)

	var long string

	possibleLong.Next()

	err = possibleLong.Scan(&long)
	if err != nil {
		log.Fatal(err)
	}

	var httpPrefix = []string{
		"http://",
		"https://",
	}

	if httpPrefix[0] != long[0:7] && httpPrefix[1] != long[0:6] {
		long = "http://" + long
	}

	c.Redirect(http.StatusMovedPermanently, long)
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

type policy struct {
	short string `db:"short"`
	long  string `db:"long"`
}

func respond(c *gin.Context, code int, res gin.H) {
	c.JSON(code, res)
}
