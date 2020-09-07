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

type Database struct {
	cl *mongo.Collection
	ctx context.Context
}

func connect() (Database, error) {
	client, err := mongo.NewClient(options.Client().ApplyURI("mongodb://mongodb:27017"))
	if err != nil {
		return Database{}, err
	}
	ctx, _ := context.WithTimeout(context.Background(), time.Second)
	err = client.Connect(ctx)
	if err != nil {
		return Database{}, err
	}
	//defer client.Disconnect(ctx)

	inventory := client.Database("inventory").Collection("books")
	return Database{inventory, ctx}, nil
}

// POST /books : Creates a new book.
func (db Database) create(b Book) error {
	_, err := db.cl.InsertOne(db.ctx, b)
	return err
}

// PUT /book/{id}: Updates a book.
func (db Database) update(pmtr string, book Book) error {
	id, _ := primitive.ObjectIDFromHex(pmtr)
	_, err := db.cl.UpdateOne(
		db.ctx,
		bson.M{"_id": id},
		bson.M{
			"$set": bson.M{
				"name": book.Name,
				"author": book.Author,
				"ISBN": book.ISBN,
				"genre": book.Genre,
			},
		},
	)
	return err
}

// GET /books: Returns a list of books in the store.
func (db Database) listAll() ([]Book, error) {
	var books []Book
	find := options.Find()
	cur, err := db.cl.Find(db.ctx, bson.D{}, find)
	if err != nil {
		return nil, err
	}
	defer cur.Close(db.ctx)

	for cur.Next(db.ctx) {
		var elem Book
		if err := cur.Decode(&elem); err != nil {
			fmt.Println(err)
		}
		books = append(books, elem)
	}

	if err := cur.Err(); err != nil {
		log.Fatal(err)
	}

	return books, err
}

// GET /book/{id}: Returns the book with id = {id}
func (db Database) getBook(ID string) (Book, error) {
	var book Book
	id, err := primitive.ObjectIDFromHex(ID)
	if err != nil {
		log.Fatal(err)
	}
	cur := db.cl.FindOne(db.ctx, bson.M{"_id": id})
	if err := cur.Decode(&book); err != nil {
		log.Fatal(err)
	}
	return book, nil
}

// DELETE /book/{id}: Deletes the book with id = {id}
func (db Database) remove(ID string) {
	id, err := primitive.ObjectIDFromHex(ID)
	if err != nil {
		log.Fatal(err)
	}
	db.cl.FindOneAndDelete(db.ctx, bson.M{"_id": id})
}

// DELETE /books: Deletes all books in the store
func (db Database) clearAll() error {
	_, err := db.cl.DeleteMany(db.ctx, bson.M{})
	return err
}

func status(w http.ResponseWriter, err int, string string) {
	w.WriteHeader(err)
	w.Write([]byte(string))
}

func booksHandler(w http.ResponseWriter, r *http.Request) {
	db, err := connect()
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
			db.create(b)
		}
	case "GET":
		res, err := db.listAll()
		if err != nil {
			status(w, http.StatusInternalServerError, err.Error())
		} else {
			temp, err := json.Marshal(res)
			if err != nil {
				status(w, http.StatusInternalServerError, err.Error())
			} else {
				w.Write(temp)
			}
		}
	case "DELETE":
		db.clearAll()
	default:
		status(w, http.StatusNotFound, "404 not found")
	}
}

func bookHandler(w http.ResponseWriter, r *http.Request) {
	db, err := connect()
	bookId := strings.TrimPrefix(r.URL.Path, "/book/")
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
			db.update(bookId, b)
		}
	case "GET":
		b, err := db.getBook(bookId)
		if err != nil {
			status(w, http.StatusInternalServerError, err.Error())
		}
		temp, err := json.Marshal(b)
		if err != nil {
			status(w, http.StatusInternalServerError, err.Error())
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write(temp)
		}
	case "DELETE":
		db.remove(bookId)
	default:
		status(w, http.StatusNotFound, "404 not found")
	}
}

func main() {
	http.HandleFunc("/books", booksHandler)
	http.HandleFunc("/book/", bookHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
