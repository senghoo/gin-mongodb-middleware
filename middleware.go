package ginm

import (
	"github.com/gin-gonic/gin"
	mgo "gopkg.in/mgo.v2"
)

type Middleware struct {
	session *mgo.Session
}

func NewMiddleware(session *mgo.Session) *Middleware {
	return &Middleware{
		session: session,
	}
}

func (m *Middleware) Connect(c *gin.Context) {
	s := m.session.Clone()

	defer s.Close()

	c.Set("db", s)
	c.Next()
}
