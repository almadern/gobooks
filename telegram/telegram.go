package telegram

import (
	"bytes"
	"context"
	"fmt"
	"gobook/arguments"
	db "gobook/database"
	"gobook/paginator"
	"gobook/zipextract"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strconv"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// Start telegram bot
func BotStart() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	opts := []bot.Option{
		bot.WithDefaultHandler(defaultHandler),
	}

	b, err := bot.New(arguments.Config.Token, opts...)
	if err != nil {
		panic(err)
	}
	b.RegisterHandler(bot.HandlerTypeMessageText, "/help", bot.MatchTypeExact, helpHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/author", bot.MatchTypeContains, authorHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/series", bot.MatchTypeContains, seriesHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/title", bot.MatchTypeContains, titleHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/set", bot.MatchTypeContains, setHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/y", bot.MatchTypeContains, showagainUIHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/n", bot.MatchTypeContains, wipeUIHandler)
	b.Start(ctx)
}

// request to work with client
func helpHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	defer recoverFromPanic()
	switch arguments.Config.LANGUAGE {
	case "ru":
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Я принимаю 3 поля для поиска: автор(author), название(title), серия(series)\nЗапросы ведутся по умолчанию на русском(указывать ничего не надо), либо язык можно указать последним в формате 2 букв(ru, en, du и т.д.).\nКниги выдаются в формате epub и fb2\nЗапрос ведется в формате: /<поле по котору происходит поиск> что ищем(пример: /author Кинг ru)",
		})
	default:
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "I accept 3 search fields: author, title, series\nRequests are conducted by default in English (no need to specify anything), or the language can be specified last in 2 letter format (ru, en, du etc.) .\nBooks are issued in epub and fb2 format\nThe request is made in the form those: /<field in which the search is performed> what we are looking for (example: /author King ru)",
		})
	}
}

func authorHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
    // this if needs to handle error if user trying to use paginator created in previus session and bot fell into panic
	if update.CallbackQuery != nil {
		switch arguments.Config.LANGUAGE {
		case "ru":
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.CallbackQuery.Message.Chat.ID,
				Text: "Бот был обновлен. Данный список больше не может быть обработан.\nПожалуйста повторите свой запрос.",
			})
		default:
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.CallbackQuery.Message.Chat.ID,
				Text:   "Bot was updated. Write you request again",
			})
		}
	}
	defer recoverFromPanic()
    // check if user have access to use bot
	if accessCheck(update) {
		if !arguments.Config.Restore {
		    // remove old message
			wipeUIHandler(ctx, b, update)
		}
		request := strings.Split(update.Message.Text, " ")
		spell := spellCheck("/author", ctx, b, update)
		req := requestCheck(ctx, b, update)
		request, lang := langrequest(request)
		log.Printf("User: %s (FirstName: %s LastName: %s ID:%d)send request to Author: %s", update.Message.Chat.Username, update.Message.Chat.FirstName, update.Message.From.LastName, update.Message.Chat.ID, request)
		if spell && req {
			user := createPaginator(ctx, b, update)
			var output []db.Extract
			switch arguments.Config.DB {
			case "file":
				output = user.FileFind(arguments.Config.Inpx, "Author", request, lang)
			case "postgres":
				output = user.PostgresFind("Author", request, lang)
			case "sqlite":
				output = user.SQLiteFind("Author", request, lang)
			}
			if len(output) >= 1 {
                // create ui page in telegram chat with finded books
				data := transformToPaginator(output)
				opts := []paginator.Option{
					paginator.PerPage(3),
					paginator.WithCloseButton("Close"),
				}
				p := paginator.New(data, opts...)
				p.Show(ctx, b, strconv.Itoa(int(update.Message.Chat.ID)))
				sendReminderToSet(ctx, b, update)
				user.Callback = p.CallbackHandlerID
				switch arguments.Config.DB {
				case "postgres":
					user.PostgresWriteUserInfo()
				case "file":
					user.FileWrtieUserInfo()
				case "sqlite":
					user.SQLiteWriteUserInfo()
				}
			} else {
				switch arguments.Config.LANGUAGE {
				case "ru":
					b.SendMessage(ctx, &bot.SendMessageParams{
						ChatID: update.Message.Chat.ID,
						Text: "Ничего не удалось найти.\nПожалуйста проверьте правильность написанного запроса или попробуйте ввести другое значение для поиска\n(Серия, Автор или Название могут быть записаны по разному).\nНа поиск не влияет в каком регистре написан запрос\n(С заглавными и/или строчными)",
					})
				default:
					b.SendMessage(ctx, &bot.SendMessageParams{
						ChatID: update.Message.Chat.ID,
						Text:   "Bot was updated. Write you request again\n",
					})
				}
			}
		}
	}
}

func seriesHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery != nil {
		switch arguments.Config.LANGUAGE {
		case "ru":
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.CallbackQuery.Message.Chat.ID,
				Text: "Бот был обновлен. Данный список больше не может быть обработан.\nПожалуйста повторите свой запрос.",
			})
		default:
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.CallbackQuery.Message.Chat.ID,
				Text: "Бот был обновлен. Данный список больше не может быть обработан.\nПожалуйста повторите свой запрос.",
			})
		}
	}
	defer recoverFromPanic()
	if accessCheck(update) {
		if !arguments.Config.Restore {
			wipeUIHandler(ctx, b, update)
		}
		request := strings.Split(update.Message.Text, " ")
		spell := spellCheck("/series", ctx, b, update)
		req := requestCheck(ctx, b, update)
		request, lang := langrequest(request)
		log.Printf("User: %s (FirstName: %s LastName: %s ID:%d)send request to Series: %s", update.Message.Chat.Username, update.Message.Chat.FirstName, update.Message.From.LastName, update.Message.Chat.ID, request)
		if spell && req {
			user := createPaginator(ctx, b, update)
			var output []db.Extract
			switch arguments.Config.DB {
			case "file":
				output = user.FileFind(arguments.Config.Inpx, "Series", request, lang)
			case "postgres":
				output = user.PostgresFind("Series", request, lang)
			case "sqlite":
				output = user.SQLiteFind("Series", request, lang)
			}
			if len(output) >= 1 {
				data := transformToPaginator(output)
				opts := []paginator.Option{
					paginator.PerPage(3),
					paginator.WithCloseButton("Close"),
				}
				p := paginator.New(data, opts...)
				user.Callback = p.CallbackHandlerID
				p.Show(ctx, b, strconv.Itoa(int(update.Message.Chat.ID)))
				sendReminderToSet(ctx, b, update)
				switch arguments.Config.DB {
				case "postgres":
					user.PostgresWriteUserInfo()
				case "file":
					user.FileWrtieUserInfo()
				case "sqlite":
					user.SQLiteWriteUserInfo()
				}
			} else {
				switch arguments.Config.LANGUAGE {
				case "ru":
					b.SendMessage(ctx, &bot.SendMessageParams{
						ChatID: update.Message.Chat.ID,
						Text: "Ничего не удалось найти.\nПожалуйста проверьте правильность написанного запроса или попробуйте ввести другое значение для поиска\n(Серия, Автор или Название могут быть записаны по разному).\nНа поиск не влияет в каком регистре написан запрос\n(С заглавными и/или строчными)",
					})
				default:
					b.SendMessage(ctx, &bot.SendMessageParams{
						ChatID: update.Message.Chat.ID,
						Text: "Ничего не удалось найти.\nПожалуйста проверьте правильность написанного запроса или попробуйте ввести другое значение для поиска\n(Серия, Автор или Название могут быть записаны по разному).\nНа поиск не влияет в каком регистре написан запрос\n(С заглавными и/или строчными)",
					})
				}
			}
		}
	}
}

func titleHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery != nil {
		switch arguments.Config.LANGUAGE {
		case "ru":
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.CallbackQuery.Message.Chat.ID,
				Text: "Бот был обновлен. Данный список больше не может быть обработан.\nПожалуйста повторите свой запрос.",
			})
		default:
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.CallbackQuery.Message.Chat.ID,
				Text:   "Bot was updated. Write you request again",
			})
		}
	}
	defer recoverFromPanic()
	if accessCheck(update) {
		if !arguments.Config.Restore {
			wipeUIHandler(ctx, b, update)
		}
		request := strings.Split(update.Message.Text, " ")
		spell := spellCheck("/title", ctx, b, update)
		req := requestCheck(ctx, b, update)
		request, lang := langrequest(request)
		log.Printf("User: %s (FirstName: %s LastName: %s ID:%d)send request to Title: %s", update.Message.Chat.Username, update.Message.Chat.FirstName, update.Message.From.LastName, update.Message.Chat.ID, request)
		if spell && req {
			user := createPaginator(ctx, b, update)
			var output []db.Extract
			switch arguments.Config.DB {
			case "file":
				output = user.FileFind(arguments.Config.Inpx, "Title", request, lang)
			case "postgres":
				output = user.PostgresFind("Title", request, lang)
			case "sqlite":
				output = user.SQLiteFind("Title", request, lang)
			}
			if len(output) >= 1 {
				data := transformToPaginator(output)
				opts := []paginator.Option{
					paginator.PerPage(3),
					paginator.WithCloseButton("Close"),
				}
				p := paginator.New(data, opts...)
				p.Show(ctx, b, strconv.Itoa(int(update.Message.Chat.ID)))
				sendReminderToSet(ctx, b, update)
				user.Callback = p.CallbackHandlerID
				switch arguments.Config.DB {
				case "postgres":
					user.PostgresWriteUserInfo()
				case "file":
					user.FileWrtieUserInfo()
				case "sqlite":
					user.SQLiteWriteUserInfo()
				}
			} else {
				switch arguments.Config.LANGUAGE {
				case "ru":
					b.SendMessage(ctx, &bot.SendMessageParams{
						ChatID: update.Message.Chat.ID,
						Text: "Ничего не удалось найти.\nПожалуйста проверьте правильность написанного запроса или попробуйте ввести другое значение для поиска\n(Серия, Автор или Название могут быть записаны по разному).\nНа поиск не влияет в каком регистре написан запрос\n(С заглавными и/или строчными)",
					})
				default:
					b.SendMessage(ctx, &bot.SendMessageParams{
						ChatID: update.Message.Chat.ID,
						Text: "Ничего не удалось найти.\nПожалуйста проверьте правильность написанного запроса или попробуйте ввести другое значение для поиска\n(Серия, Автор или Название могут быть записаны по разному).\nНа поиск не влияет в каком регистре написан запрос\n(С заглавными и/или строчными)",
					})
				}
			}
		}
	}
}

func defaultHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery != nil {
		switch arguments.Config.LANGUAGE {
		case "ru":
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.CallbackQuery.Message.Chat.ID,
				Text: "Бот был обновлен. Данный список больше не может быть обработан.\nПожалуйста повторите свой запрос.",
			})
		default:
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.CallbackQuery.Message.Chat.ID,
				Text:   "Bot was updated. Write you request again",
			})
		}
		defer recoverFromPanic()
	}
	if accessCheck(update) {
		switch arguments.Config.LANGUAGE {
		case "ru":
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text: "Привет!\nЭто стартовая страница. Если нужна помощь напишите:\n\t/help - чтобы получить справочную информацию\nили\n/author \n/series\n/title запрос - чтобы начать искать книги",
			})
		default:
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "Hello!\nThis is start page. If you need help type:\n\t/help - to see help massage\nor\n/author \n/series\n/genres\n/title and what you try to find - to start finding books",
			})
		}
	}
}

// request to work with chosen books
func setHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.CallbackQuery != nil {
		switch arguments.Config.LANGUAGE {
		case "ru":
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.CallbackQuery.Message.Chat.ID,
				Text: "Бот был обновлен. Данный список больше не может быть обработан.\nПожалуйста повторите свой запрос.",
			})
		default:
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.CallbackQuery.Message.Chat.ID,
				Text:   "Bot was updated. Write you request again",
			})
		}
	}
	defer recoverFromPanic()
	if accessCheck(update) {
		request := strings.Split(update.Message.Text, " ")
		if len(request) == 1 {
			switch arguments.Config.LANGUAGE {
			case "ru":
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text: "В вашем запросе нету книг для обработки\nПожалуйта напишите запрос в формтае\n/set номера книг которые нужны и формат вывода(по умолчанию epub)(Пример: /set 4 5 10 25 epub) ",
				})
			default:
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "Your message don't have what you to choose\nPlease type your request in format\n /author again",
				})
			}
		}
		user := createPaginator(ctx, b, update)
		if arguments.Config.Restore {
			switch arguments.Config.DB {
			case "postgres":
				user.PostgresWipeUserInfo()
			case "file":
				user.FileWipeUserInfo()
			case "sqlite":
				user.SQLiteWipeUserInfo()
			}
		}
		var decodedData []db.Extract
		switch arguments.Config.DB {
		case "file":
			decodedData = user.FileReadRequest()
		case "postgres":
			decodedData = user.PostgresReadRequest()
		case "sqlite":
			decodedData = user.SQLiteReadRequest()
		}
		formatoutput := []string{"fb2", "epub"}
		format := "epub"
		// check if the last element is string to set output format
		if _, err := strconv.Atoi(request[len(request)-1]); err != nil {
			for _, i := range formatoutput {
				if i == request[len(request)-1] {
					format = i
					request = request[:len(request)-1]
				}
			}
		}

		for _, choose := range request[1:] {
			value, err := strconv.Atoi(choose)
			if err != nil {
				fmt.Println(err)
				switch arguments.Config.LANGUAGE {
				case "ru":
					b.SendMessage(ctx, &bot.SendMessageParams{
						ChatID: update.Message.Chat.ID,
						Text: "Вы вписали не цифру: " + choose + "\nЭтот запрос не будет обработан\nПожалуйста указывайте только цифры",
					})
				default:
					b.SendMessage(ctx, &bot.SendMessageParams{
						ChatID: update.Message.Chat.ID,
						Text:   "You write not a int: " + choose + "\nit can't be done\nPlease write only numbers list",
					})
				}
				continue
			}
			if value > len(decodedData) {
				switch arguments.Config.LANGUAGE {
				case "ru":
					b.SendMessage(ctx, &bot.SendMessageParams{
						ChatID: update.Message.Chat.ID,
						Text: "Вы выбрали значение не из списка: " + choose + " Этот запрос не будет обработан",
					})
				default:
					b.SendMessage(ctx, &bot.SendMessageParams{
						ChatID: update.Message.Chat.ID,
						Text:   "You choose not in list " + choose + " it can't be done",
					})
				}
				continue
			}
			output := decodedData[value-1]
			title := ""
			name := output.Title
			nametofb2 := fmt.Sprintf("%v", user.Chat) + "_" + fmt.Sprintf("%v", value) + ".fb2"
			nameepub := fmt.Sprintf("%v", user.Chat) + "_" + fmt.Sprintf("%v", value) + ".epub"
			if len(output.SeriesNums) > 2 {
				title = output.SeriesNums + "-" + name
			} else {
				title = name
			}
			err = zipextract.Open(output.Dir, output.FileInArchive, nametofb2)
			var fileData []byte
			if err == nil {
				if format == "epub" {
					var cmd *exec.Cmd
					switch os := runtime.GOOS; os {
					case "darwin":
						cmd = exec.Command(arguments.Config.CONVERTER_PATH+"/fb2c_mac", "convert", nametofb2)
					case "windows":
						cmd = exec.Command(arguments.Config.CONVERTER_PATH+"\fb2c.exe", "convert", nametofb2)
					case "linux":
						cmd = exec.Command(arguments.Config.CONVERTER_PATH+"/fb2c_linux", "convert", nametofb2)
					}
					err = cmd.Run()
					if err != nil {
						log.Printf("When exec converter error: %v", err)
						switch arguments.Config.LANGUAGE {
						case "ru":
							b.SendMessage(ctx, &bot.SendMessageParams{
								ChatID: update.Message.Chat.ID,
								Text:   "При конвертировании книги: " + name + " Прозошла ошибка.\nОбратитесь к администратору, если такая ошибка повторится вновь",
							})
						default:
							b.SendMessage(ctx, &bot.SendMessageParams{
								ChatID: update.Message.Chat.ID,
								Text:   "When converting book: " + name + " error was thrown.\nPlease contact admin, if this will continue",
							})
						}
					}
					fileData, err = os.ReadFile(nameepub)
					title += ".epub"
					if err != nil {
						log.Printf("Can't open file after convertation %v", err)
					}
				} else if format == "fb2" {
					fileData, err = os.ReadFile(nametofb2)
					title += ".fb2"
					if err != nil {
						log.Printf("Can't open file %v", err)
					}
				}
			}

			params := &bot.SendDocumentParams{
				ChatID:   update.Message.Chat.ID,
				Document: &models.InputFileUpload{Filename: title, Data: bytes.NewReader(fileData)},
			}
			b.SendDocument(ctx, params)

			err = os.Remove(nametofb2)
			if err != nil {
				log.Printf("Error when delete file %v", err)
			}
			if format == "epub" {
				err = os.Remove(nameepub)
				if err != nil {
					log.Printf("Error when delete file %v", err)
				}
			}
		}
		// Delete created files
		switch arguments.Config.DB {
		case "postgres":
			if arguments.Config.Restore {
				user.PostgresWipeUserInfo()
			}
			user.PostgresWipeRequest()
		case "file":
			user.FileWipeRequestandUserInfo()
		case "sqlite":
			if arguments.Config.Restore {
				user.SQLiteWipeUserInfo()
			}
			user.SQLiteWipeRequest()
		}
	}
	log.Printf("User: %s (FirstName: %s LastName: %s ID:%d)send request, but will ignored", update.Message.Chat.Username, update.Message.Chat.FirstName, update.Message.From.LastName, update.Message.Chat.ID)
}
// function shows user his last search. If option RESTORE set to default it's dosen't work and user will receive a message about it depricated status
func showagainUIHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	defer recoverFromPanic()
	if accessCheck(update) {
		if arguments.Config.Restore {
			user := createPaginator(ctx, b, update)

			var output []db.Extract
			switch arguments.Config.DB {
			case "postgres":
				output = user.PostgresReadRequest()
			case "file":
				output = user.FileReadRequest()
			case "sqlite":
				output = user.PostgresReadRequest()
			}
			data := transformToPaginator(output)
			opts := []paginator.Option{
				paginator.PerPage(3),
				paginator.WithCloseButton("Close"),
			}
			p := paginator.New(data, opts...)
			p.Show(ctx, b, strconv.Itoa(int(update.Message.Chat.ID)))
			user.Callback = p.CallbackHandlerID
			switch arguments.Config.DB {
			case "postgres":
				user.PostgresWriteUserInfo()
			case "file":
				user.FileWrtieUserInfo()
			case "sqlite":
				user.SQLiteWriteUserInfo()
			}
		} else {
			switch arguments.Config.LANGUAGE {
			case "ru":
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "Функция отключена администратором",
				})
			default:
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "Function disabled by admin",
				})
			}
		}
	}
}
// cleanup user messages and handlers(paginator) from previus search
func wipeUIHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	defer recoverFromPanic()
	if accessCheck(update) {
		user := createPaginator(ctx, b, update)
		switch arguments.Config.DB {
		case "postgres":
			output := user.PostgresReadUserInfo()
			for _, i := range output {
				// remove Handler(paginator) from chat
				b.UnregisterHandler(i.Callback)
				b.DeleteMessage(ctx, &bot.DeleteMessageParams{
					ChatID:    i.Chat,
					MessageID: i.Message,
				})
				// remove message from user
				b.DeleteMessage(ctx, &bot.DeleteMessageParams{
					ChatID:    i.Chat,
					MessageID: i.Message - 1,
				})
				// remove reminder how to set books
				b.DeleteMessage(ctx, &bot.DeleteMessageParams{
					ChatID:    i.Chat,
					MessageID: update.Message.ID + 1,
				})
			}
			user.PostgresWipeUserInfo()
			user.PostgresWipeRequest()
		case "file":
			output := user.FileReadUserInfo()
			for _, i := range output {
				b.UnregisterHandler(i.Callback)
				b.DeleteMessage(ctx, &bot.DeleteMessageParams{
					ChatID:    i.Chat,
					MessageID: i.Message,
				})
			}
			user.FileWipeRequestandUserInfo()
		case "sqlite":
			output := user.SQLiteReadUserInfo()
			for _, i := range output {
				b.UnregisterHandler(i.Callback)
				b.DeleteMessage(ctx, &bot.DeleteMessageParams{
					ChatID:    i.Chat,
					MessageID: i.Message,
				})
				b.DeleteMessage(ctx, &bot.DeleteMessageParams{
					ChatID:    i.Chat,
					MessageID: i.Message - 1,
				})
				b.DeleteMessage(ctx, &bot.DeleteMessageParams{
					ChatID:    i.Chat,
					MessageID: i.Message + 1,
				})
			}
			user.SQLiteWipeUserInfo()
			user.SQLiteWipeRequest()
		}
	}
}

// check if user allowed to access Bot
func accessCheck(update *models.Update) bool {
	userID := update.Message.Chat.ID
	// all allowed
	if arguments.Config.AllAccess && arguments.Config.StrictAcc {
		return true
	}
	if arguments.Config.AllAccess {
		for _, i := range arguments.Config.BlackList {
			// check if user in BlackList
			if i == userID {
				log.Printf("User: %s (FirstName: %s LastName: %s ID:%d)send request, but will ignored", update.Message.Chat.Username, update.Message.Chat.FirstName, update.Message.From.LastName, update.Message.Chat.ID)
				return false
			}
		}
		return true
	} else if arguments.Config.StrictAcc {
		for _, i := range arguments.Config.Whitelist {
			// check if user in Whitelist
			if i == userID {
				return true
			}
		}
		log.Printf("User: %s (FirstName: %s LastName: %s ID:%d)send request, but will ignored", update.Message.Chat.Username, update.Message.Chat.FirstName, update.Message.From.LastName, update.Message.Chat.ID)
		return false
	}
	// In case some bug will be return false
	return false
}

// check request from client
func spellCheck(handler string, ctx context.Context, b *bot.Bot, update *models.Update) bool {
	if update.CallbackQuery != nil {
		switch arguments.Config.LANGUAGE {
		case "ru":
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.CallbackQuery.Message.Chat.ID,
				Text:   "Бот был обновлен. Данный список больше не может быть обработан.\nПожалуйста повторите свой запрос.",
			})
		default:
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.CallbackQuery.Message.Chat.ID,
				Text:   "The bot has been updated. This list can no longer be processed.\nPlease repeat your request.",
			})
		}
		return false
	}
	request := strings.Split(update.Message.Text, " ")
	if len(request) == 1 {
		switch arguments.Config.LANGUAGE {
		case "ru":
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "В сообщении нету значения для поиска.\nПожалуйста напишите свой запрос в формате\n" + handler + " ваш зарос",
			})
		default:
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "There is no search value in the message.\nPlease write your request in the format\n" + handler + " your request",
			})
		}
		return false
	}
	if request[0] != handler {
		switch arguments.Config.LANGUAGE {
		case "ru":
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "В сообщении опечатка\nПожалуйста повторите запрос",
			})
		default:
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "There is a typo in the message\nPlease repeat your request",
			})
		}
		return false
	}
	return true
}

// check for previus request if RESTORE is true in another case it will always sent true
func requestCheck(ctx context.Context, b *bot.Bot, update *models.Update) bool {
	ok := true
	if arguments.Config.Restore {
		switch arguments.Config.DB {
		case "postgres":
			ok = db.PostgresRequestCheck(update.Message.Chat.ID)
		case "file":
			ok = db.FileRequestCheck(update.Message.Chat.ID)
		case "sqlite":
			ok = db.SQLiteRequestCheck(update.Message.Chat.ID)
		}
		if !ok {
			switch arguments.Config.LANGUAGE {
			case "ru":
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "У вас есть незавершенный поиск. Хотите продолжить работать с ним?\nПожалуйста напечатайте или нажмите на нужную команду:\n/y - если хотите продолжить\n/n - чтобы удалить запрос.\nПРЕДУПРЕЖДЕНИЕ: Новые запросы не могут быть обработаны, пока не будет выбран один из вариантов",
				})
			default:
				b.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "You have an unfinished search. Do you want to continue working with it?\nPlease type or click on the desired command:\n/y - if you want to continue\n/n - to delete the request.\nWARNING: New requests cannot be processed until one of the options is selected",
				})
			}
		}
	}
	return ok
}

// check if user set the language in request. If note will be added default
func langrequest(request []string) ([]string, string) {
	lang := []string{"ab", "aa", "af", "sq", "am", "ar", "hy", "as", "ay", "az", "ba", "eu", "bn", "dz", "bh", "bi", "br", "bg", "my", "be", "km", "ca", "zh", "co", "hr", "cs", "da", "nl", "en", "eo", "et", "fo", "fj", "fi", "fr", "fy", "gd", "gl", "ka", "de", "el", "kl", "gn", "gu", "ha", "iw", "hi", "hu", "is", "in", "ia", "ie", "ik", "ga", "it", "ja", "jw", "kn", "ks", "kk", "rw", "ky", "rn", "ko", "ku", "lo", "la", "lv", "ln", "lt", "mk", "mg", "ms", "ml", "mt", "mi", "mr", "mo", "mn", "na", "ne", "no", "oc", "or", "om", "ps", "fa", "pl", "pt", "pa", "qu", "rm", "ro", "ru", "sm", "sg", "sa", "sr", "sh", "st", "tn", "sn", "sd", "si", "ss", "sk", "sl", "so", "es", "su", "sw", "sv", "tl", "tg", "ta", "tt", "te", "th", "bo", "ti", "to", "ts", "tr", "tk", "tw", "uk", "ur", "uz", "vi", "vo", "cy", "wo", "xh", "ji", "yo", "zu"}
    // set variable to assume language in request
	reqlang := request[len(request)-1]
	if reqlang == arguments.Config.LANGUAGE {
		return request[1:], arguments.Config.LANGUAGE
	} else if len(reqlang) == 2{
		for i := range lang {
			if strings.ToLower(reqlang) == lang[i] {
				return request[1 : len(request)-1], reqlang
			}
		}
	}
	return request[1:], arguments.Config.LANGUAGE
}
// reminder how set request for get finded books
func sendReminderToSet(ctx context.Context, b *bot.Bot, update *models.Update) {
	switch arguments.Config.LANGUAGE {
	case "ru":
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Список книг которые удалось найти\nПожалуйта напишите запрос в формтае\n/set номера книг которые нужны и формат вывода fb2 или epub(по умолчанию epub, указывать для него ничего не надо)\n(Пример: /set 4 5 10 25 fb2) и нажмите кнопку Close для получения книг",
		})
	default:
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "List of books that were found\nPlease write a request in the format\n/set book numbers that are needed and the output format fb2 or epub (by default epub, you don’t need to specify anything for it)\n(Example: /set 4 5 10 25 fb2) and click the Close button to get the books",
		})
	}
}
// transform finded books info into paginator valid format to send
func transformToPaginator(output []db.Extract) []string {
	var data []string
	// array of Escape characters
	charsToChange := []string{"[", "]", "-", ".", "(", ")", "/", "_", "+", "!"}
	for v, i := range output {
		authors := i.Authors
		ganres := i.Ganres
		title := i.Title
		series := i.Series
		seriesnms := i.SeriesNums
		for _, char := range charsToChange {
			title = strings.ReplaceAll(title, char, "\\"+char)
			series = strings.ReplaceAll(series, char, "")
			authors = strings.ReplaceAll(authors, char, "")
			ganres = strings.ReplaceAll(ganres, char, " ")
			seriesnms = strings.ReplaceAll(seriesnms, char, " ")
		}
		data = append(data, fmt.Sprintf("*%d* Authors: %s\n Title: %s\nGanres: %s\nSeries: %s\nSeriesNum: %s", v+1, authors, title, ganres, series, seriesnms))
	}
	return data
}

// function need's to conver info from request to db data
func createPaginator(_ context.Context, _ *bot.Bot, update *models.Update) db.Pagenator {
	user := db.Pagenator{
		Chat:      update.Message.Chat.ID,
		Message:   update.Message.ID + 1,
		FirstName: update.Message.Chat.FirstName,
		LastName:  update.Message.Chat.LastName,
		Username:  update.Message.Chat.Username,
	}
	return user
}

// recover function in case user addressed to UI that was called before restart
func recoverFromPanic() {
	if r := recover(); r != nil {
		log.Println("RecoverFromPanic after user call UI after restart. Err:", r)
	}
}
