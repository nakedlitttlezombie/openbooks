package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/evan-buss/openbooks/core"
	"github.com/evan-buss/openbooks/irc"
)

func terminalMenu(irc *irc.Conn) {
	fmt.Print("\ns)search\ng)et book\nd)one\n~> ")

	// Trim user input so we don't send 2 messages
	clean := func(message string) string { return strings.Trim(message, "\r\n") }

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = clean(input)

	switch input {
	case "s":
		fmt.Print("@search ")
		message, _ := reader.ReadString('\n')
		core.SearchBook(irc, clean(message))
		fmt.Println("\nSent search request.")
	case "g":
		fmt.Print("Download String: ")
		message, _ := reader.ReadString('\n')
		core.DownloadBook(irc, clean(message))
		fmt.Println("\nSent download request.")
	case "d":
		fmt.Println("Disconnecting.")
		irc.Disconnect()
		os.Exit(0)
	default:
		fmt.Println("Invalid Selection.")
		terminalMenu(irc)
	}
}

func fullHandler(config Config) core.EventHandler {
	handler := core.EventHandler{}

	handler[core.BadServer] = func(text string) {
		config.badServerHandler(text)
		terminalMenu(config.irc)
	}
	handler[core.BookResult] = func(text string) {
		config.downloadHandler(text)
		terminalMenu(config.irc)
	}
	handler[core.SearchResult] = func(text string) {
		config.searchHandler(text)
		terminalMenu(config.irc)
	}
	handler[core.SearchAccepted] = config.searchAcceptedHandler
	handler[core.NoResults] = func(text string) {
		config.noResultsHandler(text)
		terminalMenu(config.irc)
	}
	handler[core.MatchesFound] = config.matchesFoundHandler
	handler[core.Ping] = config.pingHandler

	return handler
}
