# Hack-Me Backend

**Starting Date:** 26/06/2025  
**Finish Date:** 19/07/2025
**Frontend Repository:** [Frontend Repo](https://github.com/RichardHoa/hack-me-frontend) 

## 🐳 Run the whole app with Docker

If you do not have Go installed locally, you can run the app using Docker:

```bash
docker-compose up --build -d
```

To view backend logs:

```bash
docker-compose logs -f app
```

To stop and remove containers and volumes after you're done:

```bash
docker-compose down -v
```

## 🧑‍💻 Running the server locally

1. Update `.env` file:

```
DEV_MODE="LOCAL"
```

2. Download Go modules:

```bash
go mod download
```

3. Running database server:
```bash
docker-compose up --build
```
> Note: When using `docker-compose up --build`, you'll see the app container crash — this is expected when using local mode, since we'll run the server manually.

4. Run the server:

```bash
go run main.go
# or, if you prefer air:
air
```

## 🧪 Running Tests

Run the full test suite:

```bash
go test ./...
```


## 📝 Important Notes

### ⚠️ Database Schema Updates
When making changes to the database schema, **you must reset both databases** to maintain consistency.  
Run the following commands in your terminal:

```bash
# Connect to both Postgres instances
psql -U postgres -h localhost -p 5432
psql -U postgres -h localhost -p 5433
```

Once inside each, drop and recreate the `public` schema:

```sql
DROP SCHEMA public CASCADE;
CREATE SCHEMA public;
```

> This ensures both environments are in sync and eliminates residual schema conflicts.
