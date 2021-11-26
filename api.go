package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/samuelventura/go-tree"
)

func api(node tree.Node) {
	dao := node.GetValue("dao").(Dao)
	ships := node.GetValue("ships").(Ships)
	endpoint := node.GetValue("endpoint").(string)
	gin.SetMode(gin.ReleaseMode) //remove debug warning
	router := gin.New()          //remove default logger
	router.Use(gin.Recovery())   //looks important
	rkapi := router.Group("/api/key")
	rkapi.GET("/list", func(c *gin.Context) {
		list := dao.ListKeys()
		c.JSON(200, list)
	})
	rkapi.GET("/info/:name", func(c *gin.Context) {
		name := c.Param("name")
		row, err := dao.GetKey(name)
		if err != nil {
			c.JSON(400, fmt.Sprintf("err: %v", err))
			return
		}
		c.JSON(200, row)
	})
	rkapi.POST("/delete/:name", func(c *gin.Context) {
		name := c.Param("name")
		err := dao.DelKey(name)
		if err != nil {
			c.JSON(400, fmt.Sprintf("err: %v", err))
			return
		}
		c.JSON(200, "ok")
	})
	rkapi.POST("/enable/:name", func(c *gin.Context) {
		name := c.Param("name")
		err := dao.EnableKey(name, true)
		if err != nil {
			c.JSON(400, fmt.Sprintf("err: %v", err))
			return
		}
		c.JSON(200, "ok")
	})
	rkapi.POST("/disable/:name", func(c *gin.Context) {
		name := c.Param("name")
		err := dao.EnableKey(name, false)
		if err != nil {
			c.JSON(400, fmt.Sprintf("err: %v", err))
			return
		}
		c.JSON(200, "ok")
	})
	rkapi.POST("/add/:name", func(c *gin.Context) {
		name := c.Param("name")
		file, err := c.FormFile("file")
		if err != nil {
			c.JSON(400, fmt.Sprintf("err: %v", err))
			return
		}
		src, err := file.Open()
		if err != nil {
			c.JSON(400, fmt.Sprintf("err: %v", err))
			return
		}
		defer src.Close()
		buf := bytes.NewBuffer(nil)
		_, err = io.Copy(buf, src)
		if err != nil {
			c.JSON(400, fmt.Sprintf("err: %v", err))
			return
		}
		err = dao.AddKey(name, buf.String())
		if err != nil {
			c.JSON(400, fmt.Sprintf("err: %v", err))
			return
		}
		c.JSON(200, "ok")
	})
	skapi := router.Group("/api/ship")
	skapi.GET("/count", func(c *gin.Context) {
		count := dao.CountShips()
		c.JSON(200, count)
	})
	skapi.GET("/count/enabled", func(c *gin.Context) {
		count := dao.CountEnabledShips()
		c.JSON(200, count)
	})
	skapi.GET("/count/disabled", func(c *gin.Context) {
		count := dao.CountDisabledShips()
		c.JSON(200, count)
	})
	skapi.GET("/info/:name", func(c *gin.Context) {
		name := c.Param("name")
		row, err := dao.GetShip(name)
		if err != nil {
			c.JSON(400, fmt.Sprintf("err: %v", err))
			return
		}
		c.JSON(200, row)
	})
	skapi.GET("/state/:name", func(c *gin.Context) {
		name := c.Param("name")
		row, err := dao.ShipState(name)
		if err != nil {
			c.JSON(400, fmt.Sprintf("err: %v", err))
			return
		}
		c.JSON(200, row)
	})
	//ensure port is added to node before ship gets added to ships
	skapi.GET("/status/:name", func(c *gin.Context) {
		name := c.Param("name")
		node := ships.Get(name)
		port := -1
		ip := ""
		id := ""
		key := ""
		hostname := ""
		if node != nil {
			id = node.Name()
			port = node.GetValue("proxy").(int)
			ip = node.GetValue("export").(string)
			key = node.GetValue("key").(string)
			hostname = node.GetValue("hostname").(string)
		}
		c.JSON(200, gin.H{"ip": ip, "port": port, "key": key,
			"host": hostname, "id": id, "name": name})
	})
	skapi.POST("/close/:name", func(c *gin.Context) {
		name := c.Param("name")
		node := ships.Get(name)
		if node == nil {
			c.JSON(400, "err: ship not connected")
			return
		}
		node.Close()
		c.JSON(200, "ok")
	})
	skapi.POST("/add/:name", func(c *gin.Context) {
		name := c.Param("name")
		err := dao.AddShip(name)
		if err != nil {
			c.JSON(400, fmt.Sprintf("err: %v", err))
			return
		}
		c.JSON(200, "ok")
	})
	skapi.POST("/port/:name/:port", func(c *gin.Context) {
		name := c.Param("name")
		port := c.Param("port")
		pv, err := strconv.ParseUint(port, 10, 16)
		if err != nil {
			c.JSON(400, fmt.Sprintf("err: %v", err))
			return
		}
		err = dao.PortShip(name, int(pv))
		if err != nil {
			c.JSON(400, fmt.Sprintf("err: %v", err))
			return
		}
		c.JSON(200, "ok")
	})
	skapi.POST("/enable/:name", func(c *gin.Context) {
		name := c.Param("name")
		err := dao.EnableShip(name, true)
		if err != nil {
			c.JSON(400, fmt.Sprintf("err: %v", err))
			return
		}
		c.JSON(200, "ok")
	})
	skapi.POST("/disable/:name", func(c *gin.Context) {
		name := c.Param("name")
		err := dao.EnableShip(name, false)
		if err != nil {
			c.JSON(400, fmt.Sprintf("err: %v", err))
			return
		}
		c.JSON(200, "ok")
	})
	listen, err := net.Listen("tcp", endpoint)
	if err != nil {
		log.Panicln(err)
	}
	node.AddCloser("listen", listen.Close)
	port := listen.Addr().(*net.TCPAddr).Port
	log.Println("port api", port)
	server := &http.Server{
		Addr:    endpoint,
		Handler: router,
	}
	node.AddProcess("server", func() {
		err = server.Serve(listen)
		if err != nil {
			log.Println(endpoint, port, err)
		}
	})
}
