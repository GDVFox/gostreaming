package common

// Command команда
type Command string

// CommandHelper структура для инициализации и выполнения команды.
type CommandHelper interface {
	// Run запускает выполнение комнады.
	Run()
	// Init инициализирует состояние команды с помощью аргументов командной строки.
	Init(args []string) error
	// Печатает сообщение с описанием использования команды.
	PrintHelp()
}
