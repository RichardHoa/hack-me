# 🚀 Hack-Me Backend

## Running the server locally

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

    Start the server from the project root. You must set up doppler first for all the secrets management

    ```bash
    # Standard run
    make run

    ```

##  Running Tests

### Standard Tests
Run the complete test suite across all packages:
```bash
make test
```

### Fuzz Testing
To run a specific fuzz test (e.g., for user sign-up):
```bash
go test -run=TestUserRoutes -fuzz=FuzzUserSignUp
```


## Note for Database Schema Updates

When you create or modify a migration file, you **must reset both the main and testing databases** to prevent schema conflicts between the dev server and the test server.

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

## Security Implementation and Lessons

I approach security proactively rather than reacting to bugs. Before writing code, I performed threat modeling using `https://www.threatdragon.com` (the model is stored in this repo) and I consult `https://top10proactive.owasp.org/the-top-10/` to guide my defensive strategies. For access control, I implemented attribute-based access control (ABAC) instead of standard role-based access control. This ensures users can strictly only modify or delete resources they have created themselves. While I aim to classify all data sent and processed, I have currently completed classifying all stored data.

This data classification process has been a major learning experience. In all my `.sql` files, I’ve added comments to every column specifying its CIA requirements based on the `NIST - Guide for Mapping Types of Information and Information Systems to Security Categories`. This exercise forced me to think about the consequences of data exposure. A concrete example of this is the `challenge_response_vote_store`. Initially, I used a standard transaction to update the vote table and then the response table. However, when considering the "Integrity" aspect, I realized that if someone managed to manipulate the database directly, the application logic wouldn't catch it. To fix this, I switched to using a database trigger, ensuring the vote counts remain accurate regardless of how the data is touched.

I’ve also tightened my code against specific logic flaws like Mass Assignment. Since I repurpose the User struct to hold unmarshalled data from the frontend rather than creating a separate DTO, I realized my `decoder.DisallowUnknownFields()` configuration wasn't enough. I learned to add the `json:"-"` tag to sensitive fields like `UserID` to prevent them from being overwritten by a malicious payload. For secret management, I use `Doppler`, and I scan the repo with `TruffleHog` to ensure no secrets have leaked (the scan is currently clean).

To track vulnerabilities, I maintain a rigorous pipeline using Software Composition Analysis (SCA) with `govulncheck` and Static Application Security Testing (SAST) with `gosec`. I also use a combination of `syft` and `grype` to generate an SBOM and check for "ghost dependencies." On the testing side, I run handler-level tests where I spin up a test server instance to ensure malformed requests are rejected with proper error codes rather than causing a 500 server crash. I also use fuzz testing, which helped me realize the application currently crashes on NUL characters, this is documented and on the to-do list to fix.

Finally, I validate my configuration externally to minimize information leakage. I use `ZAP` to attack both the frontend and backend. I verify my SSL strength using `www.ssllabs.com` (while not absolute, it helps discover weaknesses). For headers, I follow `https://web.dev/articles/strict-csp` to minimize XSS risks and use `https://securityheaders.com/`, `https://developer.mozilla.org`, and `https://domsignal.com/secure-header-test` to double-check my work (the latter actually helped me find a few missing headers I have since added). For the database, since we use Supabase to host Postgres, I follow all their security recommendations

## TODO lists
- [ ] Check all the error message, a lot of them is vague and has no tracibility
- [ ] set up to reject NUL character
- [ ] password recovery
- [ ] email check
- [ ] login blocking after `n` attempts
- [ ] implement allowing app to consume only maximum 80% of resources, set timeout times
- [ ] instrumenting the server
- [ ] design test for library to capture their behaviour, make sure future version still do what we expect it to do
- [ ] design test for CSRF token
- [ ] Implment all the proper security header


## Observe behaviour in the project
- AI chat is very long since the AI response take 99% of the time. NOTE: this function is currently not used
