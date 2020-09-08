package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	//"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"net/http"
	"os"
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
	cl  *mongo.Collection
	ctx context.Context
}

type Server struct {
	start time.Time
	waitDuration time.Duration
}

var numberRequests = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "_bookstore_requests_served_total",
		Help: "Counts number of total requests",
	},
	[]string{"code"},
	// CPU usage query: rate(node_cpu_seconds_total{mode="user"}[1m])
	// Memory usage query: node_memory_active_bytes
)

var bookMetrics = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Namespace: "idk",
		Subsystem: "idk",
		Name: "_bookstore_application",
		Help: "Keeps track of number of books of genre and total",
	},
	[]string{"genre"},
)

func connect() (Database, error) {
	uri := fmt.Sprintf("mongodb://%s:27017", os.Getenv("DB"))
	client, err := mongo.NewClient(options.Client().ApplyURI(uri))
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
	var books = make([]Book, 0)
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
		numberRequests.WithLabelValues("500").Inc()
		return
	}
	w.Header().Set("content-type", "application/json")
	switch r.Method {
	case "POST":
		var b Book
		err := json.NewDecoder(r.Body).Decode(&b)
		if err != nil {
			status(w, http.StatusInternalServerError, err.Error())
			numberRequests.WithLabelValues("500").Inc()
		} else {
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte("Success"))
			numberRequests.WithLabelValues("201").Inc()
			bookMetrics.WithLabelValues(b.Genre).Inc()
			db.create(b)
		}
	case "GET":
		res, err := db.listAll()
		if err != nil {
			status(w, http.StatusInternalServerError, err.Error())
			numberRequests.WithLabelValues("500").Inc()
		} else if res == nil {
			w.Write([]byte(""))
			numberRequests.WithLabelValues("200").Inc()
		} else {
			temp, err := json.Marshal(res)
			if err != nil {
				status(w, http.StatusInternalServerError, err.Error())
				numberRequests.WithLabelValues("500").Inc()
			} else {
				w.Write(temp)
				numberRequests.WithLabelValues("200").Inc()
			}
		}
	case "DELETE":
		w.Write([]byte("Success"))
		numberRequests.WithLabelValues("200").Inc()
		bookMetrics.Reset()
		db.clearAll()
	default:
		status(w, http.StatusNotFound, "404 not found")
		numberRequests.WithLabelValues("404").Inc()
	}
}

func bookHandler(w http.ResponseWriter, r *http.Request) {
	db, err := connect()
	bookId := strings.TrimPrefix(r.URL.Path, "/book/")
	if err != nil {
		status(w, http.StatusInternalServerError, err.Error())
		numberRequests.WithLabelValues("500").Inc()
		return
	}
	w.Header().Set("content-type", "application/json")
	switch r.Method {
	case "PUT":
		var b Book
		err := json.NewDecoder(r.Body).Decode(&b)
		if err != nil {
			status(w, http.StatusInternalServerError, err.Error())
			numberRequests.WithLabelValues("500").Inc()
		} else {
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte("Success"))
			numberRequests.WithLabelValues("201").Inc()
			db.update(bookId, b)
		}
	case "GET":
		b, err := db.getBook(bookId)
		if err != nil {
			status(w, http.StatusInternalServerError, err.Error())
			numberRequests.WithLabelValues("500").Inc()
		}
		temp, err := json.Marshal(b)
		if err != nil {
			status(w, http.StatusInternalServerError, err.Error())
			numberRequests.WithLabelValues("500").Inc()
		} else {
			w.WriteHeader(http.StatusOK)
			numberRequests.WithLabelValues("200").Inc()
			w.Write(temp)
		}
	case "DELETE":
		w.Write([]byte("Success"))
		numberRequests.WithLabelValues("200").Inc()
		book, _ := db.getBook(bookId)
		bookMetrics.WithLabelValues(book.Genre).Dec()
		db.remove(bookId)
	default:
		status(w, http.StatusNotFound, "404 not found")
		numberRequests.WithLabelValues("404").Inc()
	}
}

//func (s *Server) livenessProbe(w http.ResponseWriter, r *http.Request) {
//	if time.Since(s.start) > s.waitDuration {
//		w.WriteHeader(http.StatusOK)
//	} else {
//		w.WriteHeader(http.StatusServiceUnavailable)
//	}
//}
//
//func (s *Server) readinessProbe(w http.ResponseWriter, r *http.Request) {
//	if time.Since(s.start) > s.waitDuration {
//		w.WriteHeader(http.StatusOK)
//	} else {
//		w.WriteHeader(http.StatusServiceUnavailable)
//	}
//}

func main() {
	//var server Server
	//server.start = time.Now()
	//server.waitDuration = 150
	prometheus.MustRegister(numberRequests)
	prometheus.MustRegister(bookMetrics)
	http.HandleFunc("/books", booksHandler)
	http.HandleFunc("/book/", bookHandler)
	http.Handle("/metrics", promhttp.Handler())
	//http.HandleFunc("/liveness", server.livenessProbe)
	//http.HandleFunc("/readiness", server.readinessProbe)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
