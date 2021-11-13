package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/samuelventura/go-tree"
)

func api(node tree.Node) {
	dao := node.GetValue("dao").(Dao)
	endpoint := node.GetValue("endpoint").(string)
	gin.SetMode(gin.ReleaseMode) //remove debug warning
	router := gin.New()          //remove default logger
	router.Use(gin.Recovery())   //looks important
	rapi := router.Group("/api/key")
	rapi.GET("/list", func(c *gin.Context) {
		list := dao.ListKeys()
		c.JSON(200, list)
	})
	rapi.GET("/info/:name", func(c *gin.Context) {
		name := c.Param("name")
		row, err := dao.GetKey(name)
		if err != nil {
			c.JSON(400, fmt.Sprintf("err: %v", err))
			return
		}
		c.JSON(200, row)
	})
	rapi.POST("/delete/:name", func(c *gin.Context) {
		name := c.Param("name")
		err := dao.DelKey(name)
		if err != nil {
			c.JSON(400, fmt.Sprintf("err: %v", err))
			return
		}
		c.JSON(200, "ok")
	})
	rapi.POST("/enable/:name", func(c *gin.Context) {
		name := c.Param("name")
		err := dao.EnableKey(name, true)
		if err != nil {
			c.JSON(400, fmt.Sprintf("err: %v", err))
			return
		}
		c.JSON(200, "ok")
	})
	rapi.POST("/disable/:name", func(c *gin.Context) {
		name := c.Param("name")
		err := dao.EnableKey(name, false)
		if err != nil {
			c.JSON(400, fmt.Sprintf("err: %v", err))
			return
		}
		c.JSON(200, "ok")
	})
	rapi.POST("/add/:name", func(c *gin.Context) {
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
	listen, err := net.Listen("tcp", endpoint)
	if err != nil {
		log.Fatal(err)
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
