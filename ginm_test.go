package ginm_test

import (
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	mgo "gopkg.in/mgo.v2"

	"github.com/gavv/httpexpect"
	"github.com/gin-gonic/gin"
	ginm "github.com/senghoo/gin-mongodb-middleware"
)

const (
	DatabaseName = "ginm_test"
	CollectName  = "article"
	DBAddress    = "127.0.0.1"
)

var server *httptest.Server
var session *mgo.Session

type Article struct {
	Title   string
	Content string
	Tag     []string
	Author  struct {
		Name  string
		Email string
	}
}

func init() {
	var err error
	// gin handle
	handler := gin.Default()

	// mongodb session
	session, err = mgo.Dial(DBAddress)
	if err != nil {
		log.Fatal(err)
	}

	// middleware
	middleware := ginm.NewMiddleware(session)
	handler.Use(middleware.Connect)

	// blueprint
	group := handler.Group("/articles")
	blueprint := ginm.NewBlueprint(new(Article), DatabaseName, CollectName)
	blueprint.Routes(group)

	server = httptest.NewServer(handler)
}

func TestMain(m *testing.M) {
	setup()
	ret := m.Run()
	if ret == 0 {
		teardown()
	}
	os.Exit(ret)
}

func setup() {

}

func teardown() {
	session.Clone().DB(DatabaseName).DropDatabase()
}

func TestNew(t *testing.T) {
	e := httpexpect.New(t, server.URL)

	article := &Article{
		Title:   "title",
		Content: "content",
		Tag:     []string{"tag1", "tag2"},
	}
	article.Author.Name = "name"
	article.Author.Email = "name@example.com"

	e.POST("/articles/").WithJSON(article).
		Expect().
		Status(http.StatusCreated)
}
