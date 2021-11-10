package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/menachem554/Bookstore/proto"
)

type server struct {
	pb.UnimplementedBookstoreServer
}

// mongo setting
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

// Book to pb response
func BookToProto(data *BookInterface) *pb.Book {
	return &pb.Book{
		BookID:   data.BookID,
		BookName: data.BookName,
		Title:    data.Title,
		Author:   data.Author,
	}
}

// Create new book
func (s *server) PostBook(ctx context.Context, req *pb.BookRequest) (*pb.BookResponse, error) {
	// Get the request
	book := req.GetBook()

	data := BookInterface{
		BookID:   book.GetBookID(),
		BookName: book.GetBookName(),
		Title:    book.GetTitle(),
		Author:   book.GetAuthor(),
	}

	// Insert the data into the database
	res, err := bookDB.InsertOne(mongoCtx, data)
	if err != nil {
		return nil,
			status.Errorf(codes.Internal, fmt.Sprintf(" Internal Error: %v", err))
	}

	fmt.Printf("Successfully inserted NEW Book into book collection!, %v \n", res)

	return &pb.BookResponse{Book: BookToProto(&data)}, nil
}

// Read book by ID of the book
func (s *server) GetBook(ctx context.Context, req *pb.GetBookReq) (*pb.BookResponse, error) {
	// Get ID of the book
	bookID := req.GetId()

	res := bookDB.FindOne(ctx, bson.M{"bookid": bookID})
	data := &BookInterface{}

	// decode and Check for error
	if err := res.Decode(data); err != nil {
		return nil,
			status.Errorf(codes.NotFound, fmt.Sprintf("Cannot found book with the ID: %v", err))
	}
	fmt.Println("Get book result", res)
	return &pb.BookResponse{Book: BookToProto(data)}, nil
}

// Update book
func (s *server) UpdateBook(ctx context.Context, req *pb.BookRequest) (*pb.BookResponse, error) {
	// Get the request
	book := req.GetBook()

	data := bson.M{
		"bookid":   req.GetBook().BookID,
		"bookname": req.GetBook().BookName,
		"title":    req.GetBook().Title,
		"author":   req.GetBook().Author,
	}
	// insert the changes
	bookDB.FindOneAndUpdate(
		ctx,
		bson.M{"bookid": req.GetBook().BookID},
		bson.M{"$set": data})

	fmt.Println("the decode result is:", book)

	return &pb.BookResponse{
		Book: &pb.Book{
			BookID:   req.GetBook().BookID,
			BookName: req.GetBook().BookName,
			Title:    req.GetBook().Title,
			Author:   req.GetBook().Author,
		},
	}, nil
}

//
func (s *server) DeleteBook(ctx context.Context, req *pb.GetBookReq) (*pb.DeleteBookRes, error) {
	// Get ID of the book
	bookID := req.GetId()

	res, err := bookDB.DeleteOne(ctx, bson.M{"bookid": bookID})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Book deleted: ", res.DeletedCount)

	return &pb.DeleteBookRes{Deleted: res.DeletedCount}, nil
}

// Get all books in the Collection
func (s *server) GetAllBooks(req *pb.GetAllReq, stream pb.Bookstore_GetAllBooksServer) error {
	fmt.Println("\n list of all book start stream")
	cur, err := bookDB.Find(context.Background(), bson.D{})
	if err != nil {
		return status.Errorf(codes.Internal, fmt.Sprintf("Unknown Internal Error: %v", err))
	}

	defer cur.Close(context.Background())

	for cur.Next(context.Background()) {
		data := &BookInterface{}
		if err := cur.Decode(data); err != nil {
			return status.Errorf(codes.Internal, fmt.Sprintf("Cannot decoding data: %v", err))
		}
		stream.Send(&pb.BookResponse{Book: BookToProto(data)})
	}
	if err = cur.Err(); err != nil {
		return status.Errorf(codes.Internal, fmt.Sprintf("Unknown Internal Error: %v", err))
	}
	return nil
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
		pb.RegisterBookstoreServer(s, &server{})
	
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
