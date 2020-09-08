from locust import HttpUser, task, between


class User(HttpUser):
    wait_time = between(1, 2)

    @task
    def create_task(self):
        book = {
            "Name": "Harry Potter and the Prisoner of Azkaban",
            "Author": "J K Rowling",
            "ISBN": "134238982734",
            "Genre": "fantasy"
        }
        self.client.post("books", json=book)

    @task
    def list_all_task(self):
        self.client.get("books")

    @task
    def clear_all_task(self):
        self.client.delete("books")

    def get_book(self):
        resp = self.client.get("books")
        books = resp.json()
        if len(books) > 0:
            return books[-1]
        else:
            return None

    @task
    def update_task(self):
        last = self.get_book()
        if last is not None:
            first = last["ID"]
            last["name"] = "Harry Potter 2.0"
            self.client.put(f"book/{first}", json=last)

    @task
    def get_book_task(self):
        last = self.get_book()
        if last is not None:
            self.client.get(f"book/{last['ID']}")

    @task
    def remove_task(self):
        last = self.get_book()
        if last is not None:
            self.client.delete(f"book/{last['ID']}")
