package ginm

import (
	"github.com/gin-gonic/gin"
	mgo "gopkg.in/mgo.v2"
)

var db *mgo.Database

func SetDB(d *mgo.Database) {
	db = d
}

func Connect(c *gin.Context) {
	s := db.Session.Clone()

	defer s.Close()

	c.Set("db", s)
	c.Next()
}
