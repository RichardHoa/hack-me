# Very cool project

starting date: 26/06/2025

# Important note

## When change database schema, make sure to 
```bash
psql -U postgres -h localhost -p 5432
psql -U postgres -h localhost -p 5433
```
 and 
```bash
DROP SCHEMA public CASCADE; CREATE SCHEMA public;
```
BOTH of the table, this will resolve any consistency between the table
