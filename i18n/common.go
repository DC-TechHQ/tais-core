package i18n

// Common i18n codes — used by errors/ and response/ packages.
// Every service can add its own codes via Register() in internal/i18n/.
const (
	// Success messages
	MsgSuccess = "MsgSuccess"
	MsgCreated = "MsgCreated"
	MsgUpdated = "MsgUpdated"
	MsgDeleted = "MsgDeleted"

	// Common errors
	ErrInternal           = "ErrInternal"
	ErrInvalidData        = "ErrInvalidData"
	ErrNotFound           = "ErrNotFound"
	ErrAlreadyExists      = "ErrAlreadyExists"
	ErrForeignKey         = "ErrForeignKey"
	ErrUnauthorized       = "ErrUnauthorized"
	ErrForbidden          = "ErrForbidden"
	ErrInvalidToken       = "ErrInvalidToken"
	ErrTokenExpired       = "ErrTokenExpired"
	ErrUserBlocked        = "ErrUserBlocked"
	ErrInvalidCredentials = "ErrInvalidCredentials"
	ErrDeadlock           = "ErrDeadlock"
)

func init() {
	Register(map[string]map[string]string{
		MsgSuccess: {
			LangTJ: "Муваффақ",
			LangRU: "Успешно",
			LangEN: "Success",
		},
		MsgCreated: {
			LangTJ: "Бомуваффақият эҷод шуд",
			LangRU: "Успешно создано",
			LangEN: "Successfully created",
		},
		MsgUpdated: {
			LangTJ: "Бомуваффақият навсозӣ шуд",
			LangRU: "Успешно обновлено",
			LangEN: "Successfully updated",
		},
		MsgDeleted: {
			LangTJ: "Бомуваффақият нест шуд",
			LangRU: "Успешно удалено",
			LangEN: "Successfully deleted",
		},

		ErrInternal: {
			LangTJ: "Хатои дохилии сервер",
			LangRU: "Внутренняя ошибка сервера",
			LangEN: "Internal server error",
		},
		ErrInvalidData: {
			LangTJ: "Маълумоти нодуруст",
			LangRU: "Некорректные данные",
			LangEN: "Invalid data",
		},
		ErrNotFound: {
			LangTJ: "Сабт ёфт нашуд",
			LangRU: "Запись не найдена",
			LangEN: "Record not found",
		},
		ErrAlreadyExists: {
			LangTJ: "Ин сабт аллакай мавҷуд аст",
			LangRU: "Запись уже существует",
			LangEN: "Record already exists",
		},
		ErrForeignKey: {
			LangTJ: "Вобастагии маълумот вайрон шудааст",
			LangRU: "Нарушена связь данных",
			LangEN: "Data reference violation",
		},
		ErrUnauthorized: {
			LangTJ: "Ташхис нагузаштааст",
			LangRU: "Не авторизован",
			LangEN: "Unauthorized",
		},
		ErrForbidden: {
			LangTJ: "Дастрасӣ манъ аст",
			LangRU: "Доступ запрещён",
			LangEN: "Access forbidden",
		},
		ErrInvalidToken: {
			LangTJ: "Токен нодуруст аст",
			LangRU: "Неверный токен",
			LangEN: "Invalid token",
		},
		ErrTokenExpired: {
			LangTJ: "Мӯҳлати токен гузаштааст",
			LangRU: "Срок действия токена истёк",
			LangEN: "Token has expired",
		},
		ErrUserBlocked: {
			LangTJ: "Корбар бастааст",
			LangRU: "Пользователь заблокирован",
			LangEN: "User is blocked",
		},
		ErrInvalidCredentials: {
			LangTJ: "Логин ё парол нодуруст аст",
			LangRU: "Неверный логин или пароль",
			LangEN: "Invalid login or password",
		},
		ErrDeadlock: {
			LangTJ: "Низоъи транзаксия. Дубора кӯшиш кунед",
			LangRU: "Конфликт транзакций. Повторите попытку",
			LangEN: "Transaction conflict. Please retry",
		},
	})
}
