package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	"github.com/menachem554/Bookstore/entity"
	pb "github.com/menachem554/Bookstore/proto"
)

func main() {
	// Set up a connection to the server.
	conn, err := grpc.Dial("localhost:9090", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	client := pb.NewBookstoreClient(conn)

	// Set up a http server.
	gin.SetMode(gin.ReleaseMode)
	gin.ForceConsoleColor()
	r := gin.Default()

	// Get book by ID
	r.GET("/api/book/:id", func(c *gin.Context) {
		bookId := c.Param("id")
		// Contact the server and print out its response.
		req := &pb.GetBookReq{Id: bookId}

		res, err := client.GetBook(c, req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"result": fmt.Sprint(res.Book),
		})
	})

	r.POST("/api/book/", func(c *gin.Context) {
		// var book = PostBook
		book := entity.PostBook{}
		err := c.ShouldBindJSON(&book)

		book1 := &pb.Book{
			BookID:   book.BookID,
			BookName: book.BookName,
			Title:    book.Title,
			Author:   book.Author,
		}

		if err != nil {
			c.JSON(http.StatusBadRequest, err)
		}

		res, err := client.PostBook(context.Background(), &pb.PostBookReq{Book: book1})
		if err != nil {
			c.JSON(http.StatusInternalServerError, err)
			return
		}
		c.JSON(http.StatusOK, res)
	})

	r.PUT("/api/book/:id", func(c *gin.Context) {
		    bookId := c.Param("id")
			book := entity.PostBook{}

			err := c.ShouldBindJSON(&book)
			if err != nil {
				c.JSON(http.StatusBadRequest, err)
			}
	
			book1 := &pb.Book{
				BookID:   bookId,
				BookName: book.BookName,
				Title:    book.Title,
				Author:   book.Author,
			}
	
			res, err := client.UpdateBook(context.Background(), &pb.UpdateBookReq{Book: book1})
			if err != nil {
				c.JSON(http.StatusInternalServerError, err)
				return
			}
			result := res.GetBook()
		
			c.JSON(http.StatusOK, result)
	})

	r.DELETE("/api/book/:id", func(c *gin.Context) {
		bookId := c.Param("id")

		req := &pb.DeleteBookReq{Id: bookId}

		res, err := client.DeleteBook(c, req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"Success": fmt.Sprint(res.Success),
		})

	})

	log.Println("start client...........")
	// Run http server
	if err := r.Run(":9091"); err != nil {
		log.Fatalf("could not run server: %v", err)
	}
}
