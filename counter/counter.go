package counter

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
)

// CLI runs the go-counter command line app and returns its exit status.
func CLI(args []string) int {
	var app appEnv
	err := app.fromArgs(args)
	if err != nil {
		return 2
	}
	if err = app.run(); err != nil {
		fmt.Fprintf(os.Stderr, "Runtime error: %v\n", err)
		return 1
	}
	return 0
}

// appEnv represents parsed command line arguments
type appEnv struct {
	word            string
	total           int
	mu              sync.RWMutex
	wg              sync.WaitGroup
	reader          io.ReadCloser
	workersNum      int
	isCaseSensetive bool
}

// fromArgs parses command line arguments into appEnv struct
func (app *appEnv) fromArgs(args []string) error {
	fl := flag.NewFlagSet("counter", flag.ContinueOnError)
	fl.StringVar(&app.word, "w", "go", "word to count")
	fl.IntVar(&app.workersNum, "n", 5, "max number of concurrent workers")
	fl.BoolVar(&app.isCaseSensetive, "c", false, "is case sensetive count")

	if err := fl.Parse(args); err != nil {
		return err
	}

	if !app.isCaseSensetive {
		app.word = strings.ToLower(app.word)
		fmt.Println(app.word)
	}

	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		app.reader = os.Stdin
		return nil
	}

	file, err := os.Open(fl.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "can't open file %s: %v\n", fl.Arg(0), err)
		return err
	}
	app.reader = file

	return nil
}

func (app *appEnv) run() error {
	defer app.reader.Close()
	limit := make(chan struct{}, app.workersNum)

	scanner := bufio.NewScanner(app.reader)
	for scanner.Scan() {
		app.wg.Add(1)
		go app.countWords(scanner.Text(), limit)
	}

	app.wg.Wait()
	fmt.Printf("Total: %d\n", app.total)

	return nil
}

// countWords counts all words in url
func (app *appEnv) countWords(url string, limit chan struct{}) {
	limit <- struct{}{}
	defer app.wg.Done()

	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Can't get %s: %s\n", url, err.Error())
		return
	}
	defer resp.Body.Close()

	total := 0

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		if !app.isCaseSensetive {
			total += strings.Count(strings.ToLower(scanner.Text()), app.word)
			continue
		}
		total += strings.Count(scanner.Text(), app.word)
	}
	fmt.Printf("Count for %s: %d\n", url, total)

	app.mu.Lock()
	app.total += total
	app.mu.Unlock()

	<-limit
}
