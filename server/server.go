package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/menachem554/Bookstore/proto"
)

type Bookstore struct {
	pb.UnimplementedBookstoreServer
}

// mongo seeting
var db *mongo.Client
var bookDB *mongo.Collection
var mongoCtx context.Context

// book interface
type BookInterface struct {
	BookID   string `bson:bookId`
	BookName string `bson:bookName`
	Title    string `bson:title`
	Author   string `bson:author`
}

func (s *Bookstore) PostBook(ctx context.Context, req *pb.PostBookReq) (*pb.PostBookRes, error) {
	// Get the request
	book := req.GetBook()

	data := BookInterface{
		BookID:   book.GetBookID(),
		BookName: book.GetBookName(),
		Title:    book.GetTitle(),
		Author:   book.GetAuthor(),
	}

	// Insert the data into the database
	bookResult, err := bookDB.InsertOne(mongoCtx, data)
	if err != nil {
		log.Fatal(err)
	}

	oid := bookResult.InsertedID.(primitive.ObjectID).Hex()
	fmt.Printf("Successfully inserted NEW Book into book collection!, Oid: %v \n", oid)

	return &pb.PostBookRes{Oid: oid}, nil
}

func (s *Bookstore) GetBook(ctx context.Context, req *pb.GetBookReq) (*pb.GetBookRes, error) {

	searchResult := bookDB.FindOne(ctx, bson.M{"bookid": req.GetId()})
	book := BookInterface{}

	// decode and Check for error
	if err := searchResult.Decode(&book); err != nil {
		return nil,
		status.Errorf(codes.NotFound, fmt.Sprintf("Could not find book with bookID %s: %v", req.GetId(), err))
	}

	fmt.Println("Get book result", book)
	return &pb.GetBookRes{
		Book: &pb.Book{
			BookID:   book.BookID,
			BookName: book.BookName,
			Title:    book.Title,
			Author:   book.Author,
		},
	}, nil
}
func (s *Bookstore) UpdateBook(ctx context.Context, req *pb.UpdateBookReq) (*pb.UpdateBookRes, error) {
	// 
	updateBook := bson.M{
		"bookid":   req.GetBook().BookID,
		"bookname": req.GetBook().BookName,
		"title":    req.GetBook().Title,
		"author":   req.GetBook().Author,
	}
	// insert the changes
	result := bookDB.FindOneAndUpdate(
		ctx,
		bson.M{"bookid": req.GetBook().BookID},
		bson.M{"$set": updateBook})

	book := BookInterface{}
	result.Decode(&book)

	fmt.Println("the decode result is:", book)

	return &pb.UpdateBookRes{
		Book: &pb.Book{
			BookID:   book.BookID,
			BookName: book.BookName,
			Title:    book.Title,
			Author:   book.Author,
		},
	}, nil
}
func (s *Bookstore) DeleteBook(ctx context.Context, req *pb.DeleteBookReq) (*pb.DeleteBookRes, error) {
	// 
	delete, err := bookDB.DeleteOne(ctx, bson.M{"bookid":req.GetId()})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Book deleted: ", delete.DeletedCount)

	return &pb.DeleteBookRes{Success: true,}, nil
}

func main() {

	// connect on port 9090
	listener, err := net.Listen("tcp", ":9090")

	if err != nil {
		log.Fatalf("Unable to listen on port :9090: %v", err)
	}

	// gRPC options and Register
	opts := []grpc.ServerOption{}
	s := grpc.NewServer(opts...)
	pb.RegisterBookstoreServer(s, &Bookstore{})

	// Connect to mongoDB
	mongoCtx = context.Background()
	db, err = mongo.Connect(mongoCtx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatal(err)
	}

	// check the connection
	err = db.Ping(mongoCtx, nil)
	if err != nil {
		log.Fatalf("Could not connect to MongoDB: %v\n", err)
	} else {
		fmt.Println("Connected to Mongodb")
	}

	// Define the db and collection
	bookDB = db.Database("Bookstore").Collection("books")

	// Start the server in a child routine
	go func() {
		if err := s.Serve(listener); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()
	fmt.Println("Server successfully started on port :9090")

	// Create a channel to receive OS signals
	c := make(chan os.Signal)

	// Relay os.Interrupt to our channel (os.Interrupt = CTRL+C)
	signal.Notify(c, os.Interrupt)

	// Block main routine until a signal is received until CTRL+C was detected
	<-c

	// After receiving CTRL+C Properly stop the server
	fmt.Println("\nStopping the server...")
	s.Stop()
	listener.Close()
	fmt.Println("Closing MongoDB connection")
	db.Disconnect(mongoCtx)
	fmt.Println("Done.")
}
