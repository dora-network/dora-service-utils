package migration

import (
	"bufio"
	"fmt"
	"io/fs"
	"strconv"
	"strings"
)

type Parser struct {
	files    fs.FS
	filename string
}

func NewParser(files fs.FS, filename string) *Parser {
	return &Parser{
		files:    files,
		filename: filename,
	}
}

func (p *Parser) ParseMigrationFile() ([]string, error) {
	return parseMigrationFilename(p.files, p.filename)
}

func parseMigrationFilename(migrationFS fs.FS, migrationFile string) ([]string, error) {
	f, err := migrationFS.Open(migrationFile)
	if err != nil {
		return nil, err
	}

	defer func() {
		err := f.Close()
		if err != nil {
			panic(err)
		}
	}()

	statements := make([]string, 0)

	scanner := bufio.NewScanner(f)
	statementBuilder := new(strings.Builder)
	multiLineComment := false
	isComment := false

	addAndReset := func() {
		statements = append(statements, statementBuilder.String())
		statementBuilder.Reset()
		multiLineComment = false
		isComment = false
	}

	for scanner.Scan() {
		line := scanner.Text()
		// Skip empty line
		if len(strings.TrimSpace(line)) == 0 {
			continue
		}
		// Migration statements should start with -- +Start
		if isStatementStart(line) {
			if statementBuilder.Len() > 0 {
				addAndReset()
			}
			continue
		}

		// And end with -- +End
		if isStatementEnd(line) {
			addAndReset()
			continue
		}

		// Comments should be ignored
		if isComment, multiLineComment = checkComment(line, multiLineComment); isComment {
			continue
		}

		// Everything else makes up the statement
		statementBuilder.WriteString(line)
	}

	if statementBuilder.Len() > 0 {
		addAndReset()
	}

	return statements, nil
}

func isStatementStart(line string) bool {
	line = strings.ToLower(line)
	return strings.HasPrefix(line, "--") && strings.Contains(line, "+start")
}

func isStatementEnd(line string) bool {
	return strings.HasPrefix(line, "--") && strings.Contains(line, "+end")
}

func checkComment(line string, multiline bool) (bool, bool) {
	if (multiline && !strings.Contains(line, "/*") && !strings.Contains(line, "*/")) ||
		(!multiline && strings.Contains(line, "/*") && !strings.Contains(line, "*/")) {
		return true, true
	}

	if multiline && strings.Contains(line, "*/") {
		return true, false
	}

	if strings.HasPrefix(line, "--") || strings.HasPrefix(line, "/*") {
		return true, false
	}

	return false, false
}

func (p *Parser) ParseVersion() (int64, error) {
	f, err := fs.Stat(p.files, p.filename)
	if err != nil {
		return 0, err
	}
	vs := strings.SplitN(f.Name(), ".", 2)
	if len(vs) != 2 && vs[1] != "sql" {
		return 0, fmt.Errorf("invalid migration file, migration files should be a numbered sequence followed by .sql, e.g. 0000010.sql: %s", p.filename)
	}

	return strconv.ParseInt(vs[0], 10, 64)
}
