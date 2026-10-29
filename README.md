# Go articles app

The app supports MySQL and MongoDB. Choose the database with environment
variables; database credentials should not be committed to the repository.

## Run with MongoDB

1. Create a MongoDB database (for example, an Atlas M0 free cluster) and copy
   its connection URI.
2. Set the environment variables:

   ```text
   DATABASE_DRIVER=mongodb
   MONGODB_URI=<your MongoDB connection URI>
   MONGODB_DATABASE=golang
   ```

   In PowerShell, for a local run:

   ```powershell
   $env:DATABASE_DRIVER = "mongodb"
   $env:MONGODB_URI = "<your MongoDB connection URI>"
   $env:MONGODB_DATABASE = "golang"
   go run .
   ```

3. If using MongoDB Atlas, add your app host to the cluster's network access
   list and create a database user with permissions to read and write the
   selected database.

The app creates the `articles` collection automatically when the first article
is saved. MongoDB article IDs are ObjectId hex strings.

## Run with MySQL

Set `DATABASE_DRIVER=mysql` and `MYSQL_DSN` to a MySQL DSN that can read and
write the `articles` table. The required columns are `id`, `title`, `anons`,
and `full_text`.

## Deploy on Render

Create a Go **Web Service**, set the build command to `go build -o app .` and
the start command to `./app`. Set `DATABASE_DRIVER=mongodb`,
`MONGODB_URI`, and `MONGODB_DATABASE` under the service's environment settings.
The app listens on Render's `PORT` environment variable; locally it defaults
to port 8081.
