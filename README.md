# 🚀 Hack-Me Backend

---
## Warning
All the installation instructions is kind of obsolete since I move to [doppler](https://www.doppler.com/) as a secret manager rather than using .env file, as such a lot of code won't work because it lack the environment variables

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
    make up
    ```

To view the backend server logs in real-time:
```bash
docker-compose logs
```

When you are finished, stop and remove all containers, networks, and volumes:
```bash
make down
```

NOTE: Some of the Makefile commands cannot be run in your machine, this github repo does not have a CI/CD pipeline so I use Makefile to automate some deploy

---

## 🧑‍💻 Running the server locally

Follow these steps for local development and debugging.

1. **Comment out the code that build go app in the docker-compose, we'll run the server manually**

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

## Security
- Doppler for secret management
- Threat modelling using https://www.threatdragon.com. The model is stored in this repo
- Security testing in the test file, I test at the handler level, spin up a test server instance and make sure the malformed request get rejected with the right error code and not 500 (which means the server crash)
- While adding CIA comment to the database, I realize it has a very interesting insights of letting you know which information is pubic, which one is private but with some digging the user can find it. it gives me a much clearer picture of what will happen and force me to think about the consequences with that data. One concrte example of this is when implementing the counting mechanism for `challenge_response_vote_store`. I initially do a transactions, I would update the `challenge_response_vote` table and then update the `challenge_response`, where the votes digits are stored. But when I consider the Integrity aspect and think about what if someone manage to change the number in the database, specifically the `challenge_response_vote` table, then that would have no effect on the store on the `challenge_response` table, thus I switch to using a trigger
- I want the app to be secure as posible, so to track all the vulnerabilities, we use Software Composition Analysis (SCA) `govulncheck` and  Static Application Security Testing (SAST) of `gosec` 
- I use ZAP to attack the frontend and backend to minimize information leakage and obvious error 
- We have fuzz testing to check whether the server crash at any input, turn out it crashes at NUL character, but we don't have the time to fix it yet
- I use www.ssllabs.com to check the SSl strength of the website, while the tool grade is not absolute, it's still good to check it to maximize our chances of discovering weakness
- I use https://securityheaders.com/ to check to see if the security headers is enough, I follow https://web.dev/articles/strict-csp to minimize the chance of being XSS, it's very nice that our frontend framework make it convient for me to implement csp
- I also use https://developer.mozilla.org to test the configuration again
- I also use https://domsignal.com/secure-header-test, turn out there are a few headers that I have not implmented yet
- Database security: we use superbase to host our postgres, so I follow all the superbase security recommendation so far





## TODO lists
- [ ] set up to reject NUL character
- [ ] password recovery
- [ ] email check
- [ ] login blocking after `n` attempts
- [ ] implement allowing app to consume only maximum 80% of resources, set timeout times
- [ ] instrumenting the server
- [ ] design test for library to capture their behaviour, make sure future version still do what we expect it to do
- [ ] design test for CSRF token
- [ ] Implment all the proper security header


# Behaviour
- AI chat is very long since the AI response take 99% of the time
