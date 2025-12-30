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

## 📝 Important Notes

### ⚠️ Database Schema Updates

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

## Security
- Doppler for secret management
- Threat modelling using https://www.threatdragon.com. The model is stored in this repo
- Security testing in the test file, I test at the handler level, spin up a test server instance and make sure the malformed request get rejected with the right error code and not 500 (which means the server crash)
- you can see in all of my `.sql` files I always have comment for each column, this comment specify the CIA framework as specified in `NIST - Guide for Mapping Types of Information and Information Systems to Security Categories`. 
- While adding CIA comment to the database, I realize it has a very interesting insights of letting you know which information is pubic, which one is private but with some digging the user can find it. it gives me a much clearer picture of what will happen and force me to think about the consequences with that data. One concrte example of this is when implementing the counting mechanism for `challenge_response_vote_store`. I initially do a transactions, I would update the `challenge_response_vote` table and then update the `challenge_response`, where the votes digits are stored. But when I consider the Integrity aspect and think about what if someone manage to change the number in the database, specifically the `challenge_response_vote` table, then that would have no effect on the store on the `challenge_response` table, thus I switch to using a trigger
- After reading the mass assignment vulerability, I realize that some of my struct has UserID in the struct that get unmarshal from the json data from the client, I've already enabled 	decoder.DisallowUnknownFields()  but since there is no  `json:"-"`, it can still be subjected to Mass Assignment since in the user route I repurpose the User struct to contains the unmarshal data from frontend instead of creating a DTO, so I've learnt to add the json:"-" tag, which prevents the UserID to be assigned whatsoever
- I want the app to be secure as posible, so to track all the vulnerabilities, we use Software Composition Analysis (SCA) `govulncheck` and  Static Application Security Testing (SAST) of `gosec`, I also use a combination of syft_and_grype as SBOM and checker to make sure that there is no ghost depencecies 
- I also consult https://top10proactive.owasp.org/the-top-10/, which help me massively in finding more security releated bugs and doing all the toolings mention here
- I use ZAP to attack the frontend and backend to minimize information leakage and obvious error 
- We have fuzz testing to check whether the server crash at any input, turn out it crashes at NUL character, but we don't have the time to fix it yet
- I use www.ssllabs.com to check the SSl strength of the website, while the tool grade is not absolute, it's still good to check it to maximize our chances of discovering weakness
- I use https://securityheaders.com/ to check to see if the security headers is enough, I follow https://web.dev/articles/strict-csp to minimize the chance of being XSS, it's very nice that our frontend framework make it convient for me to implement csp
- I also use https://developer.mozilla.org to test the configuration again
- I also use https://domsignal.com/secure-header-test, turn out there are a few headers that I have not implmented yet
- I use TruffleHog to scan for all the potential secrets leak in the repo, there is not any according to the scan
- Database security: we use superbase to host our postgres, so I follow all the superbase security recommendation so far
- Using the built-in fuzz test help me readlize my application is not protected against NUL character, so I add it into the future to-do lists.

In app security:
- We use attribute-based access control, which means user can only delete or modify what they've created and not others, we do not use role based access control, as recommended by OWASP
- We are supposed to Classify the data sent, processed, and stored in your system, but I've only manage to classify the data being stored


### TODO lists
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
