package ngxapi

import "testing"

func TestSlugify(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"Bank Statement": "bank-statement",
		"  Invoice #42 ":  "invoice-42",
		"":                "",
	}

	for input, want := range cases {
		if got := slugify(input); got != want {
			t.Fatalf("slugify(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestStripHTML(t *testing.T) {
	t.Parallel()

	got := stripHTML("<p>Hello <b>world</b></p>")
	if got != "Hello world" {
		t.Fatalf("stripHTML() = %q", got)
	}
}

func TestMapJobStatus(t *testing.T) {
	t.Parallel()

	if mapJobStatus("completed") != "SUCCESS" {
		t.Fatal("expected SUCCESS for completed")
	}
	if mapJobStatus("pending") != "PENDING" {
		t.Fatal("expected PENDING for pending")
	}
}

func TestCreatedDateOnly(t *testing.T) {
	t.Parallel()

	if got := createdDateOnly("2024-05-01 12:00:00.000Z"); got != "2024-05-01" {
		t.Fatalf("createdDateOnly() = %q", got)
	}
}

func TestFormatNgxDateTime(t *testing.T) {
	t.Parallel()

	got := formatNgxDateTime("2026-06-13 11:56:31.599Z")
	want := "2026-06-13T11:56:31.599Z"
	if got != want {
		t.Fatalf("formatNgxDateTime() = %q, want %q", got, want)
	}
}

func TestFormatNgxCreatedDate(t *testing.T) {
	t.Parallel()

	if got := formatNgxCreatedDate("2024-03-15"); got != "2024-03-15T00:00:00Z" {
		t.Fatalf("formatNgxCreatedDate(date) = %q", got)
	}
	if got := formatNgxCreatedDate("2025-12-19 00:00:00.000Z"); got != "2025-12-19T00:00:00.000Z" {
		t.Fatalf("formatNgxCreatedDate(datetime) = %q", got)
	}
}
