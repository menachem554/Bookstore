package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/joho/godotenv"
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
	Category    string `bson:category`
	Author   string `bson:author`
}

// Book to pb response
func BookToProto(data *BookInterface) *pb.Book {
	return &pb.Book{
		BookID:   data.BookID,
		BookName: data.BookName,
		Category:    data.Category,
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
		Category:    book.GetCategory(),
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
		"category":    req.GetBook().Category,
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
			Category:    req.GetBook().Category,
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
	// log if go crash, with the file name and line number
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// get env vars
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	// mongoLocal := os.Getenv("MONGO_LOCAL")
	mongoImage := os.Getenv("MONGO_LOCAL")

	// create the mongo context
	mongoCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// connect MongoDB
	fmt.Println("Connecting to MongoDB...")
	client, err := mongo.Connect(mongoCtx, options.Client().ApplyURI(mongoImage))
	if err != nil {
		log.Fatalf("Error Starting MongoDB Client: %v", err)
	}

	// check the connection
	err = client.Ping(mongoCtx, nil)
	if err != nil {
		log.Fatalf("Could not connect to MongoDB: %v\n", err)
	} else {
		fmt.Println("Connected to Mongodb")
	}

	bookDB = client.Database("Bookstore").Collection("books")

	fmt.Println("Starting Listener...")
	l, err := net.Listen("tcp", "0.0.0.0:9090")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	opts := []grpc.ServerOption{}
	s := grpc.NewServer(opts...)
	pb.RegisterBookstoreServer(s, &server{})

	// Start a GO Routine
	go func() {
		fmt.Println("Bookstore Server Started...")
		if err := s.Serve(l); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait to exit (Ctrl+C)
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)

	// Block the channel until the signal is received
	<-ch
	fmt.Println("Stopping Bookstore Server...")
	s.Stop()
	fmt.Println("Closing Listener...")
	l.Close()
	fmt.Println("Closing MongoDB...")
	client.Disconnect(mongoCtx)
	fmt.Println("All done!")
}