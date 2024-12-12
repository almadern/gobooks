<h1>
Телеграм бот для выдачи книг в формате fb2 и epub из личной библиотеке в формате inpx запакованных в zip архивы в формате fb2 и epub
</h1>

### Бот работает с домашней inpx библиотекой и архивами с книгами. 
- Может хранить и обрабатывать данные в 3 разных формата: 
    - file(без БД, при каждом запросе проверяется все библиотека.Самый медленный вариант работы)
    - sqlite(работает и хранит данные в sqlite библиотеке. Средняя скорость работы, и долгая инициализация базы данных - может занимать от 2 минут до 5 часов(в зависмости от того где запущено приложение)),        
    - postgres(хранит данные в PostgreSQL. Самый быстрый вариант работы. инициализация происходит быстрее за счет многопоточности)
- Может работать как в виде бинарного файла c поддержкой конфигурационного файла формата: 
    - yaml
    - json
    - toml
    - получать через переменные при запуске
    - в докер контейнере получая конфигурацию через переменные окружение.

### Из дополнительных фич:
    - Возможность сохранять последний запрос пользователя(по дефолту выключено)
    - Авто проверка базы данных на соответсвие с inpx библиотекой
    - Имеется возможность включения web страницы для поиска книг(по дефолту выключено) 
    - Возможность поиска книг на определенном языке - добавляется двухсимвольный формат в конце запроса: ru,en и т.д. по дефолту en, если значение установлено то добавлять язык не надо. Бот может отвечать на выбранному по умолчанию языку(поддерживается en и ru). Значение задается через --language в консоли и LANGUAGE в конфигурационном файле\env
    - Ограничение доступа к боту. Доступно 3 варианта: полный доступ(Оба списка остаются пустыми), доступ с blacklist(заполняется только BLACKLIST), ограниченный доступ через whitelist(заолняется только WHITELIST)

### Пример docker-compose для sqlite:
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
### Пример docker-compose для postgres:
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
### Пример docker-compose для file:
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
Собрать из исходников(предполагается, что go уже установлен в системе и доступен)
```
GOARCH=<arch> GOOS=<os> go build -o gobooks main.go
```
Пример запуска через бинарный файл без конфигурации:
```
./gobooks --Telegram true --Web false --token <mytoken> etc.(for full list use --help)
```
Пример запуска через бинарный файл c конфигурацией:
```
gobooks --config /path/to/config
```
Пример конфига:
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
