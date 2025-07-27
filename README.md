# TODO
[ ] implement password recovery
[ ] implement email check
[ ] implement login blocking after `n` attempts
[ ] implement allowing app to consume only maximum 80% of resources, set timeout times


# CODE

```bash
docker buildx build --platform linux/amd64 -t hack-me/backend .

docker tag hack-me/backend:latest 004843574486.dkr.ecr.ap-southeast-1.amazonaws.com/hack-me/backend:latest

docker push 004843574486.dkr.ecr.ap-southeast-1.amazonaws.com/hack-me/backend:latest
```


# 🚀 Hack-Me Backend

| | |
| :--- | :--- |
| **Start Date** | 26/06/2025 |
| **Finish Date** | 19/07/2025 |
| **Frontend Repository** | [**github.com/RichardHoa/hack-me-frontend**](https://github.com/RichardHoa/hack-me-frontend) |

---

## 📋 Table of Contents

- [🐳 **Run with Docker**](#-run-the-whole-app-with-docker) (Recommended for a quick start)
- [🧑‍💻 **Run Locally**](#-running-the-server-locally) (For active development)
- [🧪 **Run Tests**](#-running-tests)
- [🏗️ **Project Structure**](#️-project-structure)
- [📝 **Important Notes**](#-important-notes)

---

## 🐳 Run the whole app with Docker

This is the fastest way to get the entire application stack running without installing Go locally.

1.  **Configure Your Environment**

    First, copy the example environment file:
    ```bash
    cp .env.example .env
    ```
    > **Important:** Open the new `.env` file and ensure the `DEV_MODE` variable is **NOT** set to `LOCAL`. It should be empty or set to another value.

2.  **Launch the Application**

    Build and run the containers in detached mode:
    ```bash
    docker-compose up --build -d
    ```

<details>
<summary><strong>View Logs & Stop the App</strong></summary>

To view the backend server logs in real-time:
```bash
docker-compose logs -f app
```

When you are finished, stop and remove all containers, networks, and volumes:
```bash
docker-compose down -v
```

</details>

---

## 🧑‍💻 Running the server locally

Follow these steps for local development and debugging.

1.  **Set Development Mode**

    In your `.env` file, set the `DEV_MODE` to `LOCAL`:
    ```env
    DEV_MODE="LOCAL"
    ```

2.  **Download Dependencies**

    Fetch all the required Go modules:
    ```bash
    go mod download
    ```

3.  **Start the Database**

    Use Docker to run only the database service:
    ```bash
    docker-compose up --build -d db
    ```
    > **Note:** If you run `docker-compose up --build` without specifying the `db` service, you will see the `app` container start and then crash. This is expected behavior in local mode, as we will run the Go server manually on our host machine.

4.  **Run the Go Server**

    Start the server from the project root. You can use either the standard Go command or `air` for live reloading.

    ```bash
    # Standard run
    go run main.go

    # Or, for live-reloading with Air
    air
    ```

---

## 🧪 Running Tests

### Standard Tests
Run the complete test suite across all packages:
```bash
go test ./...
```

### Fuzz Testing
To run a specific fuzz test (e.g., for user sign-up):
```bash
go test -run=TestUserRoutes -fuzz=FuzzUserSignUp -parallel=4
```

---

## 🏗️ Project Structure

The application follows a layered architecture to separate concerns. The data flows from the entry point to the database as follows:

-   `main.go`
    -   The entry point of the application. It initializes the database connection, runs migrations, and starts the HTTP server.
-   `app.go`
    -   Defines the core `application` struct. This struct holds dependencies like the database store and is passed to the routers to create handlers.
-   `handler/`
    -   Contains functions that directly handle incoming HTTP requests, parse data, and call the appropriate services.
-   `store/`
    -   Contains all database logic. It's responsible for executing SQL queries and managing data persistence.
-   `migrations/`
    -   Contains all `.sql` files for database schema migrations. These are run automatically when the server starts up to ensure the database schema is up-to-date.

---

## 📝 Important Notes

### ⚠️ Database Schema Updates

When you create or modify a migration file, you **must reset both the main and testing databases** to prevent schema conflicts.

1.  **Connect to each PostgreSQL instance:**

    ```bash
    # Connect to the main database
    psql -U postgres -h localhost -p 5432

    # Connect to the testing database in a separate terminal
    psql -U postgres -h localhost -p 5433
    ```

2.  **Inside each `psql` shell**, run the following SQL commands to completely wipe and recreate the schema:

    ```sql
    DROP SCHEMA public CASCADE;
    CREATE SCHEMA public;
    ```
    This ensures that both your development and testing environments are perfectly in sync with the latest schema.
