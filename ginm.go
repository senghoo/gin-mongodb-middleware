package ginm

import (
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/gedex/inflector"
	"github.com/gin-gonic/gin"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// Blueprint ...
type Blueprint struct {
	session    *mgo.Session
	database   string
	collection string
	t          reflect.Type
	*gin.RouterGroup
}

func NewBlueprint(instance interface{}, session *mgo.Session, database, collection string) *Blueprint {
	t := reflect.TypeOf(instance)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return &Blueprint{
		session:    session,
		t:          t,
		database:   database,
		collection: collection,
	}
}

const (
	RouteNew = 1 << iota
	RouteList
	RouteGet
	RouteUpdate
	RoutePatch
	RouteDelete
	RouteALL = RouteNew | RouteList | RouteGet | RouteUpdate | RoutePatch | RouteDelete
)

func (b *Blueprint) coll(c *gin.Context) *mgo.Collection {
	session := c.MustGet("db").(*mgo.Session)
	return session.Copy().DB(b.database).C(b.collection)
}

func (b *Blueprint) Routes(router *gin.RouterGroup, routes ...int) {
	var route = RouteALL
	name := inflector.Pluralize(strings.ToLower(b.t.Name()))
	group := router.Group(name)
	if len(routes) == 1 {
		route = routes[0]
	}
	if (route & RouteNew) != 0 {
		group.POST("/", b.New)
	}

	if (route & RouteList) != 0 {
		group.GET("/", b.List)
	}
	if (route & RouteGet) != 0 {
		group.GET("/:id", b.Get)
	}
	if (route & RouteUpdate) != 0 {
		group.PUT("/:id", b.Update)
	}
	if (route & RoutePatch) != 0 {
		group.PATCH("/:id", b.Patch)
	}
	if (route & RouteDelete) != 0 {
		group.DELETE("/:id", b.Delete)
	}
	b.RouterGroup = group
}

func (b *Blueprint) New(c *gin.Context) {
	instValue := reflect.New(b.t)
	inst := instValue.Interface()
	err := c.BindJSON(inst)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	coll := b.coll(c)
	if i, ok := inst.(PreCreate); ok {
		i.PreCreate()
	}

	err = coll.Insert(inst)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	if i, ok := inst.(PostCreate); ok {
		i.PostCreate()
	}

	c.Status(http.StatusCreated)

}

func (b *Blueprint) List(c *gin.Context) {
	queryM := bson.M{}
	for key, val := range c.Request.URL.Query() {
		if !inArray(key, []string{"_limit", "_offset", "_sort"}) {
			queryM[key] = val
		}

	}
	coll := b.coll(c)
	query := coll.Find(queryM)
	// limit and offset
	limitString := c.DefaultQuery("_limit", "0")
	limit, err := strconv.Atoi(limitString)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	offsetString := c.DefaultQuery("_offset", "0")
	offset, err := strconv.Atoi(offsetString)

	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	if offset > 0 {
		query = query.Skip(offset)
	}

	if limit > 0 {
		query = query.Limit(limit)
	}

	// sorting
	sort := c.DefaultQuery("_sort", "")
	if sort != "" {
		query.Sort(sort)
	}

	res := reflect.New(reflect.SliceOf(reflect.PtrTo(b.t))).Interface()
	err = query.All(res)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	c.JSON(200, res)
}

func (b *Blueprint) Get(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.AbortWithError(http.StatusNotFound, fmt.Errorf("not found %s", id))
		return
	}
	instValue := reflect.New(b.t)
	inst := instValue.Interface()
	coll := b.coll(c)
	err := coll.FindId(id).One(inst)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	c.JSON(200, inst)
}

func (b *Blueprint) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.AbortWithError(http.StatusNotFound, fmt.Errorf("not found %s", id))
		return
	}

	instValue := reflect.New(b.t)
	inst := instValue.Interface()
	err := c.BindJSON(inst)
	if err != nil {
		c.AbortWithError(http.StatusNotFound, err)
		return
	}

	if i, ok := inst.(PreUpdate); ok {
		i.PreUpdate()
	}

	coll := b.coll(c)
	err = coll.UpdateId(id, inst)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	if i, ok := inst.(PostUpdate); ok {
		i.PostUpdate(inst)
	}
	err = coll.FindId(id).One(inst)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	c.JSON(http.StatusCreated, inst)
}

func (b *Blueprint) Patch(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.AbortWithError(http.StatusNotFound, fmt.Errorf("not found %s", id))
		return
	}

	coll := b.coll(c)
	instValue := reflect.New(b.t)
	inst := instValue.Interface()
	err := coll.FindId(id).One(inst)
	if err != nil {
		c.AbortWithError(http.StatusNotFound, err)
		return
	}

	data := new(bson.M)
	err = c.BindJSON(data)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	if i, ok := inst.(PreUpdate); ok {
		i.PreUpdate()
	}
	err = coll.UpdateId(id, bson.M{"$set": data})
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	instValue = reflect.New(b.t)
	inst = instValue.Interface()
	err = coll.FindId(id).One(inst)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	if i, ok := inst.(PostUpdate); ok {
		i.PostUpdate()
	}
	c.JSON(http.StatusCreated, inst)
}

func (b *Blueprint) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.AbortWithError(http.StatusNotFound, fmt.Errorf("not found %s", id))
		return
	}
	coll := b.coll(c)
	err := coll.RemoveId(id)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	c.Status(http.StatusNoContent)
}
