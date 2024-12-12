<h1>
Telegram Bot for Providing Books in FB2 and EPUB Formats from a Personal Library in INPX Format Packed in ZIP Archives in FB2 and EPUB Formats
</h1>

[on russian](https://github.com/almadern/gobooks/blob/main/README_rus.md)

### The bot works with a home INPX library and book archives.
- It can store and process data in three different formats:
    - file (without a database, the entire library is checked with each request. This is the slowest operation mode)
    - sqlite (works with and stores data in an SQLite database. It has average operation speed, but long database initialization - it can take from 2 minutes to 5 hours depending on where the application is running)
    - postgres (stores data in PostgreSQL. This is the fastest operation mode, with faster initialization due to multithreading)
- It can run as a binary file with support for a configuration file in formats such as:
    - yaml
    - json
    - toml
    - environment variables during startup
    - in a Docker container receiving configuration through environment variables.

### Additional features:
    - Ability to save the user's last request (disabled by default)
    - Auto-checking the database for compliance with the INPX library
    - Option to enable a web page for book search (disabled by default)
    - Ability to search for books in a specific language - a two-letter code is added at the end of the query: ru, en, etc. By default, it is set to en, and if a value is set, there is no need to add the language. The bot can respond in the default language (supports en and ru). The value is set through --language in the console and LANGUAGE in the configuration file/env.
    - Restricting access to the bot. Three options are available: full access (both lists remain empty), access with a blacklist (only BLACKLIST is filled), restricted access through a whitelist (only WHITELIST is filled).

### Sample docker-compose for sqlite:
``` 
version: "3"
  services:
  gobooks:
    image: almadern/gobooks
    container_name: gobooks
    restart: always
    environment:
      DB: "sqlite"
      DB_DB: "/var/books.db"
      INPX: "/books/f.inpx"
      WEB: "false"
      TELEGRAM: "true"
      CONVERTER_PATH: "/opt/converter"
      TOKEN: "telegram token"
      BLACKLIST: "1111,2222"
      WHITELIST: "1111,2222"
      LANGUAGE: "en"
    volumes:
      - /books:/books:ro
    depends_on:
      - postgres`
```
### Sample docker-compose for postgres:
```
version: "3"
  
services:
  gobooks:
    image: almadern/gobooks
    container_name: gobooks
    restart: always
    environment:
      DB: "postgres"
      DB_DB: "gobooks"
      DB_USER: "postgres"
      DB_URL: "postgres"
      DB_PORT: "5432"
      DB_PASSWORD: "test"
      INPX: "/books/f.inpx"
      WEB: "false"
      TELEGRAM: "true"
      TOKEN: "telegram token"
      BLACKLIST: "1111,2222"
      WHITELIST: "1111,2222"
      LANGUAGE: "en"
    volumes:
      - /books:/books:ro
    depends_on:
      - postgres

  postgres:
    image: postgres
    container_name: postgres
    restart: always
    user: postgres
    environment:
      POSTGRES_PASSWORD: "test"
      POSTGRES_DB: "books"
      POSTGRES_USER: "postgres"
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data
volumes:
  pgdata:
```
### Sample docker-compose for file:
```
services:
  gobooks:
    image: almadern/gobooks 
    container_name: gobooks
    restart: always
    environment:
      DB: "file"
      INPX: "/books/f.inpx"
      WEB: "false"
      TELEGRAM: "true"
      TOKEN: "telegram token"
      BLACKLIST: "1111,2222"
      WHITELIST: "1111,2222"
      LANGUAGE: "en"
    volumes:
      - /books:/books:ro`
```
Build from sources (assuming that go is already installed on the system and is available)
```
GOARCH=<arch> GOOS=<os> go build -o gobooks main.go
```
Example of launching via a binary file without configuration:
```
./gobooks --Telegram true --Web false --token <mytoken> etc.(for full list use --help)
```
Example of launching via a binary file with configuration:
```
gobooks --config /path/to/config
```
Example config:
```
DB = "sqlite" # file , sqlite, postgres
DB_DB = "/var/test.db" # if file = nothing is need to be set if sqlite = path to db if postgres = database when will be stored books
DB_USER = "postgres" #only need for postgres
DB_URL = "localhost" # only need for postgres
DB_PORT = "5432" # only need for postgres
DB_PASSWORD = "test" #only need for postgres
Inpx = "/path/to/*.inpx" # path to inpx file(books need to be moved in same directory)
Port = "8080" # if web is set to true. On which port it will be working. Only http request
Token = "token from telegram" #if telegram is set to true. Token from botFather
Web =  false # true or false. Set the work of web page
Telegram = true # true of false. Set the work of Telegram bot
RESTORE = false # true of false. Set the function to restore request(if user don't download books need's to set /n to remove prev request)`
HTML_PATH = "/path/to/html/folder" #Path to html folder need's to be set to proper work. In docker container it's already set
BLACKLIST: "1111,2222"
WHITELIST: "1111,2222"
LANGUAGE: "en"
```
