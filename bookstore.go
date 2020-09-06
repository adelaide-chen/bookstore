//{
//"id": "auto_generated_id",
//"name": "Harry Potter and the Prisoner of Azkaban",
//"author": "J K Rowling",
//"ISBN": "134238982734",
//"genre": "fantasy"
//}

// POST /books : Creates a new book.
// PUT /book/{id}: Updates a book.
// GET /books: Returns a list of books in the store.
// GET /book/{id}: Returns the book with id = {id}
// DELETE /book/{id}: Deletes the book with id = {id}
// DELETE /books: Deletes all books in the store

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"net/http"
	"strings"
	"time"
)

type Book struct {
	ID     primitive.ObjectID `bson:"_id,omitempty"`
	Name   string             `bson:"name,omitempty"`
	Author string             `bson:"author,omitempty"`
	ISBN   string             `bson:"ISBN,omitempty"`
	Genre  string             `bson:"genre,omitempty"`
}

func connect() (*mongo.Collection, context.Context, error) {
	client, err := mongo.NewClient(options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		return nil, nil, err
	}
	ctx, _ := context.WithTimeout(context.Background(), time.Second)
	err = client.Connect(ctx)
	if err != nil {
		return nil, nil, err
	}
	//defer client.Disconnect(ctx)

	inventory := client.Database("inventory").Collection("books")
	return inventory, ctx, nil
}

// POST /books : Creates a new book.
func create(b Book, db *mongo.Collection, ctx context.Context) error {
	_, err := db.InsertOne(ctx, b)
	return err
}

// PUT /book/{id}: Updates a book.
func update(pmtr string, b Book, db *mongo.Collection, ctx context.Context) error {
	id, _ := primitive.ObjectIDFromHex(pmtr)
	_, err := db.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{
			"$set": bson.M{
				"name": b.Name,
				"author": b.Author,
				"ISBN": b.ISBN,
				"genre": b.Genre,
			},
		},
	)
	return err
}

// GET /books: Returns a list of books in the store.
func listAll(db *mongo.Collection, ctx context.Context) ([]*Book, error) {
	var books []*Book
	find := options.Find()
	cur, err := db.Find(ctx, bson.D{}, find)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	for cur.Next(ctx) {
		var elem Book
		if err := cur.Decode(&elem); err != nil {
			fmt.Println(err)
		}
		books = append(books, &elem)
	}

	if err := cur.Err(); err != nil {
		log.Fatal(err)
	}

	return books, err
}

// GET /book/{id}: Returns the book with id = {id}
func getBook(ID string, db *mongo.Collection, ctx context.Context) (*Book, error) {
	var book *Book
	id, err := primitive.ObjectIDFromHex(ID)
	if err != nil {
		log.Fatal(err)
	}
	cur := db.FindOne(ctx, bson.M{"_id": id})
	if err := cur.Decode(&book); err != nil {
		log.Fatal(err)
	}
	return book, err
}

// DELETE /book/{id}: Deletes the book with id = {id}
func remove(ID string, db *mongo.Collection, ctx context.Context) {
	id, err := primitive.ObjectIDFromHex(ID)
	if err != nil {
		log.Fatal(err)
	}
	db.FindOneAndDelete(ctx, bson.M{"_id": id})
}

// DELETE /books: Deletes all books in the store
func clearAll(db *mongo.Collection, ctx context.Context) error {
	_, err := db.DeleteMany(ctx, bson.M{})
	return err
}

func status(w http.ResponseWriter, err int, string string) {
	w.WriteHeader(err)
	w.Write([]byte(string))
}

func books(w http.ResponseWriter, r *http.Request) {
	db, ctx, err := connect()
	if err != nil {
		status(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("content-type", "application/json")
	switch r.Method {
	case "POST":
		var b Book
		err := json.NewDecoder(r.Body).Decode(&b)
		if err != nil {
			status(w, http.StatusInternalServerError, err.Error())
		} else {
			w.WriteHeader(http.StatusCreated)
			create(b, db, ctx)
		}
	case "GET":
		res, err := listAll(db, ctx)
		if err != nil {
			status(w, http.StatusInternalServerError, err.Error())
		} else {
			for _, j := range res {
				temp, err := json.Marshal(*j)
				if err != nil {
					status(w, http.StatusInternalServerError, err.Error())
				} else {
					w.Write(temp)
				}
			}
		}
	case "DELETE":
		clearAll(db, ctx)
	default:
		status(w, http.StatusNotFound, "404 not found")
	}
}

func book(w http.ResponseWriter, r *http.Request) {
	db, ctx, err := connect()
	pmtr := strings.TrimPrefix(r.URL.Path, "/book/")
	if err != nil {
		status(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("content-type", "application/json")
	switch r.Method {
	case "PUT":
		var b Book
		err := json.NewDecoder(r.Body).Decode(&b)
		if err != nil {
			status(w, http.StatusInternalServerError, err.Error())
		} else {
			w.WriteHeader(http.StatusCreated)
			update(pmtr, b, db, ctx)
		}
	case "GET":
		b, _ := getBook(pmtr, db, ctx)
		temp, err := json.Marshal(*b)
		if err != nil {
			status(w, http.StatusInternalServerError, err.Error())
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write(temp)
		}
	case "DELETE":
		remove(pmtr, db, ctx)
	default:
		status(w, http.StatusNotFound, "404 not found")
	}
}

func main() {
	http.HandleFunc("/books", books)
	http.HandleFunc("/book/", book)
	log.Fatal(http.ListenAndServe(":8080", nil))
}