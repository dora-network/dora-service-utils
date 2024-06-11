package migration_test

import (
	"embed"
	"github.com/dora-network/dora-service-utils/migration"
	"github.com/stretchr/testify/assert"
	"testing"
)

//go:embed test_parser/*.sql
var parserFS embed.FS

func TestParser_ParseVersion(t *testing.T) {
	tests := []struct {
		name     string
		fileName string
		want     int64
		wantErr  bool
	}{
		{
			name:     "valid sql migration file should be numbered sequence followed by .sql",
			fileName: "test_parser/001.sql",
			want:     1,
			wantErr:  false,
		},
		{
			name:     "non-compliant sql file names should generate an error",
			fileName: "test_parser/bad.sql",
			want:     0,
			wantErr:  true,
		},
		{
			name:     "even if they have a numbered sequence in the name",
			fileName: "test_parser/bad_000002.sql",
			want:     0,
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := migration.NewParser(parserFS, tt.fileName)
			got, err := p.ParseVersion()
			assert.Equal(t, tt.wantErr, err != nil, "Parser.ParseVersion() error = %v, wantErr %v", err, tt.wantErr)
			assert.Equal(t, tt.want, got, "Parser.ParseVersion() = %v, want %v", got, tt.want)
		})
	}
}

func TestParser_ParseMigrationFile(t *testing.T) {
	tests := []struct {
		name     string
		fileName string
		want     []string
		wantErr  bool
	}{
		{
			name:     "comments should be ignored by the test_parser",
			fileName: "test_parser/comments_only.sql",
			want:     []string{},
			wantErr:  false,
		},
		{
			name:     "sql blocks with an end marker should be optional",
			fileName: "test_parser/mixed_start_end.sql",
			want: []string{
				`select 1;`,
				`select 2;`,
				`select 3;`,
				`select 4;`,
				`select 5;`,
			},
			wantErr: false,
		},
		{
			name:     "the last block in the file should always be captured, even if there is no end marker",
			fileName: "test_parser/start_no_end.sql",
			want: []string{
				`select 1;`,
				`select 2;`,
				`select 3;`,
				`select 4;`,
				`select 5;`,
			},
			wantErr: false,
		},
		{
			name:     "should return all the statements in the file",
			fileName: "test_parser/001.sql",
			want: []string{
				`select 1;`,
			},
			wantErr: false,
		},
		{
			name:     "even if it has multiple statements in the file",
			fileName: "test_parser/002.sql",
			want: []string{
				`select 1;`,
				`select 2;`,
				`select 3;`,
				`select 4;`,
				`select 5;`,
			},
			wantErr: false,
		},
		{
			name:     "markers should be case-insensitive",
			fileName: "test_parser/case_insensitive.sql",
			want: []string{
				`select 1;`,
				`select 2;`,
				`select 3;`,
				`select 4;`,
				`select 5;`,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := migration.NewParser(parserFS, tt.fileName)
			got, err := p.ParseMigrationFile()
			assert.Equal(t, tt.wantErr, err != nil, "Parser.ParseMigrationFile() error = %v, wantErr %v", err, tt.wantErr)
			assert.Equal(t, tt.want, got, "Parser.ParseMigrationFile() = %v, want %v", got, tt.want)
		})
	}
}
