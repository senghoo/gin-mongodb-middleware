package ginm

import (
	"fmt"
	"net/http"
	"reflect"
	"strconv"

	"github.com/gin-gonic/gin"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// Blueprint ...
type Blueprint struct {
	database   string
	collection string
	t          reflect.Type
}

func NewBlueprint(instance interface{}, database, collection string) *Blueprint {
	t := reflect.TypeOf(instance)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return &Blueprint{
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

func (b *Blueprint) Routes(r gin.IRoutes, routes ...int) {
	var route = RouteALL
	if len(routes) == 1 {
		route = routes[0]
	}
	if (route & RouteNew) != 0 {
		r.POST("/", b.New)
	}

	if (route & RouteList) != 0 {
		r.GET("/", b.List)
	}
	if (route & RouteGet) != 0 {
		r.GET("/:id", b.Get)
	}
	if (route & RouteUpdate) != 0 {
		r.PUT("/:id", b.Update)
	}
	if (route & RoutePatch) != 0 {
		r.PATCH("/:id", b.Patch)
	}
	if (route & RouteDelete) != 0 {
		r.DELETE("/:id", b.Delete)
	}
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

		if err := i.PreCreate(); err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
		}
	}

	err = coll.Insert(inst)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	if i, ok := inst.(PostCreate); ok {
		if err := i.PostCreate(); err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
		}
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

	resVal := reflect.New(reflect.SliceOf(reflect.PtrTo(b.t)))
	res := resVal.Interface()
	err = query.All(res)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	fmt.Printf("blablabl %#v\n", resVal.Elem().Interface())
	if resVal.Elem().Interface() == nil {
		c.JSON(http.StatusOK, make([]interface{}, 0, 0))
		return
	}
	c.JSON(http.StatusOK, res)
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
	err := coll.FindId(bson.ObjectIdHex(id)).One(inst)
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
		if err := i.PreUpdate(); err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
		}
	}

	coll := b.coll(c)
	err = coll.UpdateId(bson.ObjectIdHex(id), inst)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	pre := inst
	instValue = reflect.New(b.t)
	inst = instValue.Interface()
	err = coll.FindId(bson.ObjectIdHex(id)).One(inst)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	if i, ok := inst.(PostUpdate); ok {
		if err := i.PostUpdate(pre); err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
		}
	}
	c.JSON(http.StatusOK, inst)
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
	err := coll.FindId(bson.ObjectIdHex(id)).One(inst)
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
		if err := i.PreUpdate(); err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
		}
	}
	err = coll.UpdateId(bson.ObjectIdHex(id), bson.M{"$set": data})
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	pre := inst
	instValue = reflect.New(b.t)
	inst = instValue.Interface()
	err = coll.FindId(bson.ObjectIdHex(id)).One(inst)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	if i, ok := inst.(PostUpdate); ok {
		if err := i.PostUpdate(pre); err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
		}
	}
	c.JSON(http.StatusOK, inst)
}

func (b *Blueprint) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.AbortWithError(http.StatusNotFound, fmt.Errorf("not found %s", id))
		return
	}

	coll := b.coll(c)
	instValue := reflect.New(b.t)
	inst := instValue.Interface()
	err := coll.FindId(bson.ObjectIdHex(id)).One(inst)
	if err != nil {
		c.AbortWithError(http.StatusNotFound, err)
		return
	}

	if i, ok := inst.(PreDelete); ok {
		if err := i.PreDelete(); err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
		}
	}
	err = coll.RemoveId(bson.ObjectIdHex(id))
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}
	if i, ok := inst.(PostDelete); ok {
		if err := i.PostDelete(); err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
		}
	}

	c.Status(http.StatusNoContent)
}
