package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

var bookID string

func TestBooksHandler(t *testing.T) {
	rawInput := `{"Name": "Harry Potter and the Prisoner of Azkaban","Author": "J K Rowling","ISBN": "134238982734","Genre": "fantasy"}`
	t.Run("deletes all books via DELETE", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", "https://localhost:8080", nil)
		res := httptest.NewRecorder()

		booksHandler(res, req)

		if res.Code != http.StatusOK {
			t.Errorf("DELETE error from clearAll %v", res.Code)
		}
	})
	t.Run("creates a new book via POST", func(t *testing.T) {
		input := strings.NewReader(rawInput)
		req := httptest.NewRequest("POST", "https://localhost:8080", input)
		res := httptest.NewRecorder()

		booksHandler(res, req)

		if res.Code != http.StatusCreated {
			t.Errorf("POST error from create %v", res.Code)
		}
	})
	t.Run("returns a list of all books via GET", func(t *testing.T) {
		req := httptest.NewRequest("GET", "https://localhost:8080", nil)
		res := httptest.NewRecorder()

		booksHandler(res, req)

		if res.Code != http.StatusOK {
			t.Errorf("GET error from listAll %v", res.Code)
		} else {
			var books []Book
			_ = json.NewDecoder(res.Body).Decode(&books)
			if len(books) == 1 {
				bookID = books[0].ID.Hex()
				fmt.Println(books)
				fmt.Println(rawInput)
			}

		}
	})
}

func TestBookHandler (t *testing.T) {
	url := fmt.Sprintf("https://localhost:8080/book/%s", bookID)

	t.Run("updates book via PUT", func(t *testing.T) {
		input := strings.NewReader(`{"name": "another one","author": "J K Rowling","ISBN": "134238982734","genre": "fantasy"}`)
		req := httptest.NewRequest("PUT", url, input)
		res := httptest.NewRecorder()

		bookHandler(res, req)

		if res.Code != http.StatusCreated {
			t.Errorf("PUT error from update %v", res.Code)
		}
	})

	t.Run("return book info via GET", func(t *testing.T) {
		req := httptest.NewRequest("GET", url, nil)
		res := httptest.NewRecorder()

		bookHandler(res, req)

		if res.Code != http.StatusOK {
			t.Errorf("GET error from getBook %v", res.Code)
		}
	})

	t.Run("delete book via DELETE", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", url, nil)
		res := httptest.NewRecorder()

		bookHandler(res, req)

		if res.Code != http.StatusOK {
			t.Errorf("DELETE error from remove %v", res.Code)
		}
	})
}

