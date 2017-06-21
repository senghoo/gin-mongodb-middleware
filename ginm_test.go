package ginm_test

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

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
	ID      bson.ObjectId `json:"id,omitempty" bson:"_id,omitempty"`
	Title   string        `json:"title" bson:"title"`
	Content string        `json:"content" bson:"content"`
	Tag     []string      `json:"tag" bson:"tag"`
	Author  struct {
		Name  string `json:"name" bson:"name"`
		Email string `json:"email" bson:"email"`
	} `json:"author" bson:"author"`
}

func (a *Article) PreCreate() error {
	a.ID = bson.NewObjectId()
	return nil
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
	teardown()
	os.Exit(ret)
}

func setup() {

}

func teardown() {
	session.Clone().DB(DatabaseName).DropDatabase()
}

func TestAPI(t *testing.T) {
	e := httpexpect.New(t, server.URL)

	article := &Article{
		Title:   "title",
		Content: "content",
		Tag:     []string{"tag1", "tag2"},
	}
	article.Author.Name = "name"
	article.Author.Email = "name@example.com"

	// new
	e.POST("/articles/").WithJSON(article).
		Expect().
		Status(http.StatusCreated)
	// list
	val := e.GET("/articles/").
		Expect().
		Status(http.StatusOK).
		JSON()
	val.Array().Length().Equal(1)
	val.Array().First().Object().Value("title").Equal("title")
	val.Array().First().Object().Value("content").Equal("content")
	tags := val.Array().First().Object().Value("tag").Array()
	tags.Length().Equal(2)
	tags.Element(0).Equal("tag1")
	tags.Element(1).Equal("tag2")
	author := val.Array().First().Object().Value("author").Object()
	author.Value("name").Equal("name")
	author.Value("email").Equal("name@example.com")

	// get
	id := val.Array().First().Object().Value("id").String().Raw()
	articleURI := fmt.Sprintf("/articles/%s", id)

	val = e.GET(articleURI).
		Expect().
		Status(http.StatusOK).
		JSON()

	val.Object().Value("title").Equal("title")
	val.Object().Value("content").Equal("content")
	tags = val.Object().Value("tag").Array()
	tags.Length().Equal(2)
	tags.Element(0).Equal("tag1")
	tags.Element(1).Equal("tag2")
	author = val.Object().Value("author").Object()
	author.Value("name").Equal("name")
	author.Value("email").Equal("name@example.com")

	// patch
	val = e.PATCH(articleURI).WithJSON(map[string]interface{}{
		"title":       "title2",
		"tag.1":       "updated_tag",
		"author.name": "new name",
	}).Expect().Status(http.StatusOK).JSON()

	val.Object().Value("title").Equal("title2")
	val.Object().Value("content").Equal("content")
	tags = val.Object().Value("tag").Array()
	tags.Length().Equal(2)
	tags.Element(0).Equal("tag1")
	tags.Element(1).Equal("updated_tag")
	author = val.Object().Value("author").Object()
	author.Value("name").Equal("new name")
	author.Value("email").Equal("name@example.com")

	// post

	val = e.PUT(articleURI).WithJSON(article).
		Expect().
		Status(http.StatusOK).JSON()

	val.Object().Value("title").Equal("title")
	val.Object().Value("content").Equal("content")
	tags = val.Object().Value("tag").Array()
	tags.Length().Equal(2)
	tags.Element(0).Equal("tag1")
	tags.Element(1).Equal("tag2")
	author = val.Object().Value("author").Object()
	author.Value("name").Equal("name")
	author.Value("email").Equal("name@example.com")

	// delete

	e.DELETE(articleURI).Expect().Status(http.StatusNoContent)

	e.GET("/articles/").
		Expect().
		Status(http.StatusOK).
		JSON().Array().Length().Equal(1)
}
